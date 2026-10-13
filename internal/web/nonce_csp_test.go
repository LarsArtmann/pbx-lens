package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/larsartmann/httputil"
)

// CSP nonce-mode tests (CSP_NONCE=1 rollout): with the middleware wired,
// every page's inline scripts carry the per-request nonce and the CSP
// header matches; without it, no nonce attributes and no CSP header appear
// (the legacy posture). Generated *_templ.go are gitignored in this repo,
// so the rendered fixtures here are the real markup.

// nonceScriptAttrs counts inline <script> tags (skipping src= externals and
// the split prefix) and asserts each carries the given nonce.
func nonceScriptAttrs(t *testing.T, html, nonce string) {
	t.Helper()

	scripts := 0

	parts := strings.Split(html, "<script")
	for i, part := range parts {
		if i == 0 || strings.HasPrefix(part, " src=") {
			continue
		}

		scripts++
		if !strings.Contains(part[:min(90, len(part))], `nonce="`+nonce+`"`) {
			t.Errorf("inline <script> without nonce=%q: %.90s", nonce, strings.ReplaceAll(part, "\n", " "))
		}
	}

	if scripts == 0 {
		t.Error("expected at least one inline <script> on the page")
	}

	if strings.Contains(html, `nonce=""`) {
		t.Error(`rendered nonce="" — empty nonce must omit the attribute entirely`)
	}
}

func TestNonceModeWired(t *testing.T) {
	mux := newTestServer(t)
	wrapped := httputil.Nonce(httputil.NonceConfig{CSPBuilder: testNonceCSP})(mux)

	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /: status %d", rec.Code)
	}

	csp := rec.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "'nonce-") {
		t.Errorf("CSP header missing nonce: %q", csp)
	}

	nonceScriptAttrs(t, rec.Body.String(), extractNonce(csp))
}

func TestNonceModeOff(t *testing.T) {
	mux := newTestServer(t)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /: status %d", rec.Code)
	}

	if csp := rec.Header().Get("Content-Security-Policy"); csp != "" {
		t.Errorf("nonce mode off must not set a CSP header, got %q", csp)
	}

	html := rec.Body.String()
	if strings.Contains(html, "nonce=") {
		t.Errorf("nonce mode off must render no nonce attributes, got: %.200s", html)
	}
}

// extractNonce pulls the base64url nonce value out of the CSP header.
func extractNonce(csp string) string {
	i := strings.Index(csp, "'nonce-")
	if i < 0 {
		return ""
	}
	rest := csp[i+len("'nonce-"):]
	if j := strings.Index(rest, "'"); j >= 0 {
		return rest[:j]
	}

	return rest
}

func testNonceCSP(nonce string) string {
	return "default-src 'self'; script-src 'self' 'nonce-" + nonce + "'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'; media-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'"
}
