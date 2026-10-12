package web

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/pbx-lens/internal/calls"
)

// TestExportStallEndsWithVisibleMarker pins plan T10: when the keyset walk
// cannot make progress (one nanosecond group spanning pages), the stream
// still ends honestly — a terminal EXPORT_INCOMPLETE marker inside the
// artifact — instead of a silently truncated download.
func TestExportStallEndsWithVisibleMarker(t *testing.T) {
	saved := exportBatchSize
	exportBatchSize = 2

	t.Cleanup(func() { exportBatchSize = saved })

	// Three calls sharing ONE timestamp: page 1 emits two, page two is all
	// duplicates, the cursor stalls, the stream must announce it.
	same := time.Date(2026, 10, 9, 19, 0, 0, 0, time.UTC)
	tied := []calls.Observation{
		webObservation("tied-a", same),
		webObservation("tied-b", same),
		webObservation("tied-c", same),
	}

	mux := newTestServer(t, tied...)

	status, body := getBody(t, mux, "/api/export.csv")
	if status != http.StatusOK {
		t.Fatalf("csv export status = %d, want 200", status)
	}

	if !strings.Contains(body, exportIncompleteCode) {
		t.Errorf("stalled csv export must end with the %s marker, got:\n%s", exportIncompleteCode, body)
	}

	if !strings.Contains(body, "tied-a") || !strings.Contains(body, "tied-b") {
		t.Error("rows before the stall must still stream")
	}

	status, body = getBody(t, mux, "/api/export.jsonl")
	if status != http.StatusOK {
		t.Fatalf("jsonl export status = %d, want 200", status)
	}

	if !strings.Contains(body, `"export_incomplete":"`+exportIncompleteCode+`"`) {
		t.Errorf("stalled jsonl export must end with the marker object, got:\n%s", body)
	}
}
