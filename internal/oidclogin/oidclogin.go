// Package oidclogin implements the usermgmt OAuth2Provider contract against
// the stack's OpenID Connect identity provider (Pocket ID): authorization-code
// + PKCE S256 with an OIDC nonce, RS256 ID-token verification, a userinfo
// fallback for Pocket ID's claim-sparse tokens, and an in-app group allowlist
// as the second gate behind the IdP-side client-group link.
//
// Design rules (the webphone sso doctrine, live since 2026-10-23):
//   - discovery is LAZY: a provider that is unreachable must fail the login
//     attempt with an actionable error, never the process boot (the lens must
//     keep serving read-only pages while the IdP is down; the passkey lane
//     keeps working either way);
//   - the client secret is read from its runtime file on every exchange, so
//     rotation via the secrets pipeline needs no restart;
//   - the group allowlist is fail-closed: a token whose claims carry none of
//     the allowed groups is denied before any user matching happens;
//   - OIDC tokens never leave this package: the caller receives the
//     normalized user info JSON, nothing else.
package oidclogin

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json/v2"
	"log/slog"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	errorfamily "github.com/larsartmann/go-error-family"
	"golang.org/x/oauth2"
)

// discoveryTimeout bounds the lazy provider lookup: the IdP normally answers
// in milliseconds; a hung one must not hang a login request for minutes.
const discoveryTimeout = 10 * time.Second

// providerTTL bounds the positive discovery cache: a long-running process
// re-discovers after this window so IdP endpoint or key rotation is picked up
// without a restart. Failures are never cached — an outage costs a login
// attempt, never the boot.
const providerTTL = 15 * time.Minute

// exchangeTimeout bounds every token/userinfo round trip to the IdP.
const exchangeTimeout = 15 * time.Second

// DefaultProviderName is the provider name this package answers to: it rides
// the usermgmt OAuth routes as {provider} and the login button's href.
const DefaultProviderName = "pocketid"

// Config is the resolved OIDC configuration. Issuer, ClientID,
// ClientSecretFile, and RedirectURL are required; everything else carries a
// working default.
type Config struct {
	// Issuer is the IdP's discovery base URL (e.g. https://auth.example.org).
	Issuer string
	// ClientID is the IdP-registered client (the lens registers as a
	// confidential client with PKCE).
	ClientID string
	// ClientSecretFile is the runtime secret file, read per exchange.
	ClientSecretFile string
	// RedirectURL is the registered callback: https://<host>/auth/oauth/pocketid/callback.
	RedirectURL string
	// Scopes defaults to openid, email, profile (the group claim rides the
	// ID token with these — the same shape the webphone runs against the
	// same IdP).
	Scopes []string
	// AllowedGroups is the in-app group allowlist. Non-empty, a login must
	// carry at least one of these groups in its claims; empty disables the
	// check (not this deployment's shape — the host pins the group).
	AllowedGroups []string
	// Name is the provider name this instance answers to (routes, button).
	// Empty selects DefaultProviderName.
	Name string
}

func (c Config) providerName() string {
	if c.Name == "" {
		return DefaultProviderName
	}

	return c.Name
}

// userInfo is the normalized identity payload returned to the usermgmt
// service. The JSON tags are the contract (kept identical to usermgmt's
// OAuth2UserInfo and cqrs-htmx's oauth2 module — the Sollbruchstelle seam).
type userInfo struct {
	Subject           string `json:"subject"`
	Email             string `json:"email"`
	EmailVerified     bool   `json:"email_verified"`
	DisplayName       string `json:"display_name"`
	PreferredUsername string `json:"preferred_username,omitempty"`
}

// idClaims is the narrow claim set this login consumes. Groups is admission
// evidence for the allowlist check only — it is never copied onto the user.
type idClaims struct {
	Email             string   `json:"email"`
	EmailVerified     *bool    `json:"email_verified"`
	PreferredUsername string   `json:"preferred_username"`
	Groups            []string `json:"groups"`
}

// Provider performs the authorization-code + PKCE dance against one IdP and
// implements the identitymodel.OAuth2Provider interface (plus its additive
// nonce capability). Safe for concurrent use.
type Provider struct {
	cfg  Config
	http *http.Client

	mu        sync.Mutex
	oidcProv  *oidc.Provider
	fetchedAt time.Time
	inflight  chan providerResult
}

// providerResult is the single-flight handoff: every goroutine that arrives
// while one discovery is running shares its outcome.
type providerResult struct {
	p   *oidc.Provider
	err error
}

// New validates the static shape and returns a provider. No network is
// touched: an IdP outage is a login failure, not a boot failure.
func New(cfg Config) (*Provider, error) {
	switch {
	case cfg.Issuer == "":
		return nil, errorfamily.NewRejection("lens.oidc.issuer", "-auth-oidc-issuer is required")
	case cfg.ClientID == "":
		return nil, errorfamily.NewRejection("lens.oidc.client_id", "-auth-oidc-client-id is required")
	case cfg.ClientSecretFile == "":
		return nil, errorfamily.NewRejection("lens.oidc.client_secret_file",
			"-auth-oidc-secret-file is required (the OIDC client secret, read per login)")
	case cfg.RedirectURL == "":
		return nil, errorfamily.NewRejection("lens.oidc.redirect_url", "-auth-oidc-redirect is required "+
			"(the registered callback: https://<host>/auth/oauth/"+DefaultProviderName+"/callback)")
	}

	if len(cfg.Scopes) == 0 {
		cfg.Scopes = []string{oidc.ScopeOpenID, "email", "profile"}
	}

	return &Provider{ //nolint:exhaustruct // zero-value cache fields are the lazy state
		cfg:  cfg,
		http: &http.Client{Timeout: exchangeTimeout},
	}, nil
}

// Names implements the usermgmt providerNamer capability so the loginpage
// panel auto-detects the sign-in button from the service.
func (p *Provider) Names() []string { return []string{p.cfg.providerName()} }

// BeginLogin implements the nonce-less baseline of the OAuth2Provider
// contract. The usermgmt service prefers the nonce-capable pair when the
// state store supports it (its default does), so this pair is the fallback.
func (p *Provider) BeginLogin(ctx context.Context, providerName, state string) (string, string, error) {
	redirectURL, verifier, _, err := p.begin(ctx, providerName, state, false)

	return redirectURL, verifier, err
}

// BeginLoginWithNonce builds one authorization URL carrying fresh state,
// PKCE S256, and an OIDC nonce; the service stores all three and hands the
// nonce back at FinishLoginWithNonce.
func (p *Provider) BeginLoginWithNonce(
	ctx context.Context, providerName, state string,
) (string, string, string, error) {
	return p.begin(ctx, providerName, state, true)
}

func (p *Provider) begin(
	ctx context.Context, providerName, state string, withNonce bool,
) (string, string, string, error) {
	if err := p.checkName(providerName); err != nil {
		return "", "", "", err
	}

	provider, err := p.providerFor(ctx)
	if err != nil {
		return "", "", "", err
	}

	verifier, err := randomToken()
	if err != nil {
		return "", "", "", err
	}

	oc := p.oauthConfig(provider.Endpoint(), "")
	nonce := ""
	opts := []oauth2.AuthCodeOption{
		oauth2.SetAuthURLParam("code_challenge", pkceChallenge(verifier)),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	}

	if withNonce {
		nonce, err = randomToken()
		if err != nil {
			return "", "", "", err
		}

		opts = append(opts, oauth2.SetAuthURLParam("nonce", nonce))
	}

	return oc.AuthCodeURL(state, opts...), verifier, nonce, nil
}

// FinishLogin implements the nonce-less baseline of the contract.
func (p *Provider) FinishLogin(
	ctx context.Context, providerName, code, pkceVerifier string,
) ([]byte, error) {
	return p.finish(ctx, providerName, code, pkceVerifier, "")
}

// FinishLoginWithNonce exchanges the authorization code, verifies the ID
// token (signature, issuer, audience, expiry, nonce), resolves the claims
// (with the userinfo fallback), enforces the group allowlist, and returns the
// normalized user info JSON. Every failure is a classification-carrying
// error, actionable in the journal.
func (p *Provider) FinishLoginWithNonce(
	ctx context.Context, providerName, code, pkceVerifier, nonce string,
) ([]byte, error) {
	return p.finish(ctx, providerName, code, pkceVerifier, nonce)
}

func (p *Provider) finish(
	ctx context.Context, providerName, code, pkceVerifier, nonce string,
) ([]byte, error) {
	if err := p.checkName(providerName); err != nil {
		return nil, err
	}

	if code == "" {
		return nil, errorfamily.NewRejection("lens.oidc.code", "the IdP redirected back without an authorization code")
	}

	if pkceVerifier == "" {
		return nil, errorfamily.NewRejection("lens.oidc.verifier",
			"the PKCE verifier is missing — restart the sign-in from this browser")
	}

	provider, err := p.providerFor(ctx)
	if err != nil {
		return nil, err
	}

	secret, err := p.clientSecret()
	if err != nil {
		return nil, err
	}

	token, idToken, err := p.exchangeAndVerify(ctx, provider, secret, code, pkceVerifier, nonce)
	if err != nil {
		return nil, err
	}

	claims, err := p.claims(ctx, provider, token, idToken)
	if err != nil {
		return nil, err
	}

	return p.gateAndMarshal(idToken, claims)
}

// exchangeAndVerify runs the token half of the dance: the PKCE code
// exchange, the id_token presence + signature/audience/expiry
// verification, and the nonce binding. Every failure names the exact wire
// step that failed.
func (p *Provider) exchangeAndVerify(
	ctx context.Context, provider *oidc.Provider, secret, code, pkceVerifier, nonce string,
) (*oauth2.Token, *oidc.IDToken, error) {
	oc := p.oauthConfig(provider.Endpoint(), secret)

	token, err := oc.Exchange(ctx, code, oauth2.SetAuthURLParam("code_verifier", pkceVerifier))
	if err != nil {
		return nil, nil, errorfamily.WrapRejection(err, "lens.oidc.exchange",
			"exchange the authorization code with the IdP (check the registered callback URL and the client secret)")
	}

	rawIDToken, _ := token.Extra("id_token").(string)
	if rawIDToken == "" {
		return nil, nil, errorfamily.NewRejection("lens.oidc.id_token",
			"the IdP's token response carried no id_token (is the openid scope enabled?)")
	}

	idToken, err := provider.Verifier(&oidc.Config{ClientID: p.cfg.ClientID}).Verify(ctx, rawIDToken)
	if err != nil {
		return nil, nil, errorfamily.WrapRejection(err, "lens.oidc.id_token.verify", "verify the IdP's ID token")
	}

	if nonce != "" && idToken.Nonce != nonce {
		return nil, nil, errorfamily.NewRejection("lens.oidc.nonce_mismatch",
			"the ID token nonce does not match this sign-in attempt — restart the sign-in from this browser")
	}

	return token, idToken, nil
}

// gateAndMarshal is the policy half: the verified-email gate, the group
// allowlist (fail-closed both ways, see groupAllowed), and the normalized
// user-info JSON the identity layer persists.
func (p *Provider) gateAndMarshal(idToken *oidc.IDToken, claims idClaims) ([]byte, error) {
	slog.Info("lens: oidc login claims",
		"email", claims.Email,
		"username", claims.PreferredUsername,
		"groups", strings.Join(claims.Groups, ","))

	if claims.EmailVerified != nil && !*claims.EmailVerified {
		return nil, errorfamily.NewRejection("lens.oidc.email_unverified",
			"the IdP reports the email as unverified — verify the address in the IdP before signing in")
	}

	if !p.groupAllowed(claims.Groups) {
		return nil, errorfamily.NewRejection(
			"lens.oidc.groups_denied",
			"the IdP account carries none of the allowed groups — add the account to the required access group in the IdP",
		).
			WithContext("email", claims.Email).
			WithContext("allowed_groups", strings.Join(p.cfg.AllowedGroups, ", "))
	}

	return json.Marshal(userInfo{
		Subject:           idToken.Subject,
		Email:             claims.Email,
		EmailVerified:     claims.EmailVerified != nil && *claims.EmailVerified,
		DisplayName:       claims.PreferredUsername,
		PreferredUsername: claims.PreferredUsername,
	})
}

// groupAllowed is the in-app group allowlist: fail-closed both ways — a token
// with no groups claim never passes a non-empty allowlist, and the check is
// off entirely when the operator configures none.
func (p *Provider) groupAllowed(groups []string) bool {
	if len(p.cfg.AllowedGroups) == 0 {
		return true
	}

	for _, group := range groups {
		if slices.Contains(p.cfg.AllowedGroups, group) {
			return true
		}
	}

	return false
}

// claims resolves the identity payload. The ID token carries the claims on a
// well-behaved IdP; Pocket ID's access tokens famously carry no profile
// claims and its ID token may omit them for some client shapes, so a missing
// email falls back to the userinfo endpoint (the same merge doctrine
// Stalwart's native OIDC path and the webphone use). Groups ride whichever
// document carries them — the allowlist consults the merged set.
func (p *Provider) claims(
	ctx context.Context, provider *oidc.Provider, token *oauth2.Token, idToken *oidc.IDToken,
) (idClaims, error) {
	var claims idClaims
	if err := idToken.Claims(&claims); err != nil {
		return idClaims{}, errorfamily.WrapRejection(err, "lens.oidc.claims", "parse the ID token claims")
	}

	if claims.Email != "" {
		return claims, nil
	}

	info, err := provider.UserInfo(ctx, oauth2.StaticTokenSource(token))
	if err != nil {
		return idClaims{}, errorfamily.WrapRejection(err, "lens.oidc.userinfo",
			"fetch the userinfo claims (the ID token carried no email)")
	}

	if err := info.Claims(&claims); err != nil {
		return idClaims{}, errorfamily.WrapRejection(err, "lens.oidc.userinfo.claims", "parse the userinfo claims")
	}

	if claims.Email == "" {
		return idClaims{}, errorfamily.NewRejection("lens.oidc.email_missing",
			"the IdP returned no email claim (check the requested scopes include email)")
	}

	return claims, nil
}

// clientSecret reads the runtime secret file per exchange (rotation takes
// effect on the next login). Empty or whitespace-only fails closed; a
// trailing newline is trimmed so the write-side newline class can never
// authenticate as secret+newline.
func (p *Provider) clientSecret() (string, error) {
	raw, err := os.ReadFile(p.cfg.ClientSecretFile)
	if err != nil {
		return "", errorfamily.WrapRejection(err, "lens.oidc.client_secret.read",
			"read the OIDC client secret file (check the secrets pipeline and the unit's LoadCredential)")
	}

	secret := strings.TrimSpace(string(raw))
	if secret == "" {
		return "", errorfamily.NewRejection("lens.oidc.client_secret.empty", "the OIDC client secret file is empty")
	}

	return secret, nil
}

// providerFor lazily discovers the provider and caches the success. A failed
// discovery is NOT cached: an IdP that was down for one login is retried on
// the next.
func (p *Provider) providerFor(ctx context.Context) (*oidc.Provider, error) {
	p.mu.Lock()
	if p.oidcProv != nil && time.Since(p.fetchedAt) < providerTTL {
		provider := p.oidcProv
		p.mu.Unlock()

		return provider, nil
	}

	if p.inflight != nil {
		ch := p.inflight
		p.mu.Unlock()

		res := <-ch

		return res.p, res.err
	}

	ch := make(chan providerResult, 1)
	p.inflight = ch
	p.mu.Unlock()

	dctx, cancel := context.WithTimeout(ctx, discoveryTimeout)
	provider, err := oidc.NewProvider(dctx, p.cfg.Issuer)

	cancel()

	p.mu.Lock()

	p.inflight = nil
	if err == nil {
		p.oidcProv = provider
		p.fetchedAt = time.Now()
	}
	p.mu.Unlock()

	ch <- providerResult{p: provider, err: err}

	close(ch)

	if err != nil {
		return nil, errorfamily.WrapRejection(err, "lens.oidc.discovery",
			"discover the OIDC provider (check -auth-oidc-issuer and that the IdP is reachable)")
	}

	return provider, nil
}

func (p *Provider) oauthConfig(endpoint oauth2.Endpoint, secret string) *oauth2.Config {
	endpoint.AuthStyle = oauth2.AuthStyleInHeader

	return &oauth2.Config{ //nolint:exhaustruct // IDTokenPII and RedirectURL-independent fields stay zero
		ClientID:     p.cfg.ClientID,
		ClientSecret: secret,
		Endpoint:     endpoint,
		RedirectURL:  p.cfg.RedirectURL,
		Scopes:       p.cfg.Scopes,
	}
}

func (p *Provider) checkName(providerName string) error {
	if providerName != p.cfg.providerName() {
		return errorfamily.NewRejection("lens.oidc.provider_name",
			"this OIDC provider answers only as "+p.cfg.providerName()).
			WithContext("requested", providerName)
	}

	return nil
}

func randomToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", errorfamily.WrapInfrastructure(err, "lens.oidc.random_token",
			"crypto/rand entropy read failed (kernel CSPRNG unavailable)")
	}

	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func pkceChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))

	return base64.RawURLEncoding.EncodeToString(sum[:])
}
