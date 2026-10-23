package cli

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/pbx-lens/internal/calls"
)

func classifyFixtures(t *testing.T) (context.Context, *metaengine.TypedReader[calls.CallView], func()) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	sys, cleanup, err := OpenSystem(ctx, "")
	if err != nil {
		cancel()

		t.Fatalf("OpenSystem: %v", err)
	}

	reader := metaengine.NewReader[calls.CallView](sys.MetaEngine(), calls.Collection)

	// Ingest one row so the classifier has a stored view to compare against.
	stored := observation("conv-refresh-stored", 26)
	if stats := Ingest(ctx, sys, []calls.Observation{stored}); stats.Recorded != 1 {
		t.Fatalf("seed ingest = %+v, want 1 recorded", stats)
	}

	if err := DrainProjections(ctx, sys); err != nil {
		t.Fatalf("drain: %v", err)
	}

	return ctx, reader, func() {
		cleanup()
		cancel()
	}
}

func TestClassifyRefreshTellsNewStaleAndUnchanged(t *testing.T) {
	ctx, reader, cleanup := classifyFixtures(t)

	defer cleanup()

	stored := observation("conv-refresh-stored", 26)
	grown := observation("conv-refresh-stored", 40)
	fresh := observation("conv-refresh-new", 12)

	report, err := ClassifyRefresh(ctx, reader, []calls.Observation{stored, grown, fresh})
	if err != nil {
		t.Fatalf("ClassifyRefresh: %v", err)
	}

	if got, want := report.Total, 3; got != want {
		t.Fatalf("Total = %d, want %d", got, want)
	}

	if got, want := report.Count(RefreshUnchanged), 1; got != want {
		t.Errorf("unchanged = %d, want %d (the identical snapshot)", got, want)
	}

	if got, want := report.Count(RefreshStale), 1; got != want {
		t.Errorf("stale = %d, want %d (the grown duration)", got, want)
	}

	if got, want := report.Count(RefreshNew), 1; got != want {
		t.Errorf("new = %d, want %d (the unseen stream)", got, want)
	}

	if got, want := report.Rewrites(), 2; got != want {
		t.Errorf("Rewrites = %d, want %d — a re-ingest writes new + stale only", got, want)
	}
}

func TestClassifyRefreshFindsUnhashedDrift(t *testing.T) {
	ctx, reader, cleanup := classifyFixtures(t)

	defer cleanup()

	// Same hashed content, one additive marker the stored row does not have:
	// dispatching this observation would replay as a no-op, which is exactly
	// the silent-drift class the schema-version bump exists to close.
	stored := observation("conv-refresh-stored", 26)
	stored.SourceEvents = []calls.SourceEvent{{
		Role: "system",
		Text: "[long silence]",
		At:   time.Date(2026, 10, 9, 19, 51, 40, 0, time.UTC),
	}}

	report, err := ClassifyRefresh(ctx, reader, []calls.Observation{stored})
	if err != nil {
		t.Fatalf("ClassifyRefresh: %v", err)
	}

	if got, want := report.Count(RefreshAdditiveOnly), 1; got != want {
		t.Fatalf("additive-only = %d, want %d (same hash, extra source event)", got, want)
	}

	reason := report.Findings[0].Reason
	if !strings.Contains(reason, "re-ingest is a no-op") {
		t.Errorf("reason %q must name why the drift is invisible to a re-ingest", reason)
	}

	if got, want := report.Rewrites(), 0; got != want {
		t.Errorf("Rewrites = %d, want %d — an equal hash writes nothing even with additive drift", got, want)
	}
}

func TestClassifyRefreshReportsUnbuildableStreams(t *testing.T) {
	ctx, reader, cleanup := classifyFixtures(t)

	defer cleanup()

	broken := observation("conv-refresh-broken", 3)
	broken.Kind = calls.SourceKind("carrier-pigeon")

	report, err := ClassifyRefresh(ctx, reader, []calls.Observation{broken})
	if err != nil {
		t.Fatalf("ClassifyRefresh: %v", err)
	}

	if got, want := report.Count(RefreshUnbuildable), 1; got != want {
		t.Fatalf("unbuildable = %d, want %d", got, want)
	}

	if got, want := report.Rewrites(), 0; got != want {
		t.Errorf("Rewrites = %d, want %d — a stream without identity is never dispatched", got, want)
	}
}

func TestRefreshReportStringNamesEveryOutcome(t *testing.T) {
	report := RefreshReport{Total: 4, Findings: []RefreshFinding{
		{Stream: "a", Outcome: RefreshNew},
		{Stream: "b", Outcome: RefreshStale},
		{Stream: "c", Outcome: RefreshAdditiveOnly},
		{Stream: "d", Outcome: RefreshUnchanged},
	}}

	got := report.String()

	for _, want := range []string{"4 streams", "1 new", "1 stale", "1 additive-only", "1 unchanged", "would write 2"} {
		if !strings.Contains(got, want) {
			t.Errorf("report %q must mention %q", got, want)
		}
	}
}
