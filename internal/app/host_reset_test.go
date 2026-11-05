package app_test

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/pbx-lens/internal/app"
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

	sys, before := seedFoldedPair(t, ctx, "conv-host-reset")

	// Host.Reset refuses while the host is running ("cannot reset while host
	// is running — Stop first"), so the sequence is Stop → Reset → drain.
	// cli.ResetReadModel owns the first two steps; DrainProjections opens with
	// its own idempotent Stop, so the replay needs no extra ceremony.
	if err := cli.ResetReadModel(ctx, sys); err != nil {
		t.Fatalf("reset read model %q: %v", app.ProjectionName, err)
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
