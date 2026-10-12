package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/pbx-lens/internal/calls"
)

// TestSearchHitCarriesQuery pins the T21 hop: search hit hrefs carry ?q= so
// the detail page can mark the matched turns.
func TestSearchHitCarriesQuery(t *testing.T) {
	started := time.Date(2026, 10, 9, 19, 51, 25, 0, time.UTC)
	mux := newTestServerWithSearch(t, webObservation("conv-out", started))

	_, body := getBody(t, mux, "/search?q=hello")
	if !strings.Contains(body, `href="/calls/telnyx-conversation/conv-out?q=hello"`) {
		t.Errorf("search hit href missing ?q= deep link, got: %s", extractHrefs(body))
	}
}

// TestDetailQueryRedirectLandsAtFirstMatch pins the redirect: ?q= with a
// matching turn 302s once to the same URL with at=<n>#turn-<n>; the at
// param breaks the loop (the second request renders).
func TestDetailQueryRedirectLandsAtFirstMatch(t *testing.T) {
	started := time.Date(2026, 10, 9, 19, 51, 25, 0, time.UTC)
	mux := newTestServerWithSearch(t, webObservation("conv-out", started))

	recorder := httptestGet(t, mux, "/calls/telnyx-conversation/conv-out?q=help")
	if recorder.Code != http.StatusFound {
		t.Fatalf("detail?q=help status = %d, want 302", recorder.Code)
	}

	location := recorder.Header().Get("Location")
	want := "/calls/telnyx-conversation/conv-out?q=help&at=1#turn-1"
	if location != want {
		t.Errorf("redirect Location = %q, want %q", location, want)
	}

	// The redirected request (fragment stripped by the browser) renders.
	status, body := getBody(t, mux, "/calls/telnyx-conversation/conv-out?q=help&at=1")
	if status != http.StatusOK {
		t.Fatalf("redirected detail status = %d, want 200", status)
	}

	if !strings.Contains(body, `id="turn-1"`) {
		t.Error("matched turn missing its anchor id")
	}

	if !strings.Contains(body, "bg-amber-100") {
		t.Error("matched turn missing the highlight mark")
	}

	if strings.Contains(body, `id="turn-0"`) {
		t.Error("unmatched turn must not carry an anchor id")
	}
}

// TestDetailQueryNoMatchRendersUnmarked pins the no-match path: the page
// renders without redirect and without marks (FTS may match on fields the
// substring rule cannot see — honest, not broken).
func TestDetailQueryNoMatchRendersUnmarked(t *testing.T) {
	started := time.Date(2026, 10, 9, 19, 51, 25, 0, time.UTC)
	mux := newTestServerWithSearch(t, webObservation("conv-out", started))

	recorder := httptestGet(t, mux, "/calls/telnyx-conversation/conv-out?q=zzz-nothing")
	if recorder.Code != http.StatusOK {
		t.Fatalf("detail?q=zzz status = %d, want 200 (no redirect)", recorder.Code)
	}

	_, body := getBody(t, mux, "/calls/telnyx-conversation/conv-out?q=zzz-nothing")
	if strings.Contains(body, "bg-amber-100") || strings.Contains(body, `id="turn-`) {
		t.Error("non-matching query must not mark or anchor turns")
	}
}

// TestTurnMatches unit-tests the highlight rule: all terms, case-insensitive,
// substring (partial numbers and mid-word snippets included).
func TestTurnMatches(t *testing.T) {
	turn := calls.TranscriptTurn{Role: calls.TurnRoleUser, Text: "Woolly invoice for +17287289311"}

	cases := []struct {
		query string
		want  bool
	}{
		{"woolly", true},
		{"WOOLLY", true},
		{"2893", true},
		{"woolly invoice", true},
		{"woolly zzz", false},
		{"", false},
		{"   ", false},
	}

	for _, tc := range cases {
		if got := turnMatches(turn, tc.query); got != tc.want {
			t.Errorf("turnMatches(%q) = %v, want %v", tc.query, got, tc.want)
		}
	}
}

// TestDetailWithoutQueryKeepsPlainPath pins the default path: no ?q= means
// no redirect, no marks, and the transcript card anchor (the fallback
// target) stands in both paths.
func TestDetailWithoutQueryKeepsPlainPath(t *testing.T) {
	started := time.Date(2026, 10, 9, 19, 51, 25, 0, time.UTC)
	mux := newTestServerWithSearch(t, webObservation("conv-out", started))

	recorder := httptestGet(t, mux, "/calls/telnyx-conversation/conv-out")
	if recorder.Code != http.StatusOK {
		t.Fatalf("plain detail status = %d, want 200 (no redirect)", recorder.Code)
	}

	_, body := getBody(t, mux, "/calls/telnyx-conversation/conv-out")
	if !strings.Contains(body, "tc-log-line") {
		t.Error("plain detail lost its transcript")
	}

	if !strings.Contains(body, `id="transcript"`) {
		t.Error("transcript card anchor missing")
	}

	if strings.Contains(body, "bg-amber-100") {
		t.Error("plain detail must not render match marks")
	}
}

func httptestGet(t *testing.T, mux *http.ServeMux, path string) *httptest.ResponseRecorder {
	t.Helper()

	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))

	return recorder
}

func extractHrefs(body string) string {
	var hrefs []string

	for chunk := range strings.SplitSeq(body, "href=\"") {
		if end := strings.Index(chunk, "\""); end >= 0 {
			hrefs = append(hrefs, chunk[:end])
		}
	}

	return strings.Join(hrefs, " ")
}
