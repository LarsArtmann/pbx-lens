// Package identitytest carries the browser-walk helpers the identity
// ceremonies need in BOTH test suites (T16, 2026-11-20): the identity
// package's own gate tests and the web package's flow tests. One copy, so
// the CSRF dance cannot drift between the packages that walk it — the
// 2026-11-20 dedup gate flagged exactly this duplication when the identity
// extraction first duplicated the walkers. The package deliberately does
// NOT import identity (an in-package test file cannot import a package
// that imports it back); the walkers speak plain http.Handler.
//
// The walkers drive a REAL handler over the full middleware chain (CSRF
// cookie + token pair, same-origin attestation) — no mocks.
package identitytest

import (
	"context"
	"database/sql"
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

// Challenge walks the CSRF half of the browser flow: fetch /login (the
// middleware issues its token cookie, the page embeds the token in its meta
// tag) and return both — the pair every auth ceremony POST must present.
func Challenge(t *testing.T, handler http.Handler) (*http.Cookie, string) {
	t.Helper()

	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/login", nil)
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

	return csrfCookie, CSRFToken(t, recorder.Body.String())
}

// PostAuth drives an auth ceremony POST the way a real browser does:
// same-origin attestation (Host + Origin + Referer — nosurf requires them
// on secure-classified requests), the CSRF cookie/token pair, and the
// cookie jar. Returns the recorder for status/cookie assertions.
func PostAuth(
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

	request := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		path,
		strings.NewReader(string(body)),
	)
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

// Session walks the browser flow headlessly: fetch /login, then POST
// /auth/register with the challenge pair. The response carries the session
// cookie the gate accepts. Registration on a fresh identity always
// succeeds — the MaxUsers=1 door is still open.
func Session(t *testing.T, handler http.Handler, email string) *http.Cookie {
	t.Helper()

	csrfCookie, token := Challenge(t, handler)

	recorder := PostAuth(t, handler, "/auth/register", map[string]string{
		"email": email, "display_name": "Operator",
	}, csrfCookie, token)

	if recorder.Code != http.StatusCreated {
		t.Fatalf(
			"POST /auth/register status = %d, want 201 (body: %s)",
			recorder.Code,
			recorder.Body.String(),
		)
	}

	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == "session_token" {
			return cookie
		}
	}

	t.Fatal("registration set no session_token cookie")

	return nil
}

// CSRFToken extracts the page-embedded token from rendered login HTML.
func CSRFToken(t *testing.T, html string) string {
	t.Helper()

	match := regexp.MustCompile(`name="csrf-token" content="([^"]+)"`).FindStringSubmatch(html)
	if match == nil {
		t.Fatal("login page carries no csrf-token meta tag — the ceremony cannot run")
	}

	return match[1]
}

// TableNames lists a SQLite file's tables, sorted: the witness the open-
// policy and backup tests use to prove a refusal created nothing and an
// open created exactly the identity schema.
func TableNames(t *testing.T, dsn string) []string {
	t.Helper()

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open %s: %v", dsn, err)
	}

	defer func() { _ = db.Close() }()

	rows, err := db.QueryContext(
		t.Context(),
		"SELECT name FROM sqlite_master WHERE type = 'table' ORDER BY name",
	)
	if err != nil {
		t.Fatalf("list tables: %v", err)
	}

	defer func() { _ = rows.Close() }()

	var names []string

	for rows.Next() {
		var name string

		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan table name: %v", err)
		}

		names = append(names, name)
	}

	if err := rows.Err(); err != nil {
		t.Fatalf("iterate tables: %v", err)
	}

	return names
}
