package app_test

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
	"github.com/larsartmann/go-cqrs-lite/systemscenario/v4"
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

// errViewNotFound is the sentinel a viewReader returns while the projected
// row does not exist (yet or anymore) — ThenQueryEventuallyFails matches it
// with errors.Is.
var errViewNotFound = errors.New("call view row not found")

// viewReader returns a ThenQuery-shaped fn over the system's call_views
// collection, replacing the hand-rolled 20ms poll loop the harness used to
// carry: the scenario's Then* assertions own the awaiting.
func viewReader(sys *system.System, key string) func() (any, error) {
	reader := metaengine.NewReader[calls.CallView](sys.MetaEngine(), calls.Collection)

	return func() (any, error) {
		view, found, err := reader.Get(context.Background(), key)
		if err != nil {
			return nil, err
		}

		if !found {
			return nil, fmt.Errorf("%w: %q", errViewNotFound, key)
		}

		return view, nil
	}
}

// TestSqliteEndToEndAndReplay boots the production wiring through the
// systemscenario harness (ADR-0123 holds: the scenario calls system.New with
// app.DomainConfig + app.DeploymentConfig — the same shapes the serve and
// CLI commands boot), records one observation, checks the projected view,
// then reboots on the same DSN and proves the replay folds an identical
// view. Re-dispatch idempotency is a Given-seeded scenario of its own
// (TestReIngestSameSnapshotIsANoOp), because a scenario's Then* window is
// cumulative since its first act.
func TestSqliteEndToEndAndReplay(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	dsn := filepath.Join(t.TempDir(), "lens.db")

	obs := sampleObservation()
	cmd, err := calls.NewRecordCommand(obs)
	if err != nil {
		t.Fatal(err)
	}

	key := calls.RecordKey(cmd.StreamID())

	sc := systemscenario.System(t, ctx, app.DomainConfig(), app.DeploymentConfig(dsn))

	var view calls.CallView

	sc.When(cmd).
		Then(calls.EventObserved).
		ThenQueryFunc(viewReader(sc.System(), key), func(got any) error {
			v, ok := got.(calls.CallView)
			if !ok {
				return fmt.Errorf("view query returned %T", got)
			}

			if v.Kind != calls.SourceTelnyxConversation || v.Direction != calls.DirectionOutbound {
				return fmt.Errorf("view kind/direction wrong: %+v", v)
			}

			if !v.IsAI || v.DurationSecs != 26 || v.NativeID != "conv-e2e-1" {
				return fmt.Errorf("view fields wrong: %+v", v)
			}

			view = v

			return nil
		})

	if view.ContentHash != obs.ContentHash() {
		t.Fatalf("view content hash mismatch: %q", view.ContentHash)
	}

	replayed := systemscenario.System(t, ctx, app.DomainConfig(), app.DeploymentConfig(dsn))
	replayed.When(cmd).
		ThenNoEvents().
		ThenQuery(viewReader(replayed.System(), key), view)
}
