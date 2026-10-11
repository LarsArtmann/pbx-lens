// Package cli holds the tested internals behind the pbx-lens command surface:
// flag-value parsing, system bootstrap, and the dispatch loop shared by the
// one-shot CLI commands and the serve poller. Nothing here prints; callers
// own output and exit codes.
package cli

import (
	"context"
	"time"

	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/larsartmann/pbx-lens/internal/app"
)

// OpenSystem boots the composition root. It deliberately does NOT start the
// projection host: the host is a batch drainer, not a live tail — one Start
// call drains the journal from the last checkpoint and its workers exit, so
// events appended after a drain stay unprojected until the next Start.
// Callers own the drain cadence: one-shot commands call DrainProjections
// after dispatching; the serve poller re-Drains on its tick.
func OpenSystem(ctx context.Context, dsn string) (*system.System, func(), error) {
	sys, err := app.New(ctx, dsn)
	if err != nil {
		return nil, nil, err
	}

	return sys, func() {
		_ = sys.Close()
	}, nil
}

// DrainProjections catches the call_views read model up with every event
// appended so far, synchronously. The host is a one-shot batch drainer, not
// a live tail: Start launches workers that drain from the last checkpoint
// and exit (WorkerStopped) when caught up, and Start refuses to run while a
// previous round is still marked started. So one drain = reset (idempotent
// Stop), launch, then await every worker's completion. Cheap when already
// caught up; safe to call repeatedly.
func DrainProjections(ctx context.Context, sys *system.System) error {
	host := sys.ProjectionHost()

	if err := host.Stop(); err != nil {
		return err
	}

	if err := host.Start(ctx); err != nil {
		return err
	}

	ticker := time.NewTicker(drainPollInterval)
	defer ticker.Stop()

	for {
		allDone, anyFailed := workerRoundState(host.Status())
		if anyFailed {
			return errorfamily.NewTransient(
				"lens.cli.projection_worker_failed",
				"a projection worker failed during drain; check the host logs",
			)
		}

		if allDone {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

// drainPollInterval is the status-poll cadence while awaiting drain workers.
const drainPollInterval = 2 * time.Millisecond

func workerRoundState(states []projectionhost.WorkerState) (bool, bool) {
	allDone, anyFailed := true, false

	for _, state := range states {
		switch state.Status {
		case projectionhost.WorkerStopped:
		case projectionhost.WorkerFailed:
			anyFailed = true
		case projectionhost.WorkerIdle,
			projectionhost.WorkerRunning,
			projectionhost.WorkerLive,
			projectionhost.WorkerBackoff,
			projectionhost.WorkerDraining:
			// WorkerIdle here means "freshly launched, not drained yet";
			// the others mean mid-drain. Only WorkerStopped proves the
			// round finished.
			allDone = false
		}
	}

	return allDone, anyFailed
}
