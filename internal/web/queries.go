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
	"github.com/larsartmann/pbx-lens/internal/calls"
)

// pageSize caps one rendered page (bounding-tables doctrine: no unbounded
// table ever ships; older calls walk via the keyset cursor).
const pageSize = 50

// ListCallsQuery is the fully-parsed list-page request. Zero fields mean
// unfiltered; Before is the exclusive keyset cursor (RFC3339).
type ListCallsQuery struct {
	Direction calls.Direction
	AI        *bool
	From      time.Time
	To        time.Time
	Before    time.Time
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

// CallListPage is one rendered page plus the cursor for the next-older walk.
type CallListPage struct {
	Rows      []CallRow
	NextOlder string // empty when this page was the last
}

// ListCalls runs the query against the read model, newest first.
func ListCalls(ctx context.Context, sys *system.System, query ListCallsQuery) (CallListPage, error) {
	reader := metaengine.NewReader[calls.CallView](sys.MetaEngine(), calls.Collection)

	// fetch one extra row to detect whether an older page exists
	views, err := reader.Scan(ctx, listOptions(query, pageSize+1)...)
	if err != nil {
		return CallListPage{}, err
	}

	page := CallListPage{Rows: make([]CallRow, 0, min(len(views), pageSize))}

	for _, view := range views {
		if len(page.Rows) == pageSize {
			page.NextOlder = views[pageSize-1].StartedAt.UTC().Format(time.RFC3339)

			break
		}

		page.Rows = append(page.Rows, CallRow{
			Kind:      view.Kind,
			NativeID:  string(view.NativeID),
			When:      view.StartedAt,
			From:      view.FromNumber,
			To:        view.ToNumber,
			Direction: view.Direction,
			Duration:  time.Duration(view.DurationSecs) * time.Second,
			IsAI:      view.IsAI,
			Excerpt:   view.TranscriptExcerpt,
		})
	}

	return page, nil
}

// maxExportRows bounds one export dump. Personal-PBX volumes are far below
// this; the cap only stops a runaway query from exhausting memory.
const maxExportRows = 100_000

// ExportCalls returns the raw views matching the same filters as the list
// page, newest first, for the JSONL/CSV dump (D5).
func ExportCalls(ctx context.Context, sys *system.System, query ListCallsQuery) ([]calls.CallView, error) {
	reader := metaengine.NewReader[calls.CallView](sys.MetaEngine(), calls.Collection)

	return reader.Scan(ctx, listOptions(query, maxExportRows)...)
}

// listOptions builds the shared scan options for the list page and the
// export dump: newest first, with the same typed filters.
func listOptions(query ListCallsQuery, limit int) []metaengine.ScanOption {
	opts := []metaengine.ScanOption{
		metaengine.WithSort("started_at", true),
		metaengine.WithLimit(limit),
	}

	if query.Direction != "" {
		opts = append(opts, metaengine.WithFilter("direction", metaengine.FilterEq, string(query.Direction)))
	}

	if query.AI != nil {
		opts = append(opts, metaengine.WithFilter("is_ai", metaengine.FilterEq, *query.AI))
	}

	// Time bounds bind as RFC3339Nano strings — the exact wire format the
	// view's JSON is stored in. Binding time.Time directly miscompares on
	// sqlite (the driver renders it "2006-01-02 15:04:05 -0700 MST", and the
	// 'T' vs ' ' format difference silently corrupts same-day comparisons).
	// Verified against eq/le/ge/lt; upstream sqliteengine issue filed.
	if !query.From.IsZero() {
		opts = append(opts, metaengine.WithFilter("started_at", metaengine.FilterGe, rfc3339Nano(query.From)))
	}

	if !query.To.IsZero() {
		opts = append(opts, metaengine.WithFilter("started_at", metaengine.FilterLe, rfc3339Nano(query.To)))
	}

	if !query.Before.IsZero() {
		opts = append(opts, metaengine.WithFilter("started_at", metaengine.FilterLt, rfc3339Nano(query.Before)))
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

	view, found, err := reader.Get(ctx, sid.String())
	if err != nil || !found {
		return calls.CallView{}, false, err
	}

	return view, true, nil
}
