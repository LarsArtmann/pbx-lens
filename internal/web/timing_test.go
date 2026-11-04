package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestServerTimingHeaderOnPlainRoute pins the middleware's every-route
// contract: even a handler that records no named metrics must respond with
// a Server-Timing header (the wrapper flushes its `total` at the first
// write). This is the guard that keeps the doctrine true — if the wiring
// ever loses the header, every timing claim in docs/http-stack.md is dead.
func TestServerTimingHeaderOnPlainRoute(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/health/live", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	timed := ServerTimingMiddleware()(mux)

	rec := httptest.NewRecorder()
	timed.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health/live", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	timing := rec.Header().Get("Server-Timing")
	if timing == "" {
		t.Fatal("Server-Timing header missing on a plain route — middleware must instrument every route")
	}

	if !strings.Contains(timing, "total") {
		t.Errorf("Server-Timing = %q, want the total metric", timing)
	}
}
