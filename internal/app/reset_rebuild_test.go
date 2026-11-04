package app_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
	"github.com/larsartmann/go-cqrs-lite/systemscenario/v4"
	"github.com/larsartmann/pbx-lens/internal/app"
	"github.com/larsartmann/pbx-lens/internal/calls"
	"github.com/larsartmann/pbx-lens/internal/cli"
)

// TestResetRebuildsReadModelFromJournal is the memory-engine dry run of the
// legacy-key retirement runbook (docs/runbooks/reset-rebuild.md, TODO_LIST
// row 113): fold two streams, prove the journal SURVIVES the reset, reset
// all engines (never partial), replay through the production drain, and
// assert the read model came back with every stream rebuilt. This is the
// proof the owner's one-shot prod rebuild stands on.
func TestResetRebuildsReadModelFromJournal(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	sc := systemscenario.System(
		t,
		ctx,
		app.DomainConfig(),
		systemscenario.Memory(),
		systemscenario.WithAwaitTimeout(scenarioAwaitTimeout),
	)

	first := sampleObservation()
	second := sampleObservation()
	second.NativeID = "conv-second"
	second.StartedAt = first.StartedAt.Add(time.Minute)

	// Seed both streams (the multi-seed Given pattern), then pin the
	// idempotent re-offer on the same act: the replay decides to ZERO
	// events (same ContentHash) and the terminal query proves both views
	// folded — the scenario's non-vacuous guard rides this phase.
	sc.Given(
		sc.Event(calls.EventObserved, streamRefOf(mustRecord(t, first)), calls.CallObserved{Observation: first}),
		sc.Event(calls.EventObserved, streamRefOf(mustRecord(t, second)), calls.CallObserved{Observation: second}),
	)

	sc.When(mustRecord(t, second)).
		ThenNoEvents().
		ThenQueryFunc(viewCountReader(sc.System()), func(got any) error {
			if count, ok := got.(int); !ok || count != 2 {
				return fmt.Errorf("after seed: views = %v, want 2", got)
			}

			return nil
		})

	sys := sc.System()

	before, err := readAllViewsForTest(ctx, sys)
	if err != nil {
		t.Fatal(err)
	}

	if len(before) != 2 {
		t.Fatalf("pre-reset views = %d, want 2", len(before))
	}

	result, err := sys.MetaEngine().Reset(ctx)
	if err != nil {
		t.Fatalf("reset: %v", err)
	}

	if result.Partial() {
		t.Fatalf("reset was partial: cleared=%v unclearable=%v", result.ClearedEngines, result.UnclearableEngines)
	}

	after, err := readAllViewsForTest(ctx, sys)
	if err != nil {
		t.Fatalf("post-reset read (engines should be empty): %v", err)
	}

	if len(after) != 0 {
		t.Fatalf("post-reset views = %d, want 0 (engines fully cleared)", len(after))
	}

	if err := cli.DrainProjections(ctx, sys); err != nil {
		t.Fatalf("replay drain: %v", err)
	}

	rebuilt, err := readAllViewsForTest(ctx, sys)
	if err != nil {
		t.Fatal(err)
	}

	if len(rebuilt) != len(before) {
		t.Fatalf("rebuilt views = %d, want %d (the replay must restore every stream)", len(rebuilt), len(before))
	}

	for _, view := range rebuilt {
		if view.NativeID == "" {
			t.Fatalf("rebuilt view has an empty native id: %+v", view)
		}
	}
}

func mustRecord(t *testing.T, obs calls.Observation) calls.RecordCallCmd {
	t.Helper()

	cmd, err := calls.NewRecordCommand(obs)
	if err != nil {
		t.Fatal(err)
	}

	return cmd
}

// viewCountReader is the ThenQuery-shaped fn over the view count.
func viewCountReader(sys *system.System) func() (any, error) {
	return func() (any, error) {
		views, err := readAllViewsForTest(context.Background(), sys)
		if err != nil {
			return nil, err
		}

		return len(views), nil
	}
}

func readAllViewsForTest(ctx context.Context, sys *system.System) ([]calls.CallView, error) {
	return metaengine.NewReader[calls.CallView](sys.MetaEngine(), calls.Collection).
		Scan(ctx, metaengine.WithLimit(0))
}
