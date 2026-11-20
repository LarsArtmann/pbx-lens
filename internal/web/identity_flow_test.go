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

	"github.com/larsartmann/pbx-lens/internal/identity"
)

// The web-side auth FLOW walkers (T16, 2026-11-20): the identity package
// owns the ceremonies and its own copies; these walk the same browser dance
// against the WEB composition (gate + app routes) for the flow and ops
// tests. Deliberate duplication — test helpers may not import across
// packages, and the web tests need a real session cookie, not a mock.
func openTestIdentity(t *testing.T) *identity.Identity {
	t.Helper()

	id, err := identity.OpenIdentity(context.Background(), identity.IdentityConfig{
		DSN:       filepath.Join(t.TempDir(), "identity.db"),
		RPID:      "localhost",
		RPOrigins: []string{"http://localhost:8300"},
	})
	if err != nil {
		t.Fatalf("OpenIdentity: %v", err)
	}

	t.Cleanup(func() {
		if cerr := id.Close(); cerr != nil {
			t.Errorf("identity close: %v", cerr)
		}
	})

	return id
}

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

func postAuthJSON(
	t *testing.T,
	handler http.Handler,
	path string,
	payload map[string]string,
	cookie *http.Cookie,
	token string,
) *httptest.ResponseRecorder {
	t.Helper()

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal %s request: %v", path, err)
	}

	request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(string(body)))
	request.Host = "localhost:8300"
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", "http://localhost:8300")
	request.Header.Set("Referer", "http://localhost:8300/login")
	request.Header.Set("X-Csrf-Token", token)

	if cookie != nil {
		request.AddCookie(cookie)
	}

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	return recorder
}

func loginSession(t *testing.T, handler http.Handler, email string) *http.Cookie {
	t.Helper()

	csrfCookie, token := loginChallenge(t, handler)

	recorder := postAuthJSON(t, handler, "/auth/register", map[string]string{
		"email": email, "display_name": "Operator",
	}, csrfCookie, token)

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

func extractCSRFToken(t *testing.T, html string) string {
	t.Helper()

	match := regexp.MustCompile(`name="csrf-token" content="([^"]+)"`).FindStringSubmatch(html)
	if match == nil {
		t.Fatal("login page carries no csrf-token meta — the token bridge is broken")
	}

	return match[1]
}

// identityOnlyHandler composes the login surface over a bare mux (with the
// stub "/" route) for web tests that only need an authenticated session —
// no app routes involved.
func identityOnlyHandler(t *testing.T, id *identity.Identity) http.Handler {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	id.Register(mux)

	return id.Middleware()(mux)
}
