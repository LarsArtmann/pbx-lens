// Package web serves the pbx-lens UI: a read-only surface over the
// call_views read model. Every route is a query; the index DB stays
// disposable and no source ever sees a write from here.
package web

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/larsartmann/pbx-lens/internal/calls"
)

// pageSize caps one rendered page (bounding-tables doctrine: no unbounded
// table ever ships; older calls walk via the keyset cursor).
const pageSize = 50

// ListCallsQuery is the fully-parsed list-page request. Zero fields mean
// unfiltered; Before is the exclusive keyset cursor for the older walk,
// After the exclusive cursor for the newer walk (a page's top row timestamp).
type ListCallsQuery struct {
	Direction calls.Direction
	AI        *bool
	// Number narrows to calls that have this number on EITHER side (the
	// PBX's own number or the counterpart's). It is the segment drill-down:
	// "show me every call with this DID" cannot be a single column filter
	// because the side depends on the call's direction.
	Number string
	From   time.Time
	To     time.Time
	Before time.Time
	After  time.Time
}

// CallRow is the list-page view model — the UI never touches domain types.
type CallRow struct {
	Kind      calls.SourceKind
	NativeID  string
	When      time.Time
	From      string
	To        string
	Direction calls.Direction
	Duration  time.Duration
	IsAI      bool
	Excerpt   string
}

// Href is the stable detail-page path (deterministic stream identity makes
// it deep-link safe).
func (r CallRow) Href() string {
	return fmt.Sprintf("/calls/%s/%s", url.PathEscape(string(r.Kind)), url.PathEscape(r.NativeID))
}

// callRow maps a stored view onto the list's display row.
func callRow(view calls.CallView) CallRow {
	return CallRow{
		Kind:      view.Kind,
		NativeID:  string(view.NativeID),
		When:      view.StartedAt,
		From:      view.FromNumber,
		To:        view.ToNumber,
		Direction: view.Direction,
		Duration:  time.Duration(view.DurationSecs) * time.Second,
		IsAI:      view.IsAI,
		Excerpt:   view.TranscriptExcerpt,
	}
}

// CallListPage is one rendered page plus the cursors for both keyset walks.
type CallListPage struct {
	Rows      []CallRow
	NextOlder string // empty when this page was the last
	NextNewer string // empty when this page is the head of the walk
}

// listScanBatch is one keyset batch of the list walk. It is intentionally
// larger than a page: collapsing a call's duplicate source row drops rows
// from the batch, and a batch that exactly fills the page would otherwise
// cost an extra round trip per dropped row. A var so tests can shrink it and
// walk the batch seams deterministically.
//
//nolint:gochecknoglobals // test-shrinkable batch size (keyset seam tests)
var listScanBatch = 4 * pageSize

// ListCalls runs the query against the read model, newest first, collapsing a
// call's duplicate source rows into its representative row (see
// representative.go) so the pager and the operator both see one call per call.
func ListCalls(ctx context.Context, sys *system.System, query ListCallsQuery) (CallListPage, error) {
	reader := metaengine.NewReader[calls.CallView](sys.MetaEngine(), calls.Collection)

	// Collect one extra row to detect whether an older page exists.
	rows := make([]CallRow, 0, pageSize+1)

	var cursor time.Time

	seen := make(map[string]struct{})

	for {
		// The probe row (one past the batch) proves whether an older row
		// exists at all — a batch that exactly fills itself must not be
		// mistaken for a walk with more to read.
		batch, err := reader.Scan(ctx, listOptionsAt(query, cursor, listScanBatch+1)...)
		if err != nil {
			return CallListPage{}, err
		}

		more := len(batch) > listScanBatch
		if more {
			batch = batch[:listScanBatch]
		}

		start := len(rows)

		for _, view := range batch {
			// The cursor binds inclusively so a timestamp group is never cut
			// off at a batch seam; the tie set skips what this walk already
			// emitted.
			if !cursor.IsZero() && view.StartedAt.Equal(cursor) {
				if _, done := seen[dedupeKey(view)]; done {
					continue
				}
			}

			representative, show, repErr := listRepresentative(ctx, reader, view)
			if repErr != nil {
				return CallListPage{}, repErr
			}

			if !show {
				continue
			}

			rows = append(rows, callRow(representative))

			if len(rows) > pageSize {
				break
			}
		}

		if len(rows) > pageSize || !more {
			break
		}

		next := advanceCursor(cursor, batch, seen)
		if next.Equal(cursor) && len(rows) == start {
			return CallListPage{}, errorfamily.NewRejection(
				"lens.web.list_cursor_stalled",
				"more than one batch of calls share started_at "+rfc3339Nano(cursor)+
					"; narrow the list with from/to filters",
			)
		}

		cursor = next
	}

	page := CallListPage{}

	if len(rows) > pageSize {
		page.Rows = rows[:pageSize]
		page.NextOlder = page.Rows[pageSize-1].When.UTC().Format(time.RFC3339)
	} else {
		page.Rows = rows
	}

	// A newer page exists whenever any row sorts above this page's top row;
	// the cursor back to it is that top row's timestamp. The probe scan
	// carries only the typed filters — both walk cursors bound below the top
	// row and must not narrow the probe.
	if len(page.Rows) > 0 {
		top := page.Rows[0].When

		probe := append(scanFilters(query),
			metaengine.WithFilter("started_at", metaengine.FilterGt, rfc3339Nano(top)),
			metaengine.WithLimit(1),
		)

		newer, err := reader.Scan(ctx, probe...)
		if err != nil {
			return CallListPage{}, err
		}

		if len(newer) > 0 {
			page.NextNewer = top.UTC().Format(time.RFC3339)
		}
	}

	return page, nil
}

// exportBatchSize is one keyset page of a streaming export (R7): memory
// stays at one batch however large the index grows, and dumps past the old
// maxExportRows cap stream fully instead of truncating. A var so tests can
// shrink the page and walk the loop's seams deterministically.
//
//nolint:gochecknoglobals // test-shrinkable page size (keyset seam tests)
var exportBatchSize = 5_000

// StreamExportCalls walks every view matching the same filters as the list
// page, newest first, calling fn per view (D5). Pagination is a started_at
// keyset; views sharing the cursor timestamp are carried across pages via a
// seen-key set, so a page boundary inside a timestamp group cannot drop rows.
// A group so large it stalls the cursor (more than one batch of calls inside
// a single nanosecond) fails loudly instead of truncating silently.
//
// It is deliberately the PER-SOURCE walk: an export is a forensic dump, so
// each source row appears under its own kind and native id exactly as stored.
// Operator-facing aggregates use streamRepresentativeCalls instead.
func StreamExportCalls(
	ctx context.Context,
	sys *system.System,
	query ListCallsQuery,
	fn func(calls.CallView) error,
) error {
	return streamViews(ctx, sys, query, fn)
}

// streamRepresentativeCalls is streamViews with the duplicate-source collapse
// applied: a call whose CDR and agent transcript are both indexed is visited
// once, as its CDR row (see representative.go). Every operator-facing
// aggregate counts this walk, never the per-source one.
func streamRepresentativeCalls(
	ctx context.Context,
	sys *system.System,
	query ListCallsQuery,
	fn func(calls.CallView) error,
) error {
	reader := metaengine.NewReader[calls.CallView](sys.MetaEngine(), calls.Collection)

	return streamViews(ctx, sys, query, func(view calls.CallView) error {
		superseded, err := supersededBySibling(ctx, reader, view)
		if err != nil {
			return err
		}

		if superseded {
			return nil
		}

		return fn(view)
	})
}

// streamViews is the keyset walk every streaming reader shares.
func streamViews(
	ctx context.Context,
	sys *system.System,
	query ListCallsQuery,
	fn func(calls.CallView) error,
) error {
	reader := metaengine.NewReader[calls.CallView](sys.MetaEngine(), calls.Collection)

	var cursor time.Time

	seen := make(map[string]struct{})

	for {
		views, err := reader.Scan(ctx, streamPageOptions(cursor, query)...)
		if err != nil {
			return err
		}

		emitted := 0

		for _, view := range views {
			if !cursor.IsZero() && view.StartedAt.Equal(cursor) {
				if _, done := seen[dedupeKey(view)]; done {
					continue
				}
			}

			if err := fn(view); err != nil {
				return err
			}

			emitted++
		}

		if len(views) < exportBatchSize {
			return nil
		}

		if emitted == 0 {
			return errorfamily.NewRejection(
				"lens.web.export_cursor_stalled",
				"more than one page of calls share started_at "+rfc3339Nano(cursor)+
					"; narrow the export with from/to filters",
			)
		}

		cursor = advanceCursor(cursor, views, seen)
	}
}

// streamPageOptions binds one keyset page: inclusively at the live cursor
// (rows sharing the cursor's timestamp reappear and get skipped by key);
// on the first page the caller's own Before (a plain exclusive bound)
// applies instead.
func streamPageOptions(cursor time.Time, query ListCallsQuery) []metaengine.ScanOption {
	opts := append([]metaengine.ScanOption{
		metaengine.WithSort("started_at", true),
		metaengine.WithLimit(exportBatchSize),
	}, scanFilters(query)...)

	switch {
	case !cursor.IsZero():
		opts = append(opts, metaengine.WithFilter("started_at", metaengine.FilterLe, rfc3339Nano(cursor)))
	case !query.Before.IsZero():
		opts = append(opts, metaengine.WithFilter("started_at", metaengine.FilterLt, rfc3339Nano(query.Before)))
	}

	return opts
}

// dedupeKey identifies a stream across the page boundary tie set.
func dedupeKey(view calls.CallView) string {
	return string(view.Kind) + "\x00" + string(view.NativeID)
}

// advanceCursor moves the keyset to the page's last (oldest) row and
// maintains the tie set: on a strict advance the seen set restarts; when a
// single timestamp group spans pages the boundary repeats and the set must
// survive so those ties stay deduped. The boundary row itself was emitted
// on THIS page and sits exactly on the cursor, so it (and any ties) is
// recorded for the next page to skip.
func advanceCursor(cursor time.Time, views []calls.CallView, seen map[string]struct{}) time.Time {
	if next := views[len(views)-1].StartedAt; cursor.IsZero() || next.Before(cursor) {
		cursor = next

		clear(seen)
	}

	for _, view := range views {
		if view.StartedAt.Equal(cursor) {
			seen[dedupeKey(view)] = struct{}{}
		}
	}

	return cursor
}

// ExportCalls collects the full matching set into one slice. Tests and
// callers that need random access use it; the HTTP export streams instead.
func ExportCalls(ctx context.Context, sys *system.System, query ListCallsQuery) ([]calls.CallView, error) {
	var views []calls.CallView

	err := StreamExportCalls(ctx, sys, query, func(view calls.CallView) error {
		views = append(views, view)

		return nil
	})
	if err != nil {
		return nil, err
	}

	return views, nil
}

// listOptionsAt builds one list-walk batch: newest first, with the same
// typed filters, bounded below by the live keyset cursor (inclusive, so the
// tie set can skip what this walk already emitted) or the caller's own
// exclusive Before on the first batch.
func listOptionsAt(query ListCallsQuery, cursor time.Time, limit int) []metaengine.ScanOption {
	opts := append([]metaengine.ScanOption{
		metaengine.WithSort("started_at", true),
		metaengine.WithLimit(limit),
	}, scanFilters(query)...)

	// Time bounds bind as RFC3339Nano strings — the exact wire format the
	// view's JSON is stored in. Binding time.Time directly miscompares on
	// sqlite (the driver renders it "2006-01-02 15:04:05 -0700 MST", and the
	// 'T' vs ' ' format difference silently corrupts same-day comparisons).
	// Verified against eq/le/ge/lt; upstream sqliteengine issue filed.
	switch {
	case !cursor.IsZero():
		opts = append(opts, metaengine.WithFilter("started_at", metaengine.FilterLe, rfc3339Nano(cursor)))
	case !query.Before.IsZero():
		opts = append(opts, metaengine.WithFilter("started_at", metaengine.FilterLt, rfc3339Nano(query.Before)))
	}

	if !query.After.IsZero() {
		opts = append(opts, metaengine.WithFilter("started_at", metaengine.FilterGt, rfc3339Nano(query.After)))
	}

	return opts
}

// scanFilters is the typed-filter half of the scan options, shared by the
// list page and the streaming export (whose cursor binds separately).
func scanFilters(query ListCallsQuery) []metaengine.ScanOption {
	var opts []metaengine.ScanOption

	if query.Direction != "" {
		opts = append(opts, metaengine.WithFilter("direction", metaengine.FilterEq, string(query.Direction)))
	}

	if query.AI != nil {
		opts = append(opts, metaengine.WithFilter("is_ai", metaengine.FilterEq, *query.AI))
	}

	if query.Number != "" {
		opts = append(opts, metaengine.WithOr(
			metaengine.FilterSpec{Column: "from_number", Op: metaengine.FilterEq, Value: query.Number},
			metaengine.FilterSpec{Column: "to_number", Op: metaengine.FilterEq, Value: query.Number},
		))
	}

	if !query.From.IsZero() {
		opts = append(opts, metaengine.WithFilter("started_at", metaengine.FilterGe, rfc3339Nano(query.From)))
	}

	if !query.To.IsZero() {
		opts = append(opts, metaengine.WithFilter("started_at", metaengine.FilterLe, rfc3339Nano(query.To)))
	}

	return opts
}

// rfc3339Nano renders a query bound in the read model's stored timestamp
// format (see ListCalls).
func rfc3339Nano(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}

// CallDetail looks one call up by its stream identity parts.
func CallDetail(ctx context.Context, sys *system.System, kind, nativeID string) (calls.CallView, bool, error) {
	obs := calls.Observation{NativeID: calls.NativeID(nativeID), Kind: calls.SourceKind(kind)}

	sid, err := obs.StreamID()
	if err != nil {
		//nolint:nilerr // a malformed kind/id is a plain miss (404), not a server error
		return calls.CallView{}, false, nil
	}

	reader := metaengine.NewReader[calls.CallView](sys.MetaEngine(), calls.Collection)

	view, found, err := calls.LookupView(ctx, reader, sid)
	if err != nil || !found {
		return calls.CallView{}, false, err
	}

	return view, true, nil
}

// DetailPage is the per-call page model: the requested source row plus what
// the OTHER sources recorded for the same call. The read model deliberately
// keeps one row per source (no source's truth is rewritten or merged at
// ingest); the operator-facing page joins them because a call is one call,
// not one row.
type DetailPage struct {
	calls.CallView

	// JoinedFrom names the sibling source rows the page had to consult to
	// complete this call. Empty when the requested row already carried the
	// transcript and the parties.
	JoinedFrom []calls.SourceKind

	// LocalRecording is the basename of the on-disk WAV resolved for the
	// call. Empty when none exists; the caller resolves it (it owns the
	// recordings directory).
	LocalRecording string
}

// siblingKinds lists the other source rows that describe the same real-world
// call under the SAME native id. FreeSWITCH's call uuid is shared by its CDR
// and the voice agent's per-call transcript; the receiver webhook log keys
// Telnyx's call_control_id and Telnyx conversations key their own
// conversation id, so neither participates in the uuid join.
func siblingKinds(kind calls.SourceKind) []calls.SourceKind {
	switch kind {
	case calls.SourceFreeSwitchCDR:
		return []calls.SourceKind{calls.SourceAgentTranscript}
	case calls.SourceAgentTranscript:
		return []calls.SourceKind{calls.SourceFreeSwitchCDR}
	case calls.SourceTelnyxConversation, calls.SourceWebhookLog:
		// Both key their own id spaces; neither joins on the uuid.
		return nil
	default:
		// A future kind opts in explicitly; no silent sibling guesswork.
		return nil
	}
}

// CallDetailPage reads one call for the detail page and fills the facts a
// single source cannot know from its sibling rows. An unreadable sibling row
// is an error, not a silent partial page: a masked failure would render the
// exact "nothing available" lie this page exists to prevent.
func CallDetailPage(
	ctx context.Context,
	sys *system.System,
	kind, nativeID string,
) (DetailPage, bool, error) {
	view, found, err := CallDetail(ctx, sys, kind, nativeID)
	if err != nil || !found {
		return DetailPage{}, found, err
	}

	page := DetailPage{CallView: view}

	reader := metaengine.NewReader[calls.CallView](sys.MetaEngine(), calls.Collection)

	for _, siblingKind := range siblingKinds(view.Kind) {
		obs := calls.Observation{NativeID: view.NativeID, Kind: siblingKind}

		sid, streamErr := obs.StreamID()
		if streamErr != nil {
			continue
		}

		sibling, siblingFound, siblingErr := calls.LookupView(ctx, reader, sid)
		if siblingErr != nil {
			return DetailPage{}, false, siblingErr
		}

		if !siblingFound {
			continue
		}

		merged, changed := mergeCall(page.CallView, sibling)
		if changed {
			page.CallView = merged
			page.JoinedFrom = append(page.JoinedFrom, siblingKind)
		}
	}

	return page, true, nil
}

// mergeCall fills the primary view's gaps from a sibling source row. Only
// facts the primary source cannot know move across: identity (Kind, NativeID,
// ContentHash), enrichment, and the recording reference stay the primary
// row's. A primary that already carries a fact keeps its own version — the
// requested row is the authority for what it observed.
func mergeCall(primary, sibling calls.CallView) (calls.CallView, bool) {
	transcriptJoined := fillTranscript(&primary, sibling)
	fromJoined := fillString(&primary.FromNumber, sibling.FromNumber)
	toJoined := fillString(&primary.ToNumber, sibling.ToNumber)
	causeJoined := fillString(&primary.HangupCause, sibling.HangupCause)
	durationJoined := fillDuration(&primary.DurationSecs, sibling.DurationSecs)
	startJoined := fillStartedAt(&primary.StartedAt, sibling.StartedAt)
	aiJoined := fillFlag(&primary.IsAI, sibling.IsAI)

	return primary, transcriptJoined || fromJoined || toJoined || causeJoined ||
		durationJoined || startJoined || aiJoined
}

func fillTranscript(primary *calls.CallView, sibling calls.CallView) bool {
	if len(primary.Transcript) > 0 || len(sibling.Transcript) == 0 {
		return false
	}

	primary.Transcript = sibling.Transcript

	if primary.TranscriptExcerpt == "" {
		primary.TranscriptExcerpt = sibling.TranscriptExcerpt
	}

	return true
}

func fillString(destination *string, source string) bool {
	if *destination != "" || source == "" {
		return false
	}

	*destination = source

	return true
}

func fillDuration(destination *int, source int) bool {
	if *destination != 0 || source == 0 {
		return false
	}

	*destination = source

	return true
}

func fillStartedAt(destination *time.Time, source time.Time) bool {
	if !destination.IsZero() || source.IsZero() {
		return false
	}

	*destination = source

	return true
}

func fillFlag(destination *bool, source bool) bool {
	if *destination || !source {
		return false
	}

	*destination = source

	return true
}
