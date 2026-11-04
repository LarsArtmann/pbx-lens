package app_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/systemscenario/v4"
	"github.com/larsartmann/pbx-lens/internal/app"
	"github.com/larsartmann/pbx-lens/internal/calls"
	"github.com/larsartmann/pbx-lens/internal/cli"
)

// TestHostResetRebuildsReadModel proves the SHORTER reset path the runbook
// recommends (docs/runbooks/reset-rebuild.md): projectionhost.Host.Reset takes
// the projection NAME, drops its checkpoint, and calls the projection's
// Resettable.Reset — which for a metaengine-backed projection delegates to the
// same Store.Reset the longer path calls directly. So one call does what
// "Store.Reset + remember to clear the checkpoint" does by hand, and the next
// drain replays the journal into a rebuilt read model.
//
// This is the test that settles the API question: Host.Reset DOES exist at
// projectionhost v4.7.1 (host_reset.go:76); an earlier session documented it as
// missing and reached for the store directly.
func TestHostResetRebuildsReadModel(t *testing.T) {
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
	second.NativeID = "conv-host-reset"
	second.StartedAt = first.StartedAt.Add(time.Minute)

	sc.Given(
		sc.Event(calls.EventObserved, streamRefOf(mustRecord(t, first)), calls.CallObserved{Observation: first}),
		sc.Event(calls.EventObserved, streamRefOf(mustRecord(t, second)), calls.CallObserved{Observation: second}),
	)

	sc.When(mustRecord(t, second)).
		ThenNoEvents().
		ThenQueryFunc(viewCountReader(sc.System()), func(got any) error {
			if count, ok := got.(int); !ok || count != 2 {
				return fmt.Errorf("%w: views = %v, want 2", errViewCount, got)
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

	// Host.Reset refuses while the host is running ("cannot reset while host
	// is running — Stop first"), so the sequence is Stop → Reset → drain.
	// DrainProjections opens with its own idempotent Stop, so the replay
	// needs no extra ceremony.
	if err := sys.ProjectionHost().Stop(); err != nil {
		t.Fatalf("stop before reset: %v", err)
	}

	if err := sys.ProjectionHost().Reset(ctx, "projections"); err != nil {
		t.Fatalf("host reset %q: %v", calls.Collection, err)
	}

	after, err := readAllViewsForTest(ctx, sys)
	if err != nil {
		t.Fatal(err)
	}

	if len(after) != 0 {
		t.Fatalf("post-reset views = %d, want 0 (the projection's Resettable cleared them)", len(after))
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
}
