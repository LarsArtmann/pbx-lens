package web

import (
	"context"
	"sort"
	"time"

	"github.com/larsartmann/go-cqrs-lite/system/v4"
	"github.com/larsartmann/pbx-lens/internal/calls"
)

// coverageStaleAfter is the ingest-coverage SLO default (plan P17): a
// source whose newest indexed row is older than this is flagged as a gap.
// One uniform ceiling keeps the flag explainable; per-source intervals are
// deployment knowledge the poller already expresses through its pass
// cadence — this flag answers the coarser "is the index drying up".
const coverageStaleAfter = 48 * time.Hour

// CoverageRow is one source kind's indexed coverage: how many rows it
// contributed and the span they cover. The counts come from the RAW walk —
// every indexed row, including siblings the list collapses — because
// coverage is about what each SOURCE delivered, not about distinct calls.
type CoverageRow struct {
	Kind   calls.SourceKind
	Views  int
	Oldest time.Time
	Newest time.Time
}

// NewestAge is the age of the source's newest row against now.
func (r CoverageRow) NewestAge(now time.Time) time.Duration { return now.Sub(r.Newest) }

// Stale reports the gap flag: the source HAS rows but the newest one is
// past the SLO ceiling. A kind with zero rows is absent from the coverage
// table entirely — "never seen" is a different fact from "went quiet",
// and the page says so.
func (r CoverageRow) Stale(now time.Time) bool { return r.NewestAge(now) > coverageStaleAfter }

// IngestCoverageQuery folds the raw per-source walk into per-kind coverage
// rows, ordered by volume then kind — the same total-order discipline the
// segment tables use, so the card never reshuffles between loads.
func IngestCoverageQuery(ctx context.Context, sys *system.System, now time.Time) ([]CoverageRow, error) {
	byKind := map[calls.SourceKind]*CoverageRow{}

	err := streamViews(ctx, sys, ListCallsQuery{}, func(view calls.CallView) error {
		row, ok := byKind[view.Kind]
		if !ok {
			row = &CoverageRow{Kind: view.Kind}
			byKind[view.Kind] = row
		}

		row.Views++
		if row.Oldest.IsZero() || view.StartedAt.Before(row.Oldest) {
			row.Oldest = view.StartedAt
		}

		if view.StartedAt.After(row.Newest) {
			row.Newest = view.StartedAt
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	rows := make([]CoverageRow, 0, len(byKind))
	for _, row := range byKind {
		rows = append(rows, *row)
	}

	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Views != rows[j].Views {
			return rows[i].Views > rows[j].Views
		}

		return rows[i].Kind < rows[j].Kind
	})

	return rows, nil
}
