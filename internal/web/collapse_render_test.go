package web

import (
	"strings"
	"testing"

	"github.com/larsartmann/pbx-lens/internal/query/querytest"
)

// TestListRendersCollapsedRowOnce pins the representative-row collapse
// through the RENDERED list (the query-side collapse math is pinned in
// internal/query; this is the page's honest-one-row contract).
func TestListRendersCollapsedRowOnce(t *testing.T) {
	t.Parallel()

	started := querytest.CallStarted()
	cdr, transcript := querytest.AICallPair("uuid-ai-1", started)

	mux := newTestServer(t, cdr, transcript)

	_, body := getBody(t, mux, "/")
	if got := strings.Count(body, "/calls/freeswitch-cdr/uuid-ai-1"); got != 1 {
		t.Errorf("CDR detail links = %d, want 1", got)
	}

	if strings.Contains(body, "/calls/agent-transcript/uuid-ai-1") {
		t.Error("superseded transcript row must not render")
	}

	if !strings.Contains(body, "hello there") {
		t.Error("collapsed row should carry the transcript's opening line")
	}
}
