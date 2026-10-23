package cli

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
	"github.com/larsartmann/pbx-lens/internal/calls"
)

func observation(nativeID string, durationSecs int) calls.Observation {
	return calls.Observation{
		NativeID:     calls.NativeID(nativeID),
		Kind:         calls.SourceTelnyxConversation,
		Direction:    calls.DirectionOutbound,
		StartedAt:    time.Date(2026, 10, 9, 19, 51, 25, 0, time.UTC),
		DurationSecs: durationSecs,
		FromNumber:   "+17287289311",
		ToNumber:     "+15555550123",
	}
}

// awaitView drains the projection host, then polls until the stream id
// converges — the one-shot command flow (dispatch, drain, read).
func awaitView(t *testing.T, sys *system.System, streamID string) calls.CallView {
	t.Helper()

	if err := DrainProjections(context.Background(), sys); err != nil {
		t.Fatalf("drain: %v", err)
	}

	reader := metaengine.NewReader[calls.CallView](sys.MetaEngine(), calls.Collection)
	deadline := time.Now().Add(5 * time.Second)

	for {
		view, found, err := reader.Get(context.Background(), streamID)
		if err != nil {
			t.Fatalf("reader.Get(%s): %v", streamID, err)
		}

		if found {
			return view
		}

		if time.Now().After(deadline) {
			t.Fatalf("projection for %s did not converge in 5s", streamID)
		}

		time.Sleep(20 * time.Millisecond)
	}
}

func viewKey(t *testing.T, obs calls.Observation) string {
	t.Helper()

	sid, err := obs.StreamID()
	if err != nil {
		t.Fatalf("StreamID: %v", err)
	}

	// calls.RecordKey owns the projection's record-key format: the bare
	// stream value (projectionadapter v4.6.0+).
	return calls.RecordKey(sid)
}

func TestIngestRecordsCorrectsAndRejects(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	sys, cleanup, err := OpenSystem(ctx, "")
	if err != nil {
		t.Fatalf("OpenSystem: %v", err)
	}

	defer cleanup()

	obs := observation("conv-a", 26)
	key := viewKey(t, obs)

	first := Ingest(ctx, sys, []calls.Observation{obs})
	if first.Total != 1 || first.Recorded != 1 || first.Corrected != 0 || len(first.Rejected) != 0 {
		t.Fatalf("first pass = %+v, want 1 recorded", first)
	}

	awaitView(t, sys, key)

	replay := Ingest(ctx, sys, []calls.Observation{obs})
	if replay.Recorded != 1 || replay.Corrected != 0 {
		t.Fatalf("identical replay = %+v, want idempotent record", replay)
	}

	if err := DrainProjections(ctx, sys); err != nil {
		t.Fatalf("drain after replay: %v", err)
	}

	grown := observation("conv-a", 40)

	corrected := Ingest(ctx, sys, []calls.Observation{grown})
	if corrected.Recorded != 0 || corrected.Corrected != 1 {
		t.Fatalf("changed snapshot = %+v, want 1 corrected", corrected)
	}

	view := awaitView(t, sys, key)
	if view.DurationSecs != 40 {
		t.Fatalf("corrected view duration = %d, want 40", view.DurationSecs)
	}

	invalid := observation("", 10)
	invalid.Kind = calls.SourceKind("bogus")

	rejected := Ingest(ctx, sys, []calls.Observation{invalid})
	if rejected.Recorded != 0 || len(rejected.Rejected) != 1 {
		t.Fatalf("invalid observation = %+v, want 1 rejection", rejected)
	}
}

func TestGateIngestRefusesOverAgeAndReportsAgainstBatch(t *testing.T) {
	ctx := context.Background()

	sys, cleanup, err := OpenSystem(ctx, "")
	if err != nil {
		t.Fatalf("OpenSystem: %v", err)
	}

	t.Cleanup(cleanup)

	fresh := observation("gate-fresh", 30)
	fresh.StartedAt = time.Now().UTC().Add(-1 * time.Hour)
	overAge := observation("gate-overage", 30)
	overAge.StartedAt = time.Now().UTC().Add(-25 * time.Hour)

	stats := GateIngest(ctx, sys, 24*time.Hour, []calls.Observation{fresh, overAge})

	if stats.Total != 2 {
		t.Errorf("Total = %d, want 2 (everything the caller offered)", stats.Total)
	}

	if stats.Recorded != 1 {
		t.Errorf("Recorded = %d, want 1", stats.Recorded)
	}

	if len(stats.Rejected) != 1 || !strings.Contains(stats.Rejected[0], "retention:age-cutoff") {
		t.Fatalf("rejected = %v, want one retention:age-cutoff entry", stats.Rejected)
	}

	// A zero gate keeps everything (retention is opt-in, never silent).
	stats = GateIngest(ctx, sys, 0, []calls.Observation{overAge})
	if stats.Recorded != 1 || len(stats.Rejected) != 0 {
		t.Errorf("zero MaxAge must admit everything, got %+v", stats)
	}
}
