package web

import (
	"encoding/csv"
	"net/http"
	"strconv"
	"time"

	"encoding/json/v2"

	"github.com/larsartmann/pbx-lens/internal/calls"
)

// exportColumns is the CSV header in emission order. The names mirror the
// JSONL field names so a JSONL consumer and a CSV consumer read the same
// record shape.
var exportColumns = []string{
	"kind", "native_id", "started_at", "from_number", "to_number",
	"direction", "duration_secs", "is_ai", "hangup_cause", "recording_ref",
	"cost_micros", "cost_currency", "insight_summary", "transcript_excerpt",
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

	if err := writer.Write(exportColumns); err != nil {
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
// error response itself and reports whether the caller may proceed.
func (s *server) exportViews(w http.ResponseWriter, r *http.Request) ([]calls.CallView, bool) {
	query, err := parseFilters(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return nil, false
	}

	views, err := ExportCalls(r.Context(), s.sys, query)
	if err != nil {
		http.Error(w, "reading the call index failed: "+err.Error(), http.StatusInternalServerError)

		return nil, false
	}

	return views, true
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
