package web

import (
	"fmt"
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

// TestExportMemoryBudgetStreamsPastOldCap pins the R7 budget (plan T10.2):
// the stream walks the keyset page by page and must deliver EVERY matching
// row in started_at order — no maxRows cap may reappear, and the walk does
// not stop at the first page. The shrunken batch makes this a multi-page
// walk over a tiny corpus deterministically.
func TestExportMemoryBudgetStreamsPastOldCap(t *testing.T) {
	saved := exportBatchSize
	exportBatchSize = 2

	t.Cleanup(func() { exportBatchSize = saved })

	base := time.Date(2026, 10, 9, 19, 0, 0, 0, time.UTC)

	seeded := make([]calls.Observation, 0, 5)

	for i := range 5 {
		seeded = append(seeded, webObservation(fmt.Sprintf("row-%d", i), base.Add(time.Duration(i)*time.Minute)))
	}

	mux := newTestServer(t, seeded...)

	status, body := getBody(t, mux, "/api/export.csv")
	if status != http.StatusOK {
		t.Fatalf("csv export status = %d, want 200", status)
	}

	for i := range 5 {
		if !strings.Contains(body, fmt.Sprintf("row-%d", i)) {
			t.Errorf("row-%d missing from the stream (cap or page-walk bug)", i)
		}
	}

	if strings.Contains(body, exportIncompleteCode) {
		t.Error("distinct timestamps must never stall the cursor")
	}

	if got := strings.Count(body, "\n"); got != 6 { // header + 5 rows
		t.Errorf("csv line count = %d, want 6 (header + every row)", got)
	}
}

// TestExportHonestyNoteVisibleOnListPage pins the operator-facing surface
// (plan T10.3): the stall semantics render as visible text under the table
// (the button tooltip hides), naming the marker code and the stall class.
func TestExportHonestyNoteVisibleOnListPage(t *testing.T) {
	mux := newTestServer(t, webObservation("conv-out", testCallStarted()))

	_, body := getBody(t, mux, "/")

	for _, want := range []string{"EXPORT_INCOMPLETE", "lens.web.export_cursor_stalled", "no row cap"} {
		if !strings.Contains(body, want) {
			t.Errorf("export honesty note missing %q on the list page", want)
		}
	}
}
