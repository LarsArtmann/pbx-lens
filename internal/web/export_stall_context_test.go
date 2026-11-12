package web

import (
	"encoding/json/v2"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/pbx-lens/internal/calls"
)

// TestExportStallMarkerCarriesStreamContext pins plan A18: the stalled walk
// attaches its cursor and the colliding stream ids AT CONSTRUCTION, so the
// terminal marker (and the log line) say WHICH calls share the timestamp —
// not just that a stall happened.
func TestExportStallMarkerCarriesStreamContext(t *testing.T) {
	saved := exportBatchSize
	exportBatchSize = 2

	t.Cleanup(func() { exportBatchSize = saved })

	same := time.Date(2026, 10, 9, 19, 0, 0, 0, time.UTC)
	tied := []calls.Observation{
		webObservation("tied-a", same),
		webObservation("tied-b", same),
		webObservation("tied-c", same),
	}

	mux := newTestServer(t, tied...)

	status, body := getBody(t, mux, "/api/export.jsonl")
	if status != http.StatusOK {
		t.Fatalf("jsonl export status = %d, want 200", status)
	}

	markerLine := ""

	for line := range strings.SplitSeq(strings.TrimSpace(body), "\n") {
		if strings.Contains(line, exportIncompleteCode) {
			markerLine = line
		}
	}

	if markerLine == "" {
		t.Fatalf("no %s marker in the stream, got:\n%s", exportIncompleteCode, body)
	}

	var marker exportError

	if err := json.Unmarshal([]byte(markerLine), &marker); err != nil {
		t.Fatalf("marker line is not JSON (%v): %s", err, markerLine)
	}

	if marker.Context == nil {
		t.Fatalf("marker carries no context map — the stall error was built without WithContext: %s", markerLine)
	}

	if marker.Context["cursor"] == "" {
		t.Errorf("context missing the cursor: %v", marker.Context)
	}

	var ids []string

	// The stalled PAGE's members are the witness (batch size 2): tied-c is
	// the probe row one past the page and legitimately absent from the
	// context — the contract is "the colliding page is named".
	for _, id := range []string{"tied-a", "tied-b"} {
		if strings.Contains(marker.Context["stream_ids"], id) {
			ids = append(ids, id)
		}
	}

	if len(ids) != 2 {
		t.Errorf("context stream_ids must name the stalled page's calls, got %v in %v", ids, marker.Context)
	}
}
