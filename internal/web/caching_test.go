package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

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
