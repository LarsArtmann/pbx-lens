package web

import (
	"net/http"
	"strings"
	"testing"
)

// TestShellCarriesGlobalErrorHandling pins the polled-region failure
// contract: every shell page ships the library's toast container (the
// visible error affordance), the sr-only aria-live announcer, and the
// GlobalErrorHandling script that turns htmx send/response errors into
// feedback. Without this wiring a dead poller leaves silently stale
// numbers on screen. The noscript path is untouched by construction —
// all three elements are inert without JS.
func TestShellCarriesGlobalErrorHandling(t *testing.T) {
	mux := newTestServer(t)

	for _, path := range []string{"/", "/search", "/dashboard"} {
		status, body := getBody(t, mux, path)
		if status != http.StatusOK {
			t.Fatalf("GET %s status = %d, want 200", path, status)
		}

		for _, want := range []string{
			`id="tc-toast-container"`,          // feedback.ToastContainer — visible toasts
			`id="tc-error-announcer"`,          // GlobalErrorHandling announcer (aria-live)
			"htmx:sendError",                   // network-level failures (server gone mid-poll)
			"htmx:responseError",               // 4xx/5xx responses on polled fragments
			"aria-live=\"polite\"",             // announced accessibly
			`src="/static/htmx.min.js"`,        // runtime the listeners bind to
		} {
			if !strings.Contains(body, want) {
				t.Errorf("%s missing %q", path, want)
			}
		}
	}
}
