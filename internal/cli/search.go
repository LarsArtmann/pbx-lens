package cli

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/larsartmann/pbx-lens/internal/calls"
	"github.com/larsartmann/pbx-lens/internal/search"
)

// RebuildSearchIndex refreshes the disposable FTS shadow index inside the
// sqlite file at dsn, wholesale from the read model. A no-op when dsn is
// empty: memory deployments have no file to search, and the index is always
// re-derivable. Called by every ingest verb after DrainProjections, so the
// file and the read model can never drift.
func RebuildSearchIndex(ctx context.Context, sys *system.System, dsn string) error {
	if dsn == "" {
		return nil
	}

	index, err := search.Open(ctx, dsn)
	if err != nil {
		return err
	}

	defer func() { _ = index.Close() }()

	return RebuildIndex(ctx, sys, index)
}

// RebuildIndex refreshes an already-open index — the serve path, which holds
// one long-lived FTS connection instead of reopening per rebuild.
func RebuildIndex(ctx context.Context, sys *system.System, index *search.Index) error {
	views, err := readAllViews(ctx, sys)
	if err != nil {
		return err
	}

	return index.Rebuild(ctx, views)
}

// readAllViews returns every CallView in the read model. WithLimit(0) means
// unlimited; the scan default silently caps at 100.
func readAllViews(ctx context.Context, sys *system.System) ([]calls.CallView, error) {
	views, err := metaengine.NewReader[calls.CallView](sys.MetaEngine(), calls.Collection).
		Scan(ctx, metaengine.WithLimit(0))
	if err != nil {
		return nil, errorfamily.NewTransient(
			"lens.cli.search_scan",
			fmt.Sprintf("scanning call views for the search index: %v", err),
		)
	}

	return views, nil
}
