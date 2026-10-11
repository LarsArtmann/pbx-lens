package cli

import (
	"context"
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

// awaitView polls the projection until the stream id converges, the way the
// smoke command does; classification in Ingest reads the same reader.
func awaitView(t *testing.T, sys *system.System, streamID string) calls.CallView {
	t.Helper()

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

func TestIngestRecordsCorrectsAndRejects(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	sys, cleanup, err := OpenSystem(ctx, "")
	if err != nil {
		t.Fatalf("OpenSystem: %v", err)
	}

	defer cleanup()

	obs := observation("conv-a", 26)

	first := Ingest(ctx, sys, []calls.Observation{obs})
	if first.Total != 1 || first.Recorded != 1 || first.Corrected != 0 || len(first.Rejected) != 0 {
		t.Fatalf("first pass = %+v, want 1 recorded", first)
	}

	awaitView(t, sys, "telnyx-conversation:conv-a")

	replay := Ingest(ctx, sys, []calls.Observation{obs})
	if replay.Recorded != 1 || replay.Corrected != 0 {
		t.Fatalf("identical replay = %+v, want idempotent record", replay)
	}

	grown := observation("conv-a", 40)

	corrected := Ingest(ctx, sys, []calls.Observation{grown})
	if corrected.Recorded != 0 || corrected.Corrected != 1 {
		t.Fatalf("changed snapshot = %+v, want 1 corrected", corrected)
	}

	view := awaitView(t, sys, "telnyx-conversation:conv-a")
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
