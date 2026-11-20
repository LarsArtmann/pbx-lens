package identity

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/larsartmann/pbx-lens/internal/identity/identitytest"
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

// identityHandler wires the composition the gate tests walk: the public
// login surface registered on a bare mux (plus a stub "/" so an authed
// request has a destination — the gate answers before any route matters),
// wrapped in the gate middleware. The same shape serve builds, minus the
// app routes (they are the web package's tests).
func identityHandler(t *testing.T, identity *Identity) http.Handler {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	identity.Register(mux)

	return identity.Middleware()(mux)
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
	request.Header.Set("Hx-Request", "true")

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

	recorder = postAuthJSON(t, handler, "/auth/register", map[string]string{
		"email": "intruder@pbx.lens", "display_name": "X",
	}, csrfCookie, token)

	if recorder.Code < 400 || recorder.Code > 499 {
		t.Errorf(
			"second registration status = %d, want a 4xx refusal (body: %s)",
			recorder.Code,
			recorder.Body.String(),
		)
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
		DSN: dsn, RPID: "localhost", RPOrigins: []string{"http://localhost:8300"},
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

func TestIdentityCookieSecureFollowsOriginScheme(t *testing.T) {
	tests := []struct {
		name        string
		origins     []string
		wantSecure  bool
		description string
	}{
		{
			name:        "http origin keeps the cookie non-Secure (LAN http posture)",
			origins:     []string{"http://localhost:8300"},
			wantSecure:  false,
			description: "browsers drop Secure cookies on plain http outside localhost",
		},
		{
			name:        "https origin marks the cookie Secure (proxy posture)",
			origins:     []string{"https://lens.example.com"},
			wantSecure:  true,
			description: "Secure must ride once TLS fronts the lens",
		},
		{
			name:        "one http origin among https keeps the cookie non-Secure",
			origins:     []string{"https://lens.example.com", "http://localhost:8300"},
			wantSecure:  false,
			description: "the weakest browser-facing scheme wins (mixed origins are a misconfig anyway)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			identity, err := OpenIdentity(context.Background(), IdentityConfig{
				DSN:       filepath.Join(t.TempDir(), "identity.db"),
				RPID:      "localhost",
				RPOrigins: tt.origins,
			})
			if err != nil {
				t.Fatalf("OpenIdentity: %v", err)
			}

			t.Cleanup(func() {
				if cerr := identity.Close(); cerr != nil {
					t.Errorf("identity close: %v", cerr)
				}
			})

			handler := identityHandler(t, identity)
			cookie := loginSession(t, handler, "operator@pbx.lens")

			if cookie.Secure != tt.wantSecure {
				t.Errorf(
					"session cookie Secure = %v, want %v (%s)",
					cookie.Secure,
					tt.wantSecure,
					tt.description,
				)
			}
		})
	}
}

// TestIdentityLogoutEndsTheSessionEverywhere walks the full sign-out
// contract: the authed nav renders the logout form (and only then),
// POST /auth/logout invalidates the session SERVER-SIDE (the same cookie
// value never works again), and the logged-out gate bounces the browser
// to /login.
func TestIdentityLogoutEndsTheSessionEverywhere(t *testing.T) {
	identity := openTestIdentity(t)

	// A bare mux with the stub "/" route: the logout contract is the
	// session's server-side death, not any app route's content — authed
	// GET / must reach A route (200), logged-out GET / must 303.
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	identity.Register(mux)
	handler := identity.Middleware()(mux)

	csrfCookie, token := loginChallenge(t, handler)
	session := loginSession(t, handler, "operator@pbx.lens")

	authed := func() *httptest.ResponseRecorder {
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		request.Host = "localhost:8300"
		request.Header.Set("Accept", "text/html")
		request.Header.Set("Origin", "http://localhost:8300")
		request.AddCookie(csrfCookie)
		request.AddCookie(session)

		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)

		return recorder
	}

	// The authed nav is honest: the logout form renders.
	first := authed()
	if first.Code != http.StatusOK {
		t.Fatalf("authed GET / status = %d, want 200 (body: %.200s)", first.Code, first.Body.String())
	}

	// The NAV's logout FORM (id="logout-form" + its CSRF token) is the web
	// package's render — flow_test.go pins that markup; this test owns the
	// session's server-side death.

	// POST /auth/logout with the browser ceremony posture.
	logout := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	logout.Host = "localhost:8300"
	logout.Header.Set("Origin", "http://localhost:8300")
	logout.Header.Set("Referer", "http://localhost:8300/")
	logout.Header.Set("X-Csrf-Token", token)
	logout.AddCookie(csrfCookie)
	logout.AddCookie(session)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, logout)

	if recorder.Code != http.StatusOK {
		t.Fatalf("POST /auth/logout status = %d, want 200 (body: %.200s)", recorder.Code, recorder.Body.String())
	}

	cleared := false

	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == defaultSessionCookie && cookie.MaxAge < 0 {
			cleared = true
		}
	}

	if !cleared {
		t.Error("logout did not clear the session cookie (want Max-Age < 0)")
	}

	// The SAME cookie value is dead: the gate bounces the browser to /login.
	after := authed()
	if after.Code != http.StatusSeeOther {
		t.Errorf(
			"logged-out GET / status = %d, want 303 (the session must be invalidated server-side)",
			after.Code,
		)
	}

	if location := after.Header().Get("Location"); location != "/login" {
		t.Errorf("logged-out redirect Location = %q, want /login", location)
	}

	// The logged-out nav must NOT render the logout form.
	bounce := httptest.NewRequest(http.MethodGet, "/login", nil)
	bounce.Host = "localhost:8300"
	bounce.Header.Set("Accept", "text/html")
	bounce.AddCookie(csrfCookie)

	loginPage := httptest.NewRecorder()
	handler.ServeHTTP(loginPage, bounce)

	if strings.Contains(loginPage.Body.String(), `id="logout-form"`) {
		t.Error("the logged-out login page renders a logout form")
	}
}

func TestAllOriginsHTTPS(t *testing.T) {
	tests := []struct {
		name    string
		origins []string
		want    bool
	}{
		{name: "nil origins", origins: nil, want: false},
		{name: "single http origin", origins: []string{"http://localhost:8300"}, want: false},
		{name: "single https origin", origins: []string{"https://pbx.artmann.tech"}, want: true},
		{name: "all https", origins: []string{"https://a.example.com", "https://b.example.com"}, want: true},
		{name: "mixed schemes", origins: []string{"https://a.example.com", "http://b.example.com"}, want: false},
		{name: "padded https origin", origins: []string{" https://padded.example.com "}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := allOriginsHTTPS(tt.origins); got != tt.want {
				t.Errorf("allOriginsHTTPS(%v) = %v, want %v", tt.origins, got, tt.want)
			}
		})
	}
}

// The register form is honest only while the operator seat is open
// (MaxUsers: 1 — T08, 2026-11-20): a fresh identity DB shows it (the first
// visitor IS the operator), and a reopened DB whose seat is taken hides it
// — a form that always fails is the worst visible UX wart. NoRegistration
// is a boot-time read of the drained read model, so the reopen leg proves
// the drain landed before the login page rendered.
func TestRegistrationSectionFollowsTheSeat(t *testing.T) {
	dsn := filepath.Join(t.TempDir(), "identity.db")

	{
		identity, err := OpenIdentity(context.Background(), IdentityConfig{
			DSN:       dsn,
			RPID:      "localhost",
			RPOrigins: []string{"http://localhost:8300"},
		})
		if err != nil {
			t.Fatalf("OpenIdentity (fresh): %v", err)
		}

		handler := identityHandler(t, identity)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/login", nil))

		if recorder.Code != http.StatusOK {
			t.Fatalf("GET /login status = %d, want 200", recorder.Code)
		}

		if !strings.Contains(recorder.Body.String(), `<section id="lp-register-section"`) {
			t.Error("fresh identity DB hides the register section — the operator could never sign up")
		}

		// Take the seat (loginSession asserts the 201).
		loginSession(t, handler, "operator@example.org")

		if cerr := identity.Close(); cerr != nil {
			t.Fatalf("identity close: %v", cerr)
		}
	}

	{
		reopened, err := OpenIdentity(context.Background(), IdentityConfig{
			DSN:       dsn,
			RPID:      "localhost",
			RPOrigins: []string{"http://localhost:8300"},
		})
		if err != nil {
			t.Fatalf("OpenIdentity (reopen): %v", err)
		}

		t.Cleanup(func() {
			if cerr := reopened.Close(); cerr != nil {
				t.Errorf("identity close: %v", cerr)
			}
		})

		handler := identityHandler(t, reopened)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/login", nil))

		if recorder.Code != http.StatusOK {
			t.Fatalf("GET /login status = %d, want 200", recorder.Code)
		}

		if strings.Contains(recorder.Body.String(), `<section id="lp-register-section"`) {
			t.Error("taken seat still renders the register section — a form that always fails")
		}
	}
}

// The ceremony walkers live in internal/identity/identitytest — one copy
// shared with the web package's flow tests (the 2026-11-20 dedup gate
// flagged the duplicated walk). Thin delegates keep the in-package call
// sites short; identitytest deliberately does not import this package, so
// the in-package tests may import it.
func loginChallenge(t *testing.T, handler http.Handler) (*http.Cookie, string) {
	return identitytest.Challenge(t, handler)
}

func postAuthJSON(
	t *testing.T,
	handler http.Handler,
	path string,
	payload map[string]string,
	cookie *http.Cookie,
	token string,
) *httptest.ResponseRecorder {
	return identitytest.PostAuth(t, handler, path, payload, cookie, token)
}

func loginSession(t *testing.T, handler http.Handler, email string) *http.Cookie {
	return identitytest.Session(t, handler, email)
}

func extractCSRFToken(t *testing.T, html string) string {
	return identitytest.CSRFToken(t, html)
}
