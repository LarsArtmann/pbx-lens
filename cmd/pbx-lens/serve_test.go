package main

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	appkit "github.com/larsartmann/go-appkit"
	"github.com/larsartmann/go-appkit/testkit"
	"github.com/larsartmann/pbx-lens/internal/cli"
	"github.com/larsartmann/pbx-lens/internal/web"
	"golang.org/x/crypto/bcrypt"
)

// noAutoGzip returns a client that does not add Accept-Encoding itself, so
// a test can negotiate gzip explicitly and still see the raw headers (the
// default transport hides Content-Encoding by decompressing transparently).
func noAutoGzip() *http.Client {
	return &http.Client{Transport: &http.Transport{DisableCompression: true}}
}

// get issues a GET against the served chain and returns status plus the
// response for header inspection. It takes testing.TB so benchmarks can
// share the same round-trip path.
func get(tb testing.TB, client *http.Client, base, path string, headers map[string]string) (int, http.Header, string) {
	tb.Helper()

	req, err := http.NewRequestWithContext(tb.Context(), http.MethodGet, base+path, nil)
	if err != nil {
		tb.Fatalf("build request %s: %v", path, err)
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		tb.Fatalf("GET %s: %v", path, err)
	}

	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		tb.Fatalf("read %s body: %v", path, err)
	}

	return resp.StatusCode, resp.Header, string(body)
}

// TestServeStackEndToEnd exercises the composition buildServiceConfig wires
// (Server-Timing, Compression, ETag, Version, Metrics) through the real
// listener via appkit's testkit: the middlewares only earn their keep if
// the served chain exhibits them.
func TestServeStackEndToEnd(t *testing.T) {
	sys, cleanup, err := cli.OpenSystem(t.Context(), "")
	if err != nil {
		t.Fatalf("open memory index: %v", err)
	}

	defer cleanup()

	serviceConfig, err := buildServiceConfig("127.0.0.1:0", "", nil, nil)
	if err != nil {
		t.Fatalf("build service config: %v", err)
	}

	svc, err := appkit.NewService(serviceConfig)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	web.Register(svc.Mux, sys, web.Options{})

	ts := testkit.Serve(t, svc)
	client := noAutoGzip()

	// The cacheable surfaces: revalidation (200 → ETag → 304) and gzip on
	// both the embedded asset and the hot 30s poll fragment. Fragment routes
	// additionally carry Cache-Control: no-cache on the 200 AND the 304, so
	// browsers and htmx polls revalidate instead of serving stale HTML.
	assertConditionalGET(t, client, ts.FullChainURL, "/static/htmx.min.js", false)
	assertGzip(t, client, ts.FullChainURL, "/static/htmx.min.js")
	assertConditionalGET(t, client, ts.FullChainURL, "/frag/dashboard", true)
	assertGzip(t, client, ts.FullChainURL, "/frag/dashboard")
	assertConditionalGET(t, client, ts.FullChainURL, "/ops", true)

	status, headers, _ := get(t, client, ts.FullChainURL, "/frag/staleness", nil)
	if status != http.StatusOK {
		t.Fatalf("GET staleness fragment status = %d, want 200", status)
	}

	if headers.Get("ETag") != "" {
		t.Errorf("staleness fragment carries ETag %q; the age it embeds changes every response", headers.Get("ETag"))
	}

	if cc := headers.Get("Cache-Control"); cc != "no-cache" {
		t.Errorf(
			"staleness fragment Cache-Control = %q, want no-cache (revalidate every poll even without an ETag)",
			cc,
		)
	}

	_, headers, _ = get(t, client, ts.FullChainURL, "/dashboard", nil)

	timing := headers.Get("Server-Timing")
	if !strings.Contains(timing, "total") || !strings.Contains(timing, "stats") {
		t.Errorf("dashboard Server-Timing = %q, want total + stats entries", timing)
	}

	status, _, body := get(t, client, ts.FullChainURL, "/version", nil)
	if status != http.StatusOK {
		t.Fatalf("GET /version status = %d, want 200", status)
	}

	if !strings.Contains(body, version) {
		t.Errorf("/version body %q does not carry the build version %q", body, version)
	}

	status, _, metricsBody := get(t, client, ts.FullChainURL, "/metrics", nil)
	if status != http.StatusOK {
		t.Fatalf("GET /metrics status = %d, want 200", status)
	}

	assertPrometheusScrape(t, metricsBody)
}

// assertPrometheusScrape is the scrape smoke (plan T14): a real scraper
// (Prometheus or Gatus) must be able to parse the surface, so the body has
// to be text format version 0.0.4 — # HELP / # TYPE commentary lines plus
// at least one sample line per family — and carry the appkit_build_info
// gauge a host dashboard pins its version panel to.
func assertPrometheusScrape(t *testing.T, body string) {
	t.Helper()

	if !strings.Contains(body, "# HELP") || !strings.Contains(body, "# TYPE") {
		t.Errorf("/metrics body is not Prometheus text format (missing # HELP/# TYPE), got:\n%.200s", body)
	}

	if !strings.Contains(body, "appkit_build_info") {
		t.Errorf("/metrics missing appkit_build_info (host version panels read it), got:\n%.200s", body)
	}

	for i, line := range strings.Split(body, "\n") {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if !strings.Contains(line, "{") && !strings.Contains(line, " ") {
			t.Errorf("metrics line %d is not a sample (<name>{labels} <value>): %q", i+1, line)
		}
	}
}

// assertConditionalGET cycles a path through the full revalidation
// contract: 200 with an ETag, then a 304 with an empty body on
// If-None-Match. wantNoCache additionally asserts Cache-Control: no-cache
// on both responses (the deterministic-revalidation contract for HTML
// fragments; static assets opt out).
func assertConditionalGET(t *testing.T, client *http.Client, base, path string, wantNoCache bool) {
	t.Helper()

	status, headers, _ := get(t, client, base, path, nil)
	if status != http.StatusOK {
		t.Fatalf("GET %s status = %d, want 200", path, status)
	}

	etag := headers.Get("ETag")
	if etag == "" {
		t.Fatalf("%s served without an ETag; conditional revalidation cannot work", path)
	}

	if wantNoCache {
		if cc := headers.Get("Cache-Control"); cc != "no-cache" {
			t.Errorf("%s 200 Cache-Control = %q, want no-cache", path, cc)
		}
	}

	status, notModifiedHeaders, body := get(t, client, base, path, map[string]string{"If-None-Match": etag})
	if status != http.StatusNotModified {
		t.Fatalf("%s conditional GET status = %d, want 304 (body: %q)", path, status, body)
	}

	if body != "" {
		t.Errorf("%s 304 carried a body of %d bytes, want empty", path, len(body))
	}

	if wantNoCache {
		if cc := notModifiedHeaders.Get("Cache-Control"); cc != "no-cache" {
			t.Errorf("%s 304 Cache-Control = %q, want no-cache (the header must survive revalidation)", path, cc)
		}
	}
}

// assertGzip checks that the path negotiates gzip when offered.
func assertGzip(t *testing.T, client *http.Client, base, path string) {
	t.Helper()

	status, headers, _ := get(t, client, base, path, map[string]string{"Accept-Encoding": "gzip"})
	if status != http.StatusOK || headers.Get("Content-Encoding") != "gzip" {
		t.Errorf(
			"%s with Accept-Encoding: gzip = %d, Content-Encoding %q, want 200 + gzip",
			path,
			status,
			headers.Get("Content-Encoding"),
		)
	}
}

// TestBuildServiceConfigWiring locks the composition decisions: no fixed
// write deadline (unbounded export/recording streams), the build stamped
// for /version and build_info, the metrics surface opted into the explicit
// unauthenticated mode (auth is the htpasswd middleware or loopback-only
// dev), and the middleware stack grows by auth + the three web middlewares.
func TestBuildServiceConfigWiring(t *testing.T) {
	plain, err := buildServiceConfig("127.0.0.1:0", "", nil, nil)
	if err != nil {
		t.Fatalf("build plain config: %v", err)
	}

	if plain.WriteTimeout != appkit.NoTimeout {
		t.Errorf("WriteTimeout = %v, want NoTimeout (exports and recordings stream unbounded)", plain.WriteTimeout)
	}

	if plain.Version != version {
		t.Errorf("Version = %q, want %q", plain.Version, version)
	}

	if plain.Metrics == nil || !plain.Metrics.AllowUnauthenticated {
		t.Error("Metrics must be enabled with the explicit unauthenticated opt-in")
	}

	if got := len(plain.ExtraMiddlewares); got != 4 {
		t.Fatalf("plain ExtraMiddlewares = %d, want 4 (security headers, timing, compression, etag)", got)
	}

	authed, err := buildServiceConfig("127.0.0.1:0", writeTempHtpasswd(t), nil, nil)
	if err != nil {
		t.Fatalf("build authed config: %v", err)
	}

	if got := len(authed.ExtraMiddlewares); got != 5 {
		t.Errorf(
			"authed ExtraMiddlewares = %d, want 5 (auth first, then security headers, timing, compression, etag)",
			got,
		)
	}
}

// TestBuildServiceConfigCSPNonceWiring covers the env-gated half of the CSP
// rollout: with CSP_NONCE=1 the nonce middleware joins the chain (the
// middleware's own behavior is pinned in internal/web), and the builder's
// policy keeps the inline-style escape hatch and framing lockout.
func TestBuildServiceConfigCSPNonceWiring(t *testing.T) {
	t.Setenv("CSP_NONCE", "1")

	nonced, err := buildServiceConfig("127.0.0.1:0", "", nil, nil)
	if err != nil {
		t.Fatalf("build nonced config: %v", err)
	}

	if got := len(nonced.ExtraMiddlewares); got != 5 {
		t.Errorf(
			"CSP_NONCE=1 ExtraMiddlewares = %d, want 5 (nonce first, then security headers, timing, compression, etag)",
			got,
		)
	}

	csp := nonceCSP("abc123")

	for _, want := range []string{"'nonce-abc123'", "style-src 'self' 'unsafe-inline'", "frame-ancestors 'none'"} {
		if !strings.Contains(csp, want) {
			t.Errorf("nonceCSP output missing %q: %s", want, csp)
		}
	}

	if _, err := buildServiceConfig("127.0.0.1:0", t.TempDir()+"/absent-htpasswd", nil, nil); err == nil {
		t.Error("buildServiceConfig with a missing htpasswd file must fail, want load error")
	}
}

// TestSecurityHeadersContract pins the P1-T5/T6 hardening: every response
// carries Referrer-Policy: no-referrer, X-Content-Type-Options: nosniff, and
// X-Robots-Tag: noindex — the HTML page, the 304 replay, a static asset, and
// a streaming export alike (the meta robots tag only ever covered HTML).
// The appkit default's X-Frame-Options: DENY must survive the inner
// overwrite, so the composition keeps its framing lockout.
func TestSecurityHeadersContract(t *testing.T) {
	sys, cleanup, err := cli.OpenSystem(t.Context(), "")
	if err != nil {
		t.Fatalf("open memory index: %v", err)
	}

	defer cleanup()

	serviceConfig, err := buildServiceConfig("127.0.0.1:0", "", nil, nil)
	if err != nil {
		t.Fatalf("build service config: %v", err)
	}

	svc, err := appkit.NewService(serviceConfig)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	web.Register(svc.Mux, sys, web.Options{})

	ts := testkit.Serve(t, svc)
	client := noAutoGzip()

	wantHeaders := map[string]string{
		"Referrer-Policy":        securityReferrerPolicy,
		"X-Content-Type-Options": securityHeaderNosniff,
		"X-Robots-Tag":           securityRobotsTagNoindex,
		"X-Frame-Options":        "DENY",
	}

	assertSecurityHeaders := func(when string, headers http.Header) {
		t.Helper()

		for header, want := range wantHeaders {
			if got := headers.Get(header); got != want {
				t.Errorf("%s %s = %q, want %q", when, header, got, want)
			}
		}
	}

	status, page, _ := get(t, client, ts.FullChainURL, "/", nil)
	if status != http.StatusOK {
		t.Fatalf("GET / status = %d, want 200", status)
	}

	assertSecurityHeaders("GET /", page)

	etag := page.Get("ETag")
	if etag == "" {
		t.Fatal("/ served without an ETag; the 304 leg cannot run")
	}

	status, replay, _ := get(t, client, ts.FullChainURL, "/", map[string]string{"If-None-Match": etag})
	if status != http.StatusNotModified {
		t.Fatalf("conditional replay status = %d, want 304", status)
	}

	assertSecurityHeaders("304 replay", replay)

	for _, path := range []string{"/app.css", "/api/export.csv", "/api/export.jsonl"} {
		status, headers, _ := get(t, client, ts.FullChainURL, path, nil)
		if status != http.StatusOK {
			t.Fatalf("GET %s status = %d, want 200", path, status)
		}

		assertSecurityHeaders(path, headers)
	}
}

// TestCacheSemanticsHardening pins the conditional-GET corner cases the
// 2026-10-15 hardening pass owed the HTTP-stack doctrine (05-40 M11):
// content negotiation must Vary, query strings must never cross-revalidate,
// HEAD rides the same tags, and buffered renders carry exact lengths.
//
// Flake triage (2026-10-18, pareto T6): the one CI failure on record
// (2026-10-16 era, 5.06s duration) is the upstream testkit.Serve start
// deadline — go-appkit's startTimeout = 5s with a 1ms poll Fatals when
// saturation delays the listener start; nothing in this test depends on
// wall-clock time. Same saturation class go-appkit already outgrew on
// leakSettleTimeout (5s→10s, proven twice). Disposition: green in every
// run since, 50x -race local loop under load 140-200 green, upstream
// filing drafted (docs/drafts/go-appkit-testkit-start-timeout-saturation.md).
func TestCacheSemanticsHardening(t *testing.T) {
	sys, cleanup, err := cli.OpenSystem(t.Context(), "")
	if err != nil {
		t.Fatalf("open memory index: %v", err)
	}

	defer cleanup()

	serviceConfig, err := buildServiceConfig("127.0.0.1:0", "", nil, nil)
	if err != nil {
		t.Fatalf("build service config: %v", err)
	}

	svc, err := appkit.NewService(serviceConfig)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	web.Register(svc.Mux, sys, web.Options{})

	ts := testkit.Serve(t, svc)
	client := noAutoGzip()

	// Vary: the compression layer sits OUTER of the ETag layer, so tags
	// hash identity bodies — but any cache between here and the browser
	// must still be told the representation varies with Accept-Encoding,
	// on the 200 AND on the 304.
	status, headers, _ := get(
		t, client, ts.FullChainURL, "/frag/dashboard",
		map[string]string{"Accept-Encoding": "gzip"},
	)
	if status != http.StatusOK {
		t.Fatalf("GET /frag/dashboard status = %d, want 200", status)
	}

	if vary := headers.Get("Vary"); !strings.Contains(vary, "Accept-Encoding") {
		t.Errorf("200 Vary = %q, want it to name Accept-Encoding", vary)
	}

	etag := headers.Get("ETag")
	if etag == "" {
		t.Fatal("fragment served without an ETag")
	}

	status, headers304, _ := get(
		t, client, ts.FullChainURL, "/frag/dashboard",
		map[string]string{"If-None-Match": etag, "Accept-Encoding": "gzip"},
	)
	if status != http.StatusNotModified {
		t.Fatalf("gzip conditional replay status = %d, want 304", status)
	}

	if vary := headers304.Get("Vary"); !strings.Contains(vary, "Accept-Encoding") {
		t.Errorf("304 Vary = %q, want it to name Accept-Encoding (a shared cache must re-key on encoding)", vary)
	}

	// Identity replay with the SAME tag: because the tag hashes the
	// identity body (compression is outer), an identity client revalidates
	// too — the ordering doctrine's whole point.
	status, _, body := get(
		t, client, ts.FullChainURL, "/frag/dashboard",
		map[string]string{"If-None-Match": etag},
	)
	if status != http.StatusNotModified {
		t.Errorf("identity conditional replay status = %d, want 304 (tags hash identity bodies)", status)
	}

	if body != "" {
		t.Errorf("identity 304 carried %d body bytes, want empty", len(body))
	}

	// Query strings must never cross-revalidate: the filtered page is a
	// different document, and /'s tag must not stand in for it.
	_, rootHeaders, _ := get(t, client, ts.FullChainURL, "/", nil)
	rootTag := rootHeaders.Get("ETag")

	if rootTag == "" {
		t.Fatal("/ served without an ETag")
	}

	status, _, _ = get(
		t, client, ts.FullChainURL, "/?direction=inbound",
		map[string]string{"If-None-Match": rootTag},
	)
	if status != http.StatusOK {
		t.Errorf("filtered page replayed with /'s tag = %d, want 200 (distinct documents, no false 304)", status)
	}

	// HEAD rides the same validators: a conditional HEAD answers 304 with
	// no body, so a HEAD-based monitor gets the same revalidation economy.
	req, err := http.NewRequestWithContext(t.Context(), http.MethodHead, ts.FullChainURL+"/frag/dashboard", nil)
	if err != nil {
		t.Fatalf("build HEAD request: %v", err)
	}

	req.Header.Set("If-None-Match", etag)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("HEAD /frag/dashboard: %v", err)
	}

	defer func() { _ = resp.Body.Close() }()

	headBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusNotModified || len(headBody) != 0 {
		t.Errorf("conditional HEAD = %d with %d body bytes, want 304 with none", resp.StatusCode, len(headBody))
	}

	// Buffered renders carry an exact Content-Length: render() writes the
	// whole page in one Write, and a missing/_chunked length would break
	// progress UX on slow LAN links for zero streaming benefit.
	status, pageHeaders, pageBody := get(t, client, ts.FullChainURL, "/", nil)
	if status != http.StatusOK {
		t.Fatalf("GET / status = %d, want 200", status)
	}

	if cl := pageHeaders.Get("Content-Length"); cl == "" || cl != strconv.Itoa(len(pageBody)) {
		t.Errorf("/ Content-Length = %q, want %d (buffered render, single write)", cl, len(pageBody))
	}

	// Trailing-slash policy is intentionally unpinned here: the owning
	// pass is still deciding between a canonical redirect and the mux's
	// exact-pattern 404 (05-40 M11) — pin it once that settles.
}

// benchRoundTrip issues one GET through the served chain and drains the
// body, returning the status. Separate from get() because the poll probe
// cares about server-side cost, so the client path must stay allocation-thin
// (drain, not materialize).
func benchRoundTrip(b *testing.B, client *http.Client, url, etag string) int {
	b.Helper()

	req, err := http.NewRequestWithContext(b.Context(), http.MethodGet, url, nil)
	if err != nil {
		b.Fatal(err)
	}

	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}

	resp, err := client.Do(req)
	if err != nil {
		b.Fatal(err)
	}

	defer func() { _ = resp.Body.Close() }()

	if _, err := io.Copy(io.Discard, resp.Body); err != nil {
		b.Fatal(err)
	}

	return resp.StatusCode
}

// BenchmarkConcurrentPoll turns the R9 poll profile (/frag/dashboard every
// 30s, htmx, one browser per viewer) into CPU ceiling numbers: parallel
// conditional GETs through the real middleware chain, steady state being a
// 304 re-hash-and-compare per poll. Run with:
//
//	GOEXPERIMENT=jsonv2 go test ./cmd/pbx-lens -bench BenchmarkConcurrentPoll -benchmem
func BenchmarkConcurrentPoll(b *testing.B) {
	sys, cleanup, err := cli.OpenSystem(b.Context(), "")
	if err != nil {
		b.Fatalf("open memory index: %v", err)
	}

	defer cleanup()

	serviceConfig, err := buildServiceConfig("127.0.0.1:0", "", nil, nil)
	if err != nil {
		b.Fatalf("build service config: %v", err)
	}

	svc, err := appkit.NewService(serviceConfig)
	if err != nil {
		b.Fatalf("new service: %v", err)
	}

	web.Register(svc.Mux, sys, web.Options{})

	ts := testkit.Serve(b, svc)
	client := noAutoGzip()

	_, headers, _ := get(b, client, ts.FullChainURL, "/frag/dashboard", nil)

	tag := headers.Get("ETag")

	if tag == "" {
		b.Fatal("fragment served without an ETag; probe cannot run")
	}

	for _, bb := range []struct {
		name string
		etag string
	}{
		{name: "revalidate-304", etag: tag},
		{name: "changed-full-200", etag: ""},
	} {
		b.Run(bb.name, func(b *testing.B) {
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					benchRoundTrip(b, client, ts.FullChainURL+"/frag/dashboard", bb.etag)
				}
			})
		})
	}
}

// writeTempHtpasswd materializes a one-user bcrypt htpasswd file (the
// middleware loads entries eagerly at construction).
func writeTempHtpasswd(t *testing.T) string {
	t.Helper()

	hash, err := bcrypt.GenerateFromPassword([]byte("test-secret"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	path := filepath.Join(t.TempDir(), "htpasswd")
	if err := os.WriteFile(path, []byte("tester:"+string(hash)+"\n"), 0o600); err != nil {
		t.Fatalf("write htpasswd: %v", err)
	}

	return path
}
