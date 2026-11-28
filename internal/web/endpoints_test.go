package web

import (
	"context"
	"encoding/json/v2"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/httputil/endpoints"
	"github.com/larsartmann/pbx-lens/internal/cli"
)

// webRouteTable is every route web.Register mounts, in listing form
// (ServeMux pattern syntax), the GET /endpoints listing included. The
// listing must serve exactly this table: each route is declared through
// mount, which mounts AND records it, so a mismatch means a route bypassed
// the seam.
func webRouteTable() []string {
	return []string{
		"GET /{$}",
		"GET /api/export.csv",
		"GET /api/export.jsonl",
		"GET /api/freshness",
		"GET /api/ops",
		"GET /app.css",
		"GET /calls/{kind}/{id}",
		"GET /calls/{kind}/{id}/recording",
		"GET /dashboard",
		"GET /dashboard/{$}",
		"GET /endpoints",
		"GET /favicon.svg",
		"GET /frag/dashboard",
		"GET /frag/staleness",
		"GET /login",
		"GET /ops",
		"GET /ops/{$}",
		"GET /robots.txt",
		"GET /search",
		"GET /search/{$}",
		"GET /segments/{number}",
		"GET /static/htmx.min.js",
		"GET /static/recent-views.js",
	}
}

// getEndpoints requests the listing from handler with the given method and
// Accept header and returns the drained response.
func getEndpoints(t *testing.T, handler http.Handler, method, accept string) (int, http.Header, string) {
	t.Helper()

	request := httptest.NewRequest(method, "/endpoints", nil)
	if accept != "" {
		request.Header.Set("Accept", accept)
	}

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	response := recorder.Result()
	defer func() { _ = response.Body.Close() }()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read /endpoints body: %v", err)
	}

	return response.StatusCode, response.Header, string(body)
}

// listedPatterns decodes the JSON listing and returns its patterns, sorted.
// Route.Pattern is the full ServeMux pattern as registered, so it compares
// directly with the route tables above.
func listedPatterns(t *testing.T, body string) []string {
	t.Helper()

	var listing endpoints.Response

	if err := json.Unmarshal([]byte(strings.TrimSpace(body)), &listing); err != nil {
		t.Fatalf("decode /endpoints listing: %v; body: %s", err, body)
	}

	patterns := make([]string, 0, len(listing.Endpoints))
	for _, route := range listing.Endpoints {
		patterns = append(patterns, route.Pattern)
	}

	slices.Sort(patterns)

	return patterns
}

// newAuthedTestHandler composes the serve wiring of an auth-enabled boot on
// ONE registry: the identity login surface, the web table (which mounts the
// listing last), then the session gate that wraps the whole mux.
func newAuthedTestHandler(t *testing.T) http.Handler {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	sys, cleanup, err := cli.OpenSystem(ctx, filepath.Join(t.TempDir(), "endpoints.db"))
	if err != nil {
		t.Fatalf("OpenSystem: %v", err)
	}

	t.Cleanup(cleanup)

	operator := openTestIdentity(t)
	mux := http.NewServeMux()
	routes := endpoints.NewRegistry()
	operator.Register(mux, routes)
	Register(mux, sys, Options{AuthEnabled: true, Identity: operator}, routes)

	return operator.Middleware()(mux)
}

func TestEndpointsListingCoversEveryWebRoute(t *testing.T) {
	status, header, body := getEndpoints(t, newTestServer(t), http.MethodGet, "")

	if status != http.StatusOK {
		t.Fatalf("GET /endpoints: status = %d, want 200; body: %s", status, body)
	}

	if contentType := header.Get("Content-Type"); !strings.HasPrefix(contentType, "application/json") {
		t.Errorf("GET /endpoints: Content-Type = %q, want application/json", contentType)
	}

	if vary := header.Get("Vary"); vary != "Accept" {
		t.Errorf("GET /endpoints: Vary = %q, want Accept", vary)
	}

	if cacheControl := header.Get("Cache-Control"); cacheControl != "no-store" {
		t.Errorf("GET /endpoints: Cache-Control = %q, want no-store", cacheControl)
	}

	want := webRouteTable()
	listed := listedPatterns(t, body)

	// An auth-free boot has no identity, so its table is the web routes
	// minus the login panel.
	want = slices.DeleteFunc(want, func(pattern string) bool { return pattern == "GET /login" })
	slices.Sort(want)

	if !slices.Equal(want, listed) {
		t.Errorf("GET /endpoints listing drifts from the route table\n listed: %v\n want:   %v", listed, want)
	}
}

func TestEndpointsListingNegotiatesHTML(t *testing.T) {
	status, header, body := getEndpoints(t, newTestServer(t), http.MethodGet, "text/html,application/xhtml+xml")

	if status != http.StatusOK {
		t.Fatalf("GET /endpoints (html): status = %d, want 200", status)
	}

	if contentType := header.Get("Content-Type"); !strings.HasPrefix(contentType, "text/html") {
		t.Errorf("GET /endpoints (html): Content-Type = %q, want text/html", contentType)
	}

	for _, want := range []string{"<table", `<th scope="col">Pattern</th>`, "GET /calls/{kind}/{id}"} {
		if !strings.Contains(body, want) {
			t.Errorf("HTML listing lacks %q; body: %.400s", want, body)
		}
	}
}

func TestEndpointsListingHEAD(t *testing.T) {
	// The listing handler renders the representation for HEAD as well; what
	// a client sees is net/http dropping that body on the wire, so the
	// contract is asserted through a real server, not a recorder.
	server := httptest.NewServer(newTestServer(t))
	t.Cleanup(server.Close)

	response, err := http.Head(server.URL + "/endpoints")
	if err != nil {
		t.Fatalf("HEAD /endpoints: %v", err)
	}

	defer func() { _ = response.Body.Close() }()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read HEAD /endpoints body: %v", err)
	}

	if response.StatusCode != http.StatusOK {
		t.Errorf("HEAD /endpoints: status = %d, want 200", response.StatusCode)
	}

	if len(body) != 0 {
		t.Errorf("HEAD /endpoints: body = %q, want empty", body)
	}
}

func TestEndpointsListingRejectsNonGET(t *testing.T) {
	// The web table has no catch-all pattern, so ServeMux itself answers a
	// POST to the GET-only listing with 405 before the listing handler
	// runs; the listing is never served.
	status, header, body := getEndpoints(t, newTestServer(t), http.MethodPost, "")

	if status != http.StatusMethodNotAllowed {
		t.Errorf("POST /endpoints: status = %d, want 405", status)
	}

	if allow := header.Get("Allow"); !strings.Contains(allow, http.MethodGet) {
		t.Errorf("POST /endpoints: Allow = %q, want it to name GET", allow)
	}

	if strings.Contains(body, `"endpoints"`) {
		t.Errorf("POST /endpoints served the listing; body: %.200s", body)
	}
}

func TestEndpointsListingBehindTheSessionGate(t *testing.T) {
	gated := newAuthedTestHandler(t)

	// The listing documents the app's surface, so an auth-enabled serve
	// must not hand it to an anonymous caller: the session gate in front of
	// the mux protects the listing exactly like every other route.
	status, _, body := getEndpoints(t, gated, http.MethodGet, "")

	if status != http.StatusUnauthorized {
		t.Errorf("GET /endpoints without a session: status = %d, want 401", status)
	}

	if strings.Contains(body, `"endpoints"`) {
		t.Errorf("GET /endpoints without a session served the listing; body: %.200s", body)
	}

	browser := httptest.NewRequest(http.MethodGet, "/endpoints", nil)
	browser.Header.Set("Accept", "text/html")

	recorder := httptest.NewRecorder()
	gated.ServeHTTP(recorder, browser)

	if recorder.Code != http.StatusSeeOther {
		t.Errorf("GET /endpoints without a session (browser): status = %d, want 303 to /login", recorder.Code)
	}

	session := loginSession(t, gated, "operator@pbx.lens")

	request := httptest.NewRequest(http.MethodGet, "/endpoints", nil)
	request.AddCookie(session)

	recorder = httptest.NewRecorder()
	gated.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("GET /endpoints with a session: status = %d, want 200", recorder.Code)
	}

	want := webRouteTable()
	listed := listedPatterns(t, recorder.Body.String())

	slices.Sort(want)

	if !slices.Equal(want, listed) {
		t.Errorf("authed GET /endpoints listing drifts from the route table\n listed: %v\n want:   %v", listed, want)
	}
}

// rawMuxRegistration matches a direct registration on a ServeMux value
// (mux.Handle, svc.Mux.HandleFunc, subMux.Handle, ...). ServeMux cannot
// enumerate its patterns, so such a route would be served but missing from
// the listing: every route the service owns goes through a mount seam.
var rawMuxRegistration = regexp.MustCompile(`\b(?:[A-Za-z_]\w*\.)*(?:m|mux|\w*[mM]ux)\.(?:Handle|HandleFunc)\(`)

// TestServiceRoutesGoThroughTheRegistry scans the two routing packages of
// the service, web and identity. internal/oidclogin/oidctest is a test
// double for the upstream OIDC provider and owns a mux of its own, which is
// why it is not scanned.
func TestServiceRoutesGoThroughTheRegistry(t *testing.T) {
	for _, dir := range []string{".", ".." + string(filepath.Separator) + "identity"} {
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("read package dir %s: %v", dir, err)
		}

		for _, entry := range entries {
			name := entry.Name()
			if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
				continue
			}

			assertNoRawMuxRegistration(t, filepath.Join(dir, name))
		}
	}
}

func assertNoRawMuxRegistration(t *testing.T, path string) {
	t.Helper()

	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	for lineNo, line := range strings.Split(string(source), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") {
			continue
		}

		if rawMuxRegistration.MatchString(trimmed) {
			t.Errorf(
				"%s:%d registers directly on the mux (%s): route it through the package's mount so the /endpoints listing cannot drift",
				path,
				lineNo+1,
				trimmed,
			)
		}
	}
}
