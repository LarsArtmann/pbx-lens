package app_test

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
	"github.com/larsartmann/pbx-lens/internal/app"
	"github.com/larsartmann/pbx-lens/internal/calls"
)

func sampleObservation() calls.Observation {
	return calls.Observation{
		NativeID:          "conv-e2e-1",
		Kind:              calls.SourceTelnyxConversation,
		Direction:         calls.DirectionOutbound,
		StartedAt:         time.Date(2026, 10, 9, 19, 51, 25, 0, time.UTC),
		DurationSecs:      26,
		FromNumber:        "+17287289311",
		ToNumber:          "+14084757593",
		HangupCause:       "normal_clearing",
		IsAI:              true,
		TranscriptExcerpt: "This is the Bear Union regarding your two-egg offer.",
		RecordingRef:      "c8daa4ef-020c-49c5-8a35-888b41e33900",
	}
}

func startProjections(t *testing.T, ctx context.Context, sys *system.System) {
	t.Helper()

	hostCtx, stop := context.WithCancel(ctx)
	t.Cleanup(stop)

	go func() { _ = sys.ProjectionHost().Start(hostCtx) }()
}

func awaitView(t *testing.T, ctx context.Context, sys *system.System, key string) calls.CallView {
	t.Helper()

	reader := metaengine.NewReader[calls.CallView](sys.MetaEngine(), calls.Collection)
	deadline := time.Now().Add(5 * time.Second)

	for {
		view, found, err := reader.Get(ctx, key)
		if err != nil {
			t.Fatal(err)
		}

		if found {
			return view
		}

		if time.Now().After(deadline) {
			t.Fatal("projection did not converge within 5s")
		}

		time.Sleep(20 * time.Millisecond)
	}
}

func TestSqliteEndToEndAndReplay(t *testing.T) {
	dsn := filepath.Join(t.TempDir(), "lens.db")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	sys, err := app.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}

	startProjections(t, ctx, sys)

	cmd, err := calls.NewRecordCommand(sampleObservation())
	if err != nil {
		t.Fatal(err)
	}

	if err := sys.CommandDispatcher().Dispatch(ctx, cmd); err != nil {
		t.Fatal(err)
	}

	if err := sys.CommandDispatcher().Dispatch(ctx, cmd); err != nil {
		t.Fatalf("idempotent re-dispatch must succeed, got %v", err)
	}

	view := awaitView(t, ctx, sys, cmd.StreamID().String())
	if view.Kind != calls.SourceTelnyxConversation || view.Direction != calls.DirectionOutbound {
		t.Fatalf("view kind/direction wrong: %+v", view)
	}

	if !view.IsAI || view.DurationSecs != 26 || view.NativeID != "conv-e2e-1" {
		t.Fatalf("view fields wrong: %+v", view)
	}

	if view.ContentHash != sampleObservation().ContentHash() {
		t.Fatalf("view content hash mismatch: %q", view.ContentHash)
	}

	if err := sys.Close(); err != nil {
		t.Fatal(err)
	}

	sys2, err := app.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer sys2.Close()

	startProjections(t, ctx, sys2)

	replayed := awaitView(t, ctx, sys2, cmd.StreamID().String())
	if !reflect.DeepEqual(replayed, view) {
		t.Fatalf("replayed view differs:\n got %+v\nwant %+v", replayed, view)
	}
}
