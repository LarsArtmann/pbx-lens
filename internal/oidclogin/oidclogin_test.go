package oidclogin

import (
	"context"
	"encoding/base64"
	"encoding/json/v2"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/larsartmann/pbx-lens/internal/oidclogin/oidctest"
)

// testConfig builds a Config whose issuer is the stub and whose secret file
// lives in a temp dir.
func testConfig(t *testing.T, idp *oidctest.StubIdP, mutate ...func(*Config)) Config {
	t.Helper()

	secretFile := filepath.Join(t.TempDir(), "client-secret")
	if err := os.WriteFile(secretFile, []byte("s3cr3t"), 0o600); err != nil {
		t.Fatalf("write secret file: %v", err)
	}

	cfg := Config{
		Issuer:           idp.URL(),
		ClientID:         "lens-test",
		ClientSecretFile: secretFile,
		RedirectURL:      "https://lens.example.org/auth/oauth/pocketid/callback",
		AllowedGroups:    []string{"pbx-history"},
	}
	for _, m := range mutate {
		m(&cfg)
	}

	return cfg
}

func TestNewValidatesShapeWithoutNetwork(t *testing.T) {
	t.Parallel()

	// An unreachable issuer must be fine at construction: discovery is lazy.
	unreachable := Config{
		Issuer:           "http://127.0.0.1:1",
		ClientID:         "c",
		ClientSecretFile: "/dev/null",
		RedirectURL:      "https://lens.example.org/cb",
	}
	if _, err := New(unreachable); err != nil {
		t.Fatalf("New with unreachable issuer must not touch the network: %v", err)
	}

	missing := []Config{
		{ClientID: "c", ClientSecretFile: "/dev/null", RedirectURL: "https://x/cb"},
		{Issuer: "https://x", ClientSecretFile: "/dev/null", RedirectURL: "https://x/cb"},
		{Issuer: "https://x", ClientID: "c", RedirectURL: "https://x/cb"},
		{Issuer: "https://x", ClientID: "c", ClientSecretFile: "/dev/null"},
	}
	for i, cfg := range missing {
		if _, err := New(cfg); err == nil {
			t.Errorf("case %d: New must reject the incomplete shape", i)
		}
	}
}

func TestBeginLoginBuildsPKCEAuthorizeURL(t *testing.T) {
	t.Parallel()

	idp := newStubIdP(t, map[string]any{"email": "lars@example.org"})
	provider, err := New(testConfig(t, idp))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	redirectURL, verifier, nonce, err := provider.BeginLoginWithNonce(context.Background(), "pocketid", "state-1")
	if err != nil {
		t.Fatalf("BeginLoginWithNonce: %v", err)
	}

	if !strings.HasPrefix(redirectURL, idp.URL()+"/authorize?") {
		t.Fatalf("authorize URL must target the IdP: %s", redirectURL)
	}

	q := parseQuery(t, redirectURL)
	for name, want := range map[string]string{
		"client_id":             "lens-test",
		"response_type":         "code",
		"scope":                 "openid email profile",
		"state":                 "state-1",
		"nonce":                 nonce,
		"code_challenge_method": "S256",
	} {
		if q.Get(name) != want {
			t.Errorf("%s = %q, want %q", name, q.Get(name), want)
		}
	}

	if nonce == "" {
		t.Fatal("nonce must be non-empty on the nonce-capable path")
	}

	wantChallenge := base64.RawURLEncoding.EncodeToString(sha256Sum(verifier))
	if q.Get("code_challenge") != wantChallenge {
		t.Errorf("code_challenge = %q, want the S256 of the verifier", q.Get("code_challenge"))
	}

	if got := q.Get("code_verifier"); got != "" {
		t.Errorf("the verifier must never ride the URL (got %q)", got)
	}
}

func TestFinishLoginWithNonceHappyPath(t *testing.T) {
	t.Parallel()

	idp := newStubIdP(t, map[string]any{
		"sub":                "user-1",
		"email":              "lars@example.org",
		"email_verified":     true,
		"preferred_username": "lars",
		"groups":             []string{"pbx-history", "pbx-phone"},
	})
	provider, err := New(testConfig(t, idp))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	info := runLogin(t, provider, idp)

	if info.Subject != "user-1" {
		t.Errorf("subject = %q, want user-1", info.Subject)
	}

	if info.Email != "lars@example.org" {
		t.Errorf("email = %q, want lars@example.org", info.Email)
	}

	if !info.EmailVerified {
		t.Error("email_verified must surface true")
	}

	if info.DisplayName != "lars" {
		t.Errorf("display_name = %q, want lars", info.DisplayName)
	}

	if idp.Exchanges() != 1 {
		t.Errorf("exchanges = %d, want 1", idp.Exchanges())
	}

	if idp.LastForm().Get("code_verifier") == "" {
		t.Error("the exchange must carry the PKCE verifier")
	}
}

func TestFinishLoginGroupGateFailsClosed(t *testing.T) {
	t.Parallel()

	cases := map[string]map[string]any{
		"wrong group": {"sub": "u", "email": "lars@example.org", "email_verified": true,
			"groups": []string{"pbx-phone"}},
		"no groups claim": {"sub": "u", "email": "lars@example.org", "email_verified": true},
	}
	for name, claims := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			idp := newStubIdP(t, claims)
			provider, err := New(testConfig(t, idp))
			if err != nil {
				t.Fatalf("New: %v", err)
			}

			_, verifier, nonce, err := provider.BeginLoginWithNonce(context.Background(), "pocketid", "s")
			if err != nil {
				t.Fatalf("begin: %v", err)
			}

			idp.SetNonce(nonce)

			_, err = provider.FinishLoginWithNonce(context.Background(), "pocketid", "code-1", verifier, nonce)
			if err == nil {
				t.Fatal("a login without an allowed group must be denied")
			}

			if !strings.Contains(err.Error(), "group") {
				t.Errorf("the denial must name the group gate, got: %v", err)
			}
		})
	}
}

func TestFinishLoginUnverifiedEmailDenied(t *testing.T) {
	t.Parallel()

	idp := newStubIdP(t, map[string]any{
		"sub": "user-1", "email": "lars@example.org", "email_verified": false,
		"groups": []string{"pbx-history"},
	})
	provider, err := New(testConfig(t, idp))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, verifier, nonce, err := provider.BeginLoginWithNonce(context.Background(), "pocketid", "s")
	if err != nil {
		t.Fatalf("begin: %v", err)
	}

	idp.SetNonce(nonce)

	if _, err := provider.FinishLoginWithNonce(context.Background(), "pocketid", "code-1", verifier, nonce); err == nil {
		t.Fatal("an unverified email must be denied")
	}
}

func TestFinishLoginNonceMismatchDenied(t *testing.T) {
	t.Parallel()

	idp := newStubIdP(t, map[string]any{
		"sub": "user-1", "email": "lars@example.org", "email_verified": true,
		"groups": []string{"pbx-history"},
	})
	provider, err := New(testConfig(t, idp))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	idp.SetNonce("nonce-in-token")

	if _, err := provider.FinishLoginWithNonce(context.Background(), "pocketid", "code-1", "verifier", "different-nonce"); err == nil {
		t.Fatal("a mismatched nonce must be denied")
	}
}

func TestFinishLoginWrongProviderNameRejected(t *testing.T) {
	t.Parallel()

	idp := newStubIdP(t, map[string]any{"email": "lars@example.org"})
	provider, err := New(testConfig(t, idp))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if _, _, _, err := provider.BeginLoginWithNonce(context.Background(), "google", "s"); err == nil {
		t.Fatal("a foreign provider name must be rejected at begin")
	}

	if _, err := provider.FinishLoginWithNonce(context.Background(), "google", "c", "v", "n"); err == nil {
		t.Fatal("a foreign provider name must be rejected at finish")
	}
}

func TestClientSecretFailClosed(t *testing.T) {
	t.Parallel()

	idp := newStubIdP(t, map[string]any{
		"sub": "u", "email": "lars@example.org", "email_verified": true, "groups": []string{"pbx-history"},
	})

	t.Run("missing file", func(t *testing.T) {
		t.Parallel()

		cfg := testConfig(t, idp, func(c *Config) { c.ClientSecretFile = filepath.Join(t.TempDir(), "absent") })
		provider, err := New(cfg)
		if err != nil {
			t.Fatalf("New: %v", err)
		}

		if _, err := provider.clientSecret(); err == nil {
			t.Fatal("a missing secret file must fail closed")
		}
	})

	t.Run("empty file", func(t *testing.T) {
		t.Parallel()

		cfg := testConfig(t, idp, func(c *Config) { c.ClientSecretFile = filepath.Join(t.TempDir(), "empty") })
		if err := os.WriteFile(cfg.ClientSecretFile, nil, 0o600); err != nil {
			t.Fatalf("write: %v", err)
		}

		provider, err := New(cfg)
		if err != nil {
			t.Fatalf("New: %v", err)
		}

		if _, err := provider.clientSecret(); err == nil {
			t.Fatal("an empty secret file must fail closed")
		}
	})

	t.Run("trailing newline trimmed", func(t *testing.T) {
		t.Parallel()

		cfg := testConfig(t, idp, func(c *Config) { c.ClientSecretFile = filepath.Join(t.TempDir(), "nl") })
		if err := os.WriteFile(cfg.ClientSecretFile, []byte("s3cr3t\n"), 0o600); err != nil {
			t.Fatalf("write: %v", err)
		}

		provider, err := New(cfg)
		if err != nil {
			t.Fatalf("New: %v", err)
		}

		secret, err := provider.clientSecret()
		if err != nil {
			t.Fatalf("clientSecret: %v", err)
		}

		if secret != "s3cr3t" {
			t.Errorf("secret = %q, want the newline-trimmed value", secret)
		}
	})
}

func TestDiscoveryFailureNotCachedAndNeverAtBoot(t *testing.T) {
	t.Parallel()

	idp := newStubIdP(t, map[string]any{"email": "lars@example.org"})
	provider, err := New(testConfig(t, idp))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// Force a discovery failure by pointing at a closed listener.
	provider.cfg.Issuer = "http://127.0.0.1:1"
	if _, err := provider.providerFor(context.Background()); err == nil {
		t.Fatal("a dead issuer must fail the discovery")
	}

	// Point back at the live stub: the next attempt must succeed (the
	// failure was not cached).
	provider.cfg.Issuer = idp.URL()
	if _, err := provider.providerFor(context.Background()); err != nil {
		t.Fatalf("a recovered IdP must be discovered on the next login: %v", err)
	}
}

func TestUserInfoFallbackWhenIDTokenLacksEmail(t *testing.T) {
	t.Parallel()

	// Pocket ID's claim-sparse token shape: no email in the ID token; the
	// userinfo endpoint carries it (and the groups).
	idp := newStubIdP(t, map[string]any{
		"sub": "user-2", "preferred_username": "lars", "groups": []string{"pbx-history"},
	})
	idp.SetUserinfoClaims(map[string]any{
		"sub": "user-2", "preferred_username": "lars", "groups": []string{"pbx-history"},
		"email": "lars@example.org", "email_verified": true,
	})

	provider, err := New(testConfig(t, idp))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	info := runLogin(t, provider, idp)

	if info.Email != "lars@example.org" {
		t.Errorf("email = %q, want the userinfo-fallback value", info.Email)
	}

	if !info.EmailVerified {
		t.Error("email_verified must surface from the userinfo document")
	}
}

// runLogin drives one full begin→finish cycle through the stub IdP and
// returns the normalized user info. The stub's nonce is set from the
// begin step so the issued ID token carries it.
func runLogin(t *testing.T, provider *Provider, idp *oidctest.StubIdP) userInfo {
	t.Helper()

	_, verifier, nonce, err := provider.BeginLoginWithNonce(context.Background(), "pocketid", "state-run")
	if err != nil {
		t.Fatalf("begin: %v", err)
	}

	idp.SetNonce(nonce)

	raw, err := provider.FinishLoginWithNonce(context.Background(), "pocketid", "code-1", verifier, nonce)
	if err != nil {
		t.Fatalf("finish: %v", err)
	}

	var info userInfo
	if err := json.Unmarshal(raw, &info); err != nil {
		t.Fatalf("unmarshal user info: %v", err)
	}

	return info
}
