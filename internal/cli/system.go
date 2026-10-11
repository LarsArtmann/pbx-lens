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

// OpenSystem boots the composition root with the projection host running in
// the background. The returned cleanup stops the host and closes the system;
// call it exactly once, after the last read.
func OpenSystem(ctx context.Context, dsn string) (*system.System, func(), error) {
	sys, err := app.New(ctx, dsn)
	if err != nil {
		return nil, nil, err
	}

	hostCtx, stopHost := context.WithCancel(ctx)
	go func() { _ = sys.ProjectionHost().Start(hostCtx) }()

	return sys, func() {
		stopHost()

		_ = sys.Close()
	}, nil
}
