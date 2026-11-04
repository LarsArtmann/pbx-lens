package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	servertiming "github.com/larsartmann/httputil/server_timing"
)

// TestOpsAPICarriesRetentionTiming pins the stop-ordering contract at
// collectRetention (ops.go): the deferred stop is correct only while the
// function fully returns before opsHandler writes the JSON — the
// Server-Timing header flushes at that first body write, so a reorder that
// writes earlier (or streams the payload) silently drops the retention
// metric and this test goes red.
func TestOpsAPICarriesRetentionTiming(t *testing.T) {
	mux := servertiming.ServerTimingMiddleware()(newTestServerWith(t, Options{
		Ops:             opsPollerPayload(),
		Cache:           opsCounters(t),
		RetentionMaxAge: 24 * time.Hour,
	}))

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/ops", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/ops status = %d, want 200 (body: %s)", rec.Code, rec.Body.String())
	}

	timing := rec.Header().Get("Server-Timing")

	if !strings.Contains(timing, "retention") {
		t.Errorf("Server-Timing = %q — the retention metric was dropped, "+
			"the ops JSON wrote before collectRetention's stop fired", timing)
	}
}
