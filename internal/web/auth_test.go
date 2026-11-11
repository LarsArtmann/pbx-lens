package web

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The identity session gate (2026-11-11 usermgmt adoption): an event-sourced
// operator account behind passwordless WebAuthn login. These tests pin the
// GATE contract — public paths stay open, browsers get redirected to /login,
// API clients get 401, and the first registration both works and closes the
// door (MaxUsers = 1). The WebAuthn ceremonies themselves need a real
// authenticator; they are covered upstream (usermgmt + loginpage suites) and
// live-verified by the browser probe lane.

// openTestIdentity brings up a real identity context on a throwaway SQLite
// file — the same constructors serve runs in production (SQL event store,
// SQL session store, SQL read models, checkpoint store), so the tests
// exercise the persistence wiring too, not an in-memory shortcut.
func openTestIdentity(t *testing.T) *Identity {
	t.Helper()

	identity, err := OpenIdentity(context.Background(), IdentityConfig{
		DSN:       filepath.Join(t.TempDir(), "identity.db"),
		RPID:      "localhost",
		RPOrigins: []string{"http://localhost:8300"},
	})
	if err != nil {
		t.Fatalf("OpenIdentity: %v", err)
	}

	t.Cleanup(func() {
		if cerr := identity.Close(); cerr != nil {
			t.Errorf("identity close: %v", cerr)
		}
	})

	return identity
}

// identityHandler wires the full composition: the app routes, the public
// login surface, and the gate middleware — the same shape serve builds.
func identityHandler(t *testing.T, identity *Identity) http.Handler {
	t.Helper()

	mux := newTestServer(t)
	identity.Register(mux)

	return identity.Middleware()(mux)
}

// loginChallenge walks the CSRF half of the browser flow: fetch /login (the
// middleware issues its token cookie, the page embeds the token in its meta
// tag) and return both — the pair every auth ceremony POST must present.
func loginChallenge(t *testing.T, handler http.Handler) (*http.Cookie, string) {
	t.Helper()

	request := httptest.NewRequest(http.MethodGet, "/login", nil)
	request.Header.Set("Accept", "text/html")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("GET /login status = %d, want 200", recorder.Code)
	}

	var csrfCookie *http.Cookie

	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == "csrf_token" {
			csrfCookie = cookie
		}
	}

	if csrfCookie == nil {
		t.Fatal("GET /login issued no csrf_token cookie — the CSRF middleware is not wired")
	}

	return csrfCookie, extractCSRFToken(t, recorder.Body.String())
}

// loginSession walks the browser flow headlessly: fetch /login (the CSRF
// middleware issues its token cookie and the page embeds the token in its
// meta tag), then POST /auth/register with both. The response carries the
// session cookie the gate accepts. Registration on a fresh identity always
// succeeds — the MaxUsers=1 door is still open.
func loginSession(t *testing.T, handler http.Handler, email string) *http.Cookie {
	t.Helper()

	csrfCookie, token := loginChallenge(t, handler)

	body, err := json.Marshal(map[string]string{"email": email, "display_name": "Operator"})
	if err != nil {
		t.Fatalf("marshal register request: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(string(body)))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Csrf-Token", token)
	request.AddCookie(csrfCookie)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("POST /auth/register status = %d, want 201 (body: %s)", recorder.Code, recorder.Body.String())
	}

	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == "session_token" {
			return cookie
		}
	}

	t.Fatal("registration set no session_token cookie")

	return nil
}

// extractCSRFToken reads the login page's <meta name="csrf-token"> value —
// the same token the login page's own JavaScript sends on every ceremony.
func extractCSRFToken(t *testing.T, html string) string {
	t.Helper()

	match := regexp.MustCompile(`name="csrf-token" content="([^"]+)"`).FindStringSubmatch(html)
	if match == nil {
		t.Fatal("login page carries no csrf-token meta — the token bridge is broken")
	}

	return match[1]
}

func TestIdentityGateRedirectsBrowsersAndChallengesAPIs(t *testing.T) {
	identity := openTestIdentity(t)
	handler := identityHandler(t, identity)

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Accept", "text/html")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusSeeOther {
		t.Errorf("browser without session status = %d, want 303", recorder.Code)
	}

	if location := recorder.Header().Get("Location"); location != "/login" {
		t.Errorf("browser redirect Location = %q, want /login", location)
	}

	request = httptest.NewRequest(http.MethodGet, "/", nil)
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Errorf("API client without session status = %d, want 401", recorder.Code)
	}

	request = httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("HX-Request", "true")
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusSeeOther {
		t.Errorf("htmx poll without session status = %d, want 303", recorder.Code)
	}
}

func TestIdentityPublicPathsStayOpen(t *testing.T) {
	identity := openTestIdentity(t)
	handler := identityHandler(t, identity)

	for _, path := range []string{"/login", "/app.css", "/robots.txt"} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusOK {
			t.Errorf("GET %s without session status = %d, want 200 (public)", path, recorder.Code)
		}
	}
}

func TestIdentityRegistrationClosesAfterFirst(t *testing.T) {
	identity := openTestIdentity(t)
	handler := identityHandler(t, identity)

	session := loginSession(t, handler, "operator@pbx.lens")
	if session == nil {
		t.Fatal("first registration produced no session")
	}

	// The app accepts the session the registration issued.
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.AddCookie(session)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Errorf("list page with session status = %d, want 200", recorder.Code)
	}

	// The door is closed: a second registration (any email) must fail at
	// the service — reaching it through a VALID CSRF ceremony, since the
	// CSRF wall itself is not the behavior under test here.
	csrfCookie, token := loginChallenge(t, handler)

	body, err := json.Marshal(map[string]string{"email": "intruder@pbx.lens", "display_name": "X"})
	if err != nil {
		t.Fatalf("marshal register request: %v", err)
	}

	request = httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(string(body)))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Csrf-Token", token)
	request.AddCookie(csrfCookie)
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code < 400 || recorder.Code > 499 {
		t.Errorf("second registration status = %d, want a 4xx refusal (body: %s)", recorder.Code, recorder.Body.String())
	}

	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == "session_token" {
			t.Error("a closed registration must not issue a session")
		}
	}
}

func TestIdentityIdentityDBSurvivesReopen(t *testing.T) {
	// Sessions and the operator account live in SQLite — a serve restart
	// must not lose them (the property that makes -auth-db deployable).
	dir := t.TempDir()
	dsn := filepath.Join(dir, "identity.db")

	identity, err := OpenIdentity(context.Background(), IdentityConfig{
		DSN: dsn, RPID: "localhost", RPOrigins: []string{"http://localhost:8300"},
	})
	if err != nil {
		t.Fatalf("open identity: %v", err)
	}

	handler := identityHandler(t, identity)
	session := loginSession(t, handler, "operator@pbx.lens")

	if err := identity.Close(); err != nil {
		t.Fatalf("close identity: %v", err)
	}

	reopened, err := OpenIdentity(context.Background(), IdentityConfig{
		DSN: dsn, RPID: "127.0.0.1", RPOrigins: []string{"http://localhost:8300"},
	})
	if err != nil {
		t.Fatalf("reopen identity: %v", err)
	}

	t.Cleanup(func() { _ = reopened.Close() })

	restarted := identityHandler(t, reopened)

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.AddCookie(session)
	recorder := httptest.NewRecorder()
	restarted.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Errorf("session from before the restart status = %d, want 200 (SQL sessions survive)", recorder.Code)
	}
}

func TestIdentityReadyAfterDrain(t *testing.T) {
	identity := openTestIdentity(t)

	// NewService blocks until the journal drain completes, so a fresh
	// identity is ready the moment OpenIdentity returns.
	if !identity.Ready() {
		t.Error("identity not ready after construction — the readiness predicate is wrong")
	}
}
