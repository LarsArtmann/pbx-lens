package web

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestETagMiddlewareCounters proves the hooks observe every outcome the
// middleware can produce: a fresh tag on the first response, a 304 on the
// conditional replay, and the derived hit ratio. The replay also re-fires
// OnETagGenerated (go-etag resolves the tag on 304s too), which is why
// ETagGenerated doubles as the tag-resolved request count.
func TestETagMiddlewareCounters(t *testing.T) {
	stats := NewCacheCounters()

	handler := ETagMiddleware(stats, nil)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("stable body"))
	}))

	first := httptest.NewRecorder()
	handler.ServeHTTP(first, httptestRequest(t, "/"))

	if first.Code != http.StatusOK {
		t.Fatalf("first response status = %d, want 200", first.Code)
	}

	tag := first.Header().Get("ETag")
	if tag == "" {
		t.Fatal("first response missing ETag")
	}

	replay := httptest.NewRecorder()
	conditional := httptestRequest(t, "/")
	conditional.Header.Set("If-None-Match", tag)
	handler.ServeHTTP(replay, conditional)

	if replay.Code != http.StatusNotModified {
		t.Fatalf("conditional replay status = %d, want 304", replay.Code)
	}

	snapshot := stats.Snapshot()

	if snapshot.ETagGenerated != 2 {
		t.Errorf("ETagGenerated = %d, want 2 (200 and its 304 replay both resolve a tag)", snapshot.ETagGenerated)
	}

	if snapshot.NotModified != 1 {
		t.Errorf("NotModified = %d, want 1", snapshot.NotModified)
	}

	if snapshot.HitRatio != 0.5 {
		t.Errorf("HitRatio = %v, want 0.5", snapshot.HitRatio)
	}

	if snapshot.BufferOverflows != 0 {
		t.Errorf("BufferOverflows = %d, want 0", snapshot.BufferOverflows)
	}
}

// TestETagMiddlewareOverflowWarns pins the degradation signal: a body past
// the buffer streams without an ETag, the counter ticks once, and the log
// hook receives the warning with the limit that was exceeded.
func TestETagMiddlewareOverflowWarns(t *testing.T) {
	stats := NewCacheCounters()

	var warnedMsg string
	var warnedArgs []any

	handler := ETagMiddleware(stats, func(msg string, args ...any) {
		warnedMsg = msg
		warnedArgs = args
	})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(bytes.Repeat([]byte("a"), (1<<20)+16))
	}))

	overflow := httptest.NewRecorder()
	handler.ServeHTTP(overflow, httptestRequest(t, "/"))

	if overflow.Code != http.StatusOK {
		t.Fatalf("overflow response status = %d, want 200", overflow.Code)
	}

	if overflow.Header().Get("ETag") != "" {
		t.Error("overflow response carries an ETag; want untagged streaming")
	}

	snapshot := stats.Snapshot()

	if snapshot.BufferOverflows != 1 {
		t.Errorf("BufferOverflows = %d, want 1", snapshot.BufferOverflows)
	}

	if snapshot.ETagGenerated != 0 {
		t.Errorf("ETagGenerated = %d, want 0 (overflowed bodies get no tag)", snapshot.ETagGenerated)
	}

	if warnedMsg == "" {
		t.Fatal("overflow did not warn")
	}

	if len(warnedArgs) != 2 || warnedArgs[0] != "limit_bytes" || warnedArgs[1] != 1<<20 {
		t.Errorf("overflow warning args = %v, want [limit_bytes %d] (the configured limit)", warnedArgs, 1<<20)
	}
}

// TestSkipBodyCaching locks the ETag Skip policy: the two exports and the
// recording proxy stream unbounded (the body buffer must never engage),
// and the staleness fragment plus freshness probe embed an age that changes
// on every response, so a body-hash tag could never revalidate.
func TestSkipBodyCaching(t *testing.T) {
	cases := []struct {
		path string
		skip bool
	}{
		{"/", false},
		{"/dashboard", false},
		{"/frag/dashboard", false},
		{"/search", false},
		{"/calls/telnyx-conversation/conv-a", false},
		{"/app.css", false},
		{"/static/htmx.min.js", false},
		{"/favicon.svg", false},
		{"/api/ops", false},
		{"/frag/staleness", true},
		{"/api/freshness", true},
		{"/api/export.jsonl", true},
		{"/api/export.csv", true},
		{"/calls/telnyx-conversation/conv-a/recording", true},
	}

	for _, tc := range cases {
		r := httptestRequest(t, tc.path)

		if got := skipBodyCaching(r); got != tc.skip {
			t.Errorf("skipBodyCaching(%q) = %v, want %v", tc.path, got, tc.skip)
		}
	}
}

func httptestRequest(t *testing.T, path string) *http.Request {
	t.Helper()

	return httptest.NewRequest(http.MethodGet, path, nil)
}
