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

// ResetReadModel wipes the call_views read model and replays the journal into
// it — the one-call rebuild behind docs/runbooks/reset-rebuild.md. It is the
// short path whose existence an earlier session denied: projectionhost.Host
// DOES expose Reset, and the metaengine projection adapter implements
// Resettable, so the host does checkpoint + read-model teardown in one call
// instead of the "Store.Reset, then remember to clear the checkpoint by hand"
// sequence. Proof: TestHostResetRebuildsReadModel.
//
// The journal is never touched (ADR-0143): events are facts and the replay
// source, so a reset that shares an engine with the journal loses nothing.
// Host.Reset refuses while the host runs, hence the Stop first; DrainProjections
// opens with its own idempotent Stop, so the replay needs no extra ceremony.
//
// The caller owns serving state: run this with the writer stopped, or a poll
// landing mid-rebuild re-folds onto partially cleared engines.
func ResetReadModel(ctx context.Context, sys *system.System) error {
	if err := sys.ProjectionHost().Stop(); err != nil {
		return err
	}

	return sys.ResetProjection(ctx, app.ProjectionName)
}

// drainPollInterval is the status-poll cadence while awaiting drain workers.
const drainPollInterval = 2 * time.Millisecond

// workerRoundState classifies a drain round: allDone (every worker reached
// WorkerStopped) and anyFailed (some worker exhausted its restart budget).
// See ProjectionWorkersReady for the OPPOSITE reading of the same states —
// a stopped worker is drain success here and health there. Do not unify the
// two predicates: drain wants "the batch finished", readiness wants "nothing
// is broken", and in a drain-then-exit host those are different questions.
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

// ProjectionWorkersReady is the serve readiness predicate (plan T17). It
// answers ONE question: is anything broken? A worker in WorkerBackoff (about
// to restart, budget not yet spent) or WorkerFailed (budget spent, read model
// permanently behind) is broken and votes NOT ready, so /health/ready answers
// 503 until that clears. Every other state is healthy in THIS architecture:
// the projection host is a batch drainer, so workers are WorkerIdle before the
// first drain, WorkerRunning/WorkerLive during it, WorkerDraining while
// in-flight events finish, and WorkerStopped once caught up — the serve poller
// re-drains on its tick, so a healthy process spends most of its life stopped.
// An empty list is vacuously ready.
//
// This is the deliberate inverse of workerRoundState above: drain success
// requires WorkerStopped, readiness requires the absence of failure. Stopped
// workers are why the first cut of this predicate answered 503 on a healthy
// serve (scripts/verify-ui-contract.js Contract F caught it).
func ProjectionWorkersReady(states []projectionhost.WorkerState) bool {
	for _, state := range states {
		if workerBroken(state.Status) {
			return false
		}
	}

	return true
}

// workerBroken names the two failure states explicitly and lists the healthy
// ones, so adding a state to the library forces a decision here (the
// exhaustive linter enforces the enumeration) instead of silently inheriting
// one. Backoff is broken because the read model is behind until the restart
// lands; failed is broken because it never will.
func workerBroken(status projectionhost.WorkerStatus) bool {
	switch status {
	case projectionhost.WorkerBackoff, projectionhost.WorkerFailed:
		return true
	case projectionhost.WorkerIdle,
		projectionhost.WorkerRunning,
		projectionhost.WorkerLive,
		projectionhost.WorkerDraining,
		projectionhost.WorkerStopped:
		return false
	}

	return false
}
