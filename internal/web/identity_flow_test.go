package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/larsartmann/httputil/endpoints"
	"github.com/larsartmann/pbx-lens/internal/identity"
	"github.com/larsartmann/pbx-lens/internal/identity/identitytest"
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

// identityOnlyHandler composes the login surface over a bare mux (with the
// stub "/" route) for web tests that only need an authenticated session —
// no app routes involved.
func identityOnlyHandler(t *testing.T, id *identity.Identity) http.Handler {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	id.Register(mux, endpoints.NewRegistry())

	return id.Middleware()(mux)
}

// The ceremony walkers live in internal/identity/identitytest (one copy,
// shared with the identity package's own gate tests); these delegates keep
// the web flow call sites short.
func loginChallenge(t *testing.T, handler http.Handler) (*http.Cookie, string) {
	t.Helper()

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
	t.Helper()

	return identitytest.PostAuth(t, handler, path, payload, cookie, token)
}

func loginSession(t *testing.T, handler http.Handler, email string) *http.Cookie {
	t.Helper()

	return identitytest.Session(t, handler, email)
}
