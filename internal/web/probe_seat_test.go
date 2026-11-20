package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestProbeSeatCount(t *testing.T) {
	dsn := filepath.Join(t.TempDir(), "identity.db")
	i1, err := OpenIdentity(context.Background(), IdentityConfig{DSN: dsn, RPID: "localhost", RPOrigins: []string{"http://localhost:8300"}})
	if err != nil {
		t.Fatal(err)
	}
	mux := newTestServer(t)
	i1.Register(mux)
	h := i1.Middleware()(mux)
	loginSession(t, h, "operator@example.org")
	t.Logf("fresh count after register: %d", i1.service.ReadModel().Count())
	_ = i1.Close()

	i2, err := OpenIdentity(context.Background(), IdentityConfig{DSN: dsn, RPID: "localhost", RPOrigins: []string{"http://localhost:8300"}})
	if err != nil {
		t.Fatal(err)
	}
	defer i2.Close()
	t.Logf("reopen count: %d", i2.service.ReadModel().Count())

	mux2 := newTestServer(t)
	i2.Register(mux2)
	h2 := i2.Middleware()(mux2)
	rec := httptest.NewRecorder()
	h2.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/login", nil))
	body := rec.Body.String()
	t.Logf("reopen login has register section: %v", contains(body, "lp-register-section"))
	t.Logf("reopen login has accent: %v", contains(body, "0f766e"))
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}
