package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

// Benchmarks for the identity gate chain (plan A19): the per-request cost
// the operator pays on every route — session enrichment (one indexed SQL
// read on a cookie hit), the CSRF wrapper, and the gate decision. The
// public path is the free lane (robots.txt/app.css bypass); the sessionless
// path is the 401/303 fast path unauthenticated probes hit; the session
// path is the steady-state operator cost. The session is minted through
// the REAL /auth/register ceremony so the bench exercises the same SQL
// rows production resolves.

func benchIdentity(b *testing.B) (http.Handler, *http.Cookie) {
	b.Helper()

	identity, err := OpenIdentity(context.Background(), IdentityConfig{
		DSN:       filepath.Join(b.TempDir(), "identity.db"),
		RPID:      "localhost",
		RPOrigins: []string{"http://localhost:8300"},
	})
	if err != nil {
		b.Fatalf("OpenIdentity: %v", err)
	}

	b.Cleanup(func() {
		if cerr := identity.Close(); cerr != nil {
			b.Errorf("identity close: %v", cerr)
		}
	})

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("GET /app.css", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	identity.Register(mux)

	handler := identity.Middleware()(mux)

	// Mint the operator session through the real ceremony: fetch /login for
	// the CSRF pair, POST /auth/register with both, keep the session cookie.
	recorder := httptest.NewRecorder()
	login := httptest.NewRequest(http.MethodGet, "/login", nil)
	login.Header.Set("Accept", "text/html")
	handler.ServeHTTP(recorder, login)

	if recorder.Code != http.StatusOK {
		b.Fatalf("GET /login status = %d, want 200", recorder.Code)
	}

	var csrfCookie *http.Cookie

	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == "csrf_token" {
			csrfCookie = cookie
		}
	}

	if csrfCookie == nil {
		b.Fatal("login page issued no csrf cookie")
	}

	const metaTag = `name="csrf-token" content="`

	page := recorder.Body.String()

	start := strings.Index(page, metaTag)

	if start < 0 {
		b.Fatal("login page embeds no csrf meta tag")
	}

	start += len(metaTag)

	end := strings.Index(page[start:], `"`)

	if end < 0 {
		b.Fatal("csrf meta tag is unterminated")
	}

	register := httptest.NewRequest(http.MethodPost, "/auth/register",
		strings.NewReader(`{"email":"operator@pbx.lens","display_name":"Operator"}`))
	register.Host = "localhost:8300"
	register.Header.Set("Content-Type", "application/json")
	register.Header.Set("Origin", "http://localhost:8300")
	register.Header.Set("Referer", "http://localhost:8300/login")
	register.AddCookie(csrfCookie)
	register.Header.Set("X-Csrf-Token", page[start:start+end])

	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, register)

	if recorder.Code != http.StatusCreated {
		b.Fatalf("POST /auth/register status = %d, want 201 (body: %s)", recorder.Code, recorder.Body.String())
	}

	var session *http.Cookie

	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == "session_token" {
			session = cookie
		}
	}

	if session == nil {
		b.Fatal("registration issued no session cookie")
	}

	return handler, session
}

func benchRequest(handler http.Handler, b *testing.B, build func() *http.Request) {
	b.Helper()

	b.ReportAllocs()

	for b.Loop() {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, build())

		if recorder.Code != http.StatusOK {
			b.Fatalf("status = %d, want 200", recorder.Code)
		}
	}
}

func BenchmarkIdentityGatePublicPath(b *testing.B) {
	handler, _ := benchIdentity(b)

	build := func() *http.Request {
		return httptest.NewRequest(http.MethodGet, "/app.css", nil)
	}

	b.ResetTimer()
	benchRequest(handler, b, build)
}

func BenchmarkIdentityGateSessionless(b *testing.B) {
	handler, _ := benchIdentity(b)

	// The protected surface with no cookie: the gate's fast rejection path
	// (API-shaped, so the 401 branch — no redirect render).
	build := func() *http.Request {
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		request.Header.Set("Accept", "application/json")

		return request
	}

	b.ReportAllocs()

	b.ResetTimer()

	for b.Loop() {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, build())

		if recorder.Code != http.StatusUnauthorized {
			b.Fatalf("status = %d, want 401", recorder.Code)
		}
	}
}

func BenchmarkIdentityGateValidSession(b *testing.B) {
	handler, session := benchIdentity(b)

	build := func() *http.Request {
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		request.AddCookie(session)

		return request
	}

	b.ResetTimer()
	benchRequest(handler, b, build)
}
