// Package cli holds the tested internals behind the pbx-lens command surface:
// flag-value parsing, system bootstrap, and the dispatch loop shared by the
// one-shot CLI commands and the serve poller. Nothing here prints; callers
// own output and exit codes.
package cli

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/system/v4"
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
// appended so far, SYNCHRONOUSLY: it restarts the host's batch-drain workers
// and waits for them to finish (the host is a one-shot drainer, not a live
// tail — Start alone returns before the drain runs). It is cheap when
// already caught up and safe to call repeatedly.
func DrainProjections(ctx context.Context, sys *system.System) error {
	host := sys.ProjectionHost()

	// Stop is idempotent on a fresh or stopped host; it resets the host to
	// the startable state.
	if err := host.Stop(); err != nil {
		return err
	}

	if err := host.Start(ctx); err != nil {
		return err
	}

	// The second Stop awaits the drain workers' graceful completion.
	return host.Stop()
}
