package web

import (
	"encoding/csv"
	"encoding/json/v2"
	"fmt"
	"net/http"
	"strconv"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
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

// beginExportResponse is the shared preamble of both export formats: the
// request gate, the X-Pii-Mode response header, and the format's own
// Content-Type/Disposition pair. A false return already wrote the error
// response.
func (s *server) beginExportResponse(
	w http.ResponseWriter,
	r *http.Request,
	contentType string,
	disposition string,
) (ListCallsQuery, PIIMode, bool) {
	query, pii, ok := s.beginExport(w, r)
	if !ok {
		return query, pii, false
	}

	w.Header().Set("X-Pii-Mode", piiHeaderValue(pii))
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", disposition)

	return query, pii, true
}

// exportJSONL streams the filtered call index as newline-delimited JSON
// (D5). One JSON object per line, newest first, so any consumer can tail or
// pipe it.
func (s *server) exportJSONL(w http.ResponseWriter, r *http.Request) {
	query, pii, ok := s.beginExportResponse(
		w,
		r,
		"application/x-ndjson; charset=utf-8",
		`attachment; filename="pbx-lens-calls.jsonl"`,
	)
	if !ok {
		return
	}

	markFirstPage := newFirstPageMetric(r.Context())

	err := StreamExportCalls(r.Context(), s.sys, query, func(view calls.CallView) error {
		markFirstPage.mark()
		applyPII(pii, &view)

		line, err := json.Marshal(view)
		if err != nil {
			return err
		}

		_, err = w.Write(append(line, '\n')) //nolint:gosec // JSON-encoded ndjson attachment; never rendered as HTML

		return err
	})
	markFirstPage.mark()
	if !markFirstPage.fired() {
		w.WriteHeader(http.StatusOK) // zero-row stream: nothing else commits the header
	}

	if err != nil {
		// The body may already be streaming, so the only honest moves are a
		// terminal marker line in the file (machine-readable) and the log.
		marker, merr := json.Marshal(exportErrorMarker(err))
		if merr == nil {
			marker = append(marker, '\n')
			_, _ = w.Write(marker) //nolint:gosec // JSON-encoded ndjson attachment; never rendered as HTML
		}

		s.logf("export.jsonl stream failed", "err", err.Error())
	}
}

// exportCSV streams the filtered call index as CSV (D5).
func (s *server) exportCSV(w http.ResponseWriter, r *http.Request) {
	query, pii, ok := s.beginExportResponse(
		w,
		r,
		"text/csv; charset=utf-8",
		`attachment; filename="pbx-lens-calls.csv"`,
	)
	if !ok {
		return
	}

	markFirstPage := newFirstPageMetric(r.Context())

	writer := csv.NewWriter(w)

	if err := writer.Write(exportHeader()); err != nil {
		return
	}

	err := StreamExportCalls(r.Context(), s.sys, query, func(view calls.CallView) error {
		markFirstPage.mark()
		applyPII(pii, &view)

		return writer.Write(exportRecord(view))
	})
	markFirstPage.mark()
	if !markFirstPage.fired() {
		w.WriteHeader(http.StatusOK) // zero-row stream: nothing else commits the header
	}

	if err != nil {
		// The body may already be streaming, so the only honest moves are a
		// terminal marker row in the file (visible in any spreadsheet) and
		// the log.
		_ = writer.Write(exportErrorRow(err))

		s.logf("export.csv stream failed", "err", err.Error())
	}

	writer.Flush()
}

// exportIncompleteCode is the marker code the export endpoints emit when a
// stream cannot make progress; exportErrorRow/exportErrorMarker render it
// into the artifact so a truncated download announces itself (plan T10).
const exportIncompleteCode = "EXPORT_INCOMPLETE"

// exportErrorRow renders a stream failure as one terminal CSV row.
func exportErrorRow(err error) []string {
	return []string{"# " + exportIncompleteCode, errorfamily.Code(err), err.Error()}
}

// exportErrorMarker renders a stream failure as one terminal JSONL object.
func exportErrorMarker(err error) exportError {
	return exportError{ExportIncomplete: exportIncompleteCode, Code: errorfamily.Code(err), Message: err.Error()}
}

type exportError struct {
	ExportIncomplete string `json:"export_incomplete"`
	Code             string `json:"code"`
	Message          string `json:"message"`
}

// beginExport parses the shared filters and the pii mode; it writes the
// error response itself and reports whether the caller may proceed.
func (s *server) beginExport(w http.ResponseWriter, r *http.Request) (ListCallsQuery, PIIMode, bool) {
	query, err := parseFilters(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return ListCallsQuery{}, PIIModePlain, false
	}

	pii := PIIMode(r.URL.Query().Get("pii"))
	switch pii {
	case PIIModePlain, PIIModeHash:
	default:
		http.Error(
			w,
			fmt.Sprintf(
				"unknown pii mode %q: the only supported mode is pii=%s",
				string(pii),
				PIIModeHash,
			),
			http.StatusBadRequest,
		)

		return ListCallsQuery{}, PIIModePlain, false
	}

	return query, pii, true
}

// PIIMode selects how participant numbers leave the server in an export.
type PIIMode string

const (
	// PIIModePlain is the unset default: numbers export exactly as stored
	// (the index stores raw numbers by design — see DOMAIN_LANGUAGE.md).
	// The response header renders it explicitly as "plain" so a dump can
	// never masquerade as hashed.
	PIIModePlain PIIMode = ""
	// PIIModeHash is the participant-hash-only export mode (T23): numbers
	// render as sha256:<16 hex chars> (or [redacted] when unparseable).
	// Transcripts, excerpts, and insight summaries stay RAW in this mode —
	// see the export field-coverage matrix in FEATURES.md; pii=hash bounds
	// participant identity, not free text.
	PIIModeHash PIIMode = "hash"
)

// piiHeaderValue renders the X-PII-Mode response value: the empty default
// mode is spelled out, so every export announces exactly what left.
func piiHeaderValue(pii PIIMode) string {
	if pii == PIIModePlain {
		return "plain"
	}

	return string(pii)
}

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

// applyPII rewrites one view in place for the chosen export mode. Hash mode
// covers participant numbers only; free text (transcript, excerpt,
// insight summary) is disclosed raw by the export matrix — the operator
// picks pii=hash knowing exactly what it covers.
func applyPII(pii PIIMode, view *calls.CallView) {
	if pii != PIIModeHash {
		return
	}

	view.FromNumber = piiParticipant(view.FromNumber)
	view.ToNumber = piiParticipant(view.ToNumber)
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
