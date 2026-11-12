package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/larsartmann/pbx-lens/internal/oidclogin/oidctest"
)

// The OIDC sign-in lane (2026-11-12 nginx-exit Track A): the Pocket ID
// dance rides the usermgmt OAuth routes under /auth/oauth/pocketid/*, the
// first group-passing login provisions the single operator, and the minted
// session opens the gate exactly like a passkey login. These e2e tests boot
// the REAL identity composition (SQL stores, service, panel, routes, gate)
// against the stub IdP — only the browser is faked.

// openOIDCTestIdentity boots the identity context with the OIDC lane
// pointed at the stub IdP.
func openOIDCTestIdentity(t *testing.T, idp *oidctest.StubIdP) *Identity {
	t.Helper()

	secretFile := filepath.Join(t.TempDir(), "client-secret")
	if err := os.WriteFile(secretFile, []byte("s3cr3t"), 0o600); err != nil {
		t.Fatalf("write secret: %v", err)
	}

	identity, err := OpenIdentity(context.Background(), IdentityConfig{
		DSN:       filepath.Join(t.TempDir(), "identity.db"),
		RPID:      "localhost",
		RPOrigins: []string{"http://localhost:8300"},
		OIDC: &OIDCConfig{
			Issuer:           idp.URL(),
			ClientID:         "lens-test",
			ClientSecretFile: secretFile,
			RedirectURL:      "https://lens.example.org/auth/oauth/pocketid/callback",
			AllowedGroups:    []string{"pbx-history"},
		},
	})
	if err != nil {
		t.Fatalf("OpenIdentity with OIDC: %v", err)
	}

	t.Cleanup(func() {
		if cerr := identity.Close(); cerr != nil {
			t.Errorf("identity close: %v", cerr)
		}
	})

	return identity
}

func TestOIDCLoginMintsSessionAndOpensGate(t *testing.T) {
	t.Parallel()

	idp := oidctest.NewStub(map[string]any{
		"sub": "user-1", "email": "lars@example.org", "email_verified": true,
		"preferred_username": "lars", "groups": []string{"pbx-history"},
	})
	t.Cleanup(idp.Close)

	identity := openOIDCTestIdentity(t, idp)
	handler := identityHandler(t, identity)

	// The login page renders the Pocket ID button.
	page := httptest.NewRequest(http.MethodGet, "/login", nil)
	page.Header.Set("Accept", "text/html")

	pageRecorder := httptest.NewRecorder()
	handler.ServeHTTP(pageRecorder, page)

	if pageRecorder.Code != http.StatusOK {
		t.Fatalf("GET /login status = %d, want 200", pageRecorder.Code)
	}

	if !strings.Contains(pageRecorder.Body.String(), "/auth/oauth/pocketid/begin") {
		t.Fatal("the login page must render the Pocket ID sign-in button")
	}

	// Begin: 302 into the IdP's authorize endpoint.
	session := driveOIDCLogin(t, handler, idp)

	// The session cookie opens a gated route.
	gated := httptest.NewRequest(http.MethodGet, "/", nil)
	gated.AddCookie(session)
	gated.Header.Set("Accept", "text/html")

	gatedRecorder := httptest.NewRecorder()
	handler.ServeHTTP(gatedRecorder, gated)

	if gatedRecorder.Code == http.StatusUnauthorized || gatedRecorder.Code == http.StatusSeeOther {
		t.Fatalf("a gated route must open for the OIDC session, got %d", gatedRecorder.Code)
	}
}

func TestOIDCLoginWrongGroupDenied(t *testing.T) {
	t.Parallel()

	idp := oidctest.NewStub(map[string]any{
		"sub": "user-2", "email": "intruder@example.org", "email_verified": true,
		"groups": []string{"pbx-phone"},
	})
	t.Cleanup(idp.Close)

	identity := openOIDCTestIdentity(t, idp)
	handler := identityHandler(t, identity)

	begin := httptest.NewRequest(http.MethodGet, "/auth/oauth/pocketid/begin", nil)
	beginRecorder := httptest.NewRecorder()
	handler.ServeHTTP(beginRecorder, begin)

	if beginRecorder.Code != http.StatusFound {
		t.Fatalf("begin status = %d, want 302", beginRecorder.Code)
	}

	state := beginQuery(t, beginRecorder.Header().Get("Location")).Get("state")
	callback := httptest.NewRequest(http.MethodGet, "/auth/oauth/pocketid/callback", nil)
	query := callback.URL.Query()
	query.Set("code", "code-1")
	query.Set("state", state)
	callback.URL.RawQuery = query.Encode()
	callbackRecorder := httptest.NewRecorder()
	handler.ServeHTTP(callbackRecorder, callback)

	if callbackRecorder.Code == http.StatusOK {
		t.Fatal("a wrong-group login must not succeed")
	}

	for _, cookie := range callbackRecorder.Result().Cookies() {
		if cookie.Name == "session_token" {
			t.Fatal("a denied login must never mint a session cookie")
		}
	}
}

func TestOIDCFirstLoginProvisionsOperatorSecondEmailClosed(t *testing.T) {
	t.Parallel()

	idp := oidctest.NewStub(map[string]any{
		"sub": "user-1", "email": "lars@example.org", "email_verified": true,
		"groups": []string{"pbx-history"},
	})
	t.Cleanup(idp.Close)

	identity := openOIDCTestIdentity(t, idp)
	handler := identityHandler(t, identity)

	if session := driveOIDCLogin(t, handler, idp); session == nil {
		t.Fatal("the first group-passing login must mint a session (operator provisioned)")
	}

	// A second, DIFFERENT identity with the right group: registration is
	// closed (MaxUsers = 1) — the login must fail closed, not create a
	// second account.
	idp.SetTokenClaims(map[string]any{
		"sub": "user-2", "email": "other@example.org", "email_verified": true,
		"groups": []string{"pbx-history"},
	})

	begin := httptest.NewRequest(http.MethodGet, "/auth/oauth/pocketid/begin", nil)
	beginRecorder := httptest.NewRecorder()
	handler.ServeHTTP(beginRecorder, begin)

	if beginRecorder.Code != http.StatusFound {
		t.Fatalf("begin status = %d, want 302", beginRecorder.Code)
	}

	state := beginQuery(t, beginRecorder.Header().Get("Location")).Get("state")
	callback := httptest.NewRequest(http.MethodGet, "/auth/oauth/pocketid/callback", nil)
	query := callback.URL.Query()
	query.Set("code", "code-2")
	query.Set("state", state)
	callback.URL.RawQuery = query.Encode()
	callbackRecorder := httptest.NewRecorder()
	handler.ServeHTTP(callbackRecorder, callback)

	if callbackRecorder.Code == http.StatusOK {
		t.Fatal("a second distinct identity must not log in (registration closed)")
	}
}

// driveOIDCLogin walks the full browser dance: begin (302 into the IdP,
// carrying state + nonce) → the stub adopts the nonce → callback mints the
// session cookie. Returns the session cookie.
func driveOIDCLogin(t *testing.T, handler http.Handler, idp *oidctest.StubIdP) *http.Cookie {
	t.Helper()

	begin := httptest.NewRequest(http.MethodGet, "/auth/oauth/pocketid/begin", nil)
	beginRecorder := httptest.NewRecorder()
	handler.ServeHTTP(beginRecorder, begin)

	if beginRecorder.Code != http.StatusFound {
		t.Fatalf("begin status = %d, want 302", beginRecorder.Code)
	}

	location := beginQuery(t, beginRecorder.Header().Get("Location"))
	idp.SetNonce(location.Get("nonce"))

	callback := httptest.NewRequest(http.MethodGet, "/auth/oauth/pocketid/callback", nil)
	query := callback.URL.Query()
	query.Set("code", "code-1")
	query.Set("state", location.Get("state"))
	callback.URL.RawQuery = query.Encode()
	callbackRecorder := httptest.NewRecorder()
	handler.ServeHTTP(callbackRecorder, callback)

	if callbackRecorder.Code != http.StatusFound {
		t.Fatalf("callback status = %d, want 302 (success redirect), body: %s",
			callbackRecorder.Code, callbackRecorder.Body.String())
	}

	if got := callbackRecorder.Header().Get("Location"); got != "/" {
		t.Errorf("callback redirect = %q, want /", got)
	}

	for _, cookie := range callbackRecorder.Result().Cookies() {
		if cookie.Name == "session_token" {
			return cookie
		}
	}

	t.Fatal("the callback must set the session cookie")

	return nil
}

// beginQuery parses a redirect Location's query.
func beginQuery(t *testing.T, location string) url.Values {
	t.Helper()

	parsed, err := url.Parse(location)
	if err != nil {
		t.Fatalf("parse redirect location %q: %v", location, err)
	}

	return parsed.Query()
}
