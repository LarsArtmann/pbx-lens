package web

import (
	"encoding/csv"
	"encoding/json/v2"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/larsartmann/pbx-lens/internal/calls"
)

// exportHeader is the CSV header in emission order. The column names reuse
// the JSONL field names; the JSONL rows are the full view and additionally
// carry transcript, mos, and content_hash, which CSV omits to stay flat.
func exportHeader() []string {
	return []string{
		"kind", "native_id", "started_at", "from_number", "to_number",
		"direction", "duration_secs", "is_ai", "hangup_cause", "recording_ref",
		"cost_micros", "cost_currency", "insight_summary", "transcript_excerpt",
	}
}

// exportJSONL streams the filtered call index as newline-delimited JSON
// (D5). One JSON object per line, newest first, so any consumer can tail or
// pipe it.
func (s *server) exportJSONL(w http.ResponseWriter, r *http.Request) {
	views, ok := s.exportViews(w, r)
	if !ok {
		return
	}

	w.Header().Set("Content-Type", "application/x-ndjson; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="pbx-lens-calls.jsonl"`)

	for _, view := range views {
		line, err := json.Marshal(view)
		if err != nil {
			return
		}

		//nolint:gosec // JSON-encoded payload served as an ndjson attachment; never rendered as HTML
		if _, err := w.Write(append(line, '\n')); err != nil {
			return
		}
	}
}

// exportCSV streams the filtered call index as CSV (D5).
func (s *server) exportCSV(w http.ResponseWriter, r *http.Request) {
	views, ok := s.exportViews(w, r)
	if !ok {
		return
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="pbx-lens-calls.csv"`)

	writer := csv.NewWriter(w)

	if err := writer.Write(exportHeader()); err != nil {
		return
	}

	for _, view := range views {
		if err := writer.Write(exportRecord(view)); err != nil {
			return
		}
	}

	writer.Flush()
}

// exportViews parses the shared filters and loads the rows; it writes the
// error response itself and reports whether the caller may proceed. The
// optional pii=hash mode swaps participants for their participant hashes
// before anything leaves the server.
func (s *server) exportViews(w http.ResponseWriter, r *http.Request) ([]calls.CallView, bool) {
	query, err := parseFilters(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return nil, false
	}

	switch r.URL.Query().Get("pii") {
	case "", piiModeHash:
	default:
		http.Error(
			w,
			fmt.Sprintf(
				"unknown pii mode %q: the only supported mode is pii=%s",
				r.URL.Query().Get("pii"),
				piiModeHash,
			),
			http.StatusBadRequest,
		)

		return nil, false
	}

	views, err := ExportCalls(r.Context(), s.sys, query)
	if err != nil {
		http.Error(w, "reading the call index failed: "+err.Error(), http.StatusInternalServerError)

		return nil, false
	}

	if r.URL.Query().Get("pii") == piiModeHash {
		for i := range views {
			views[i].FromNumber = piiParticipant(views[i].FromNumber)
			views[i].ToNumber = piiParticipant(views[i].ToNumber)
		}
	}

	return views, true
}

// piiModeHash names the participant-hash-only export mode (T23): numbers
// render as sha256:<16 hex chars>, everything else as a redaction marker.
const piiModeHash = "hash"

// piiRedacted is the honest marker for a party that was never a parseable
// number: real CDRs carry unknown or garbled parties, and free text must not
// pass through a PII-limited export pretending to be a number.
const piiRedacted = "[redacted]"

func piiParticipant(number string) string {
	if calls.E164(number).Valid() {
		return "sha256:" + string(calls.E164(number).Hash())
	}

	return piiRedacted
}

// exportRecord flattens one view into the CSV column order.
func exportRecord(view calls.CallView) []string {
	return []string{
		string(view.Kind),
		string(view.NativeID),
		view.StartedAt.UTC().Format(time.RFC3339),
		view.FromNumber,
		view.ToNumber,
		string(view.Direction),
		strconv.Itoa(view.DurationSecs),
		strconv.FormatBool(view.IsAI),
		view.HangupCause,
		view.RecordingRef,
		strconv.FormatInt(view.CostMicros, 10),
		view.CostCurrency,
		view.InsightSummary,
		view.TranscriptExcerpt,
	}
}
