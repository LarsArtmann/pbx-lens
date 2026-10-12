package main

import (
	"context"
	"time"

	"github.com/larsartmann/go-cqrs-lite/system/v4"
	"github.com/larsartmann/pbx-lens/internal/calls"
	"github.com/larsartmann/pbx-lens/internal/cli"
)

const ingestTimeout = 5 * time.Minute

// openIndexSession boots a one-shot index command: a bounded context plus
// the composition root, torn down in OpenSystem order (cleanup first,
// context cancel last).
func openIndexSession(db string) (context.Context, *system.System, func(), error) {
	ctx, cancel := context.WithTimeout(context.Background(), ingestTimeout)

	sys, cleanup, err := cli.OpenSystem(ctx, db)
	if err != nil {
		cancel()

		return nil, nil, nil, err
	}

	return ctx, sys, func() {
		cleanup()
		cancel()
	}, nil
}

// finalizeIngest is the shared tail of every ingest verb: catch the read
// model up, then refresh the search index so file and read model never
// drift.
func finalizeIngest(ctx context.Context, sys *system.System, dsn string) error {
	if err := cli.DrainProjections(ctx, sys); err != nil {
		return err
	}

	return cli.RebuildSearchIndex(ctx, sys, dsn)
}

// gateAndFinalize owns the sequence three of the four ingest verbs share
// (plan T12): open the index session, age-gate + ingest the observations,
// then drain and rebuild before teardown. The telnyx verb stays bespoke —
// its enrichment interleaves between two drains, so the linear shape does
// not fit it.
func gateAndFinalize(db string, maxAge time.Duration, observations []calls.Observation) cli.IngestStats {
	ctx, sys, cleanup, err := openIndexSession(db)
	if err != nil {
		fatal(err)
	}

	defer cleanup()

	stats := cli.GateIngest(ctx, sys, maxAge, observations)

	if err := finalizeIngest(ctx, sys, db); err != nil {
		fatal(err)
	}

	return stats
}
