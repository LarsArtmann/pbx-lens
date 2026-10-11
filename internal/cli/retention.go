package cli

import (
	"context"
	"time"

	"github.com/larsartmann/go-cqrs-lite/system/v4"
	"github.com/larsartmann/pbx-lens/internal/calls"
)

// RetentionReport summarizes one retention reset pass.
type RetentionReport struct {
	// Views is the number of call views present before the reset — the
	// population the pass dropped.
	Views int
	// ClearedEngines lists the engines whose materialized state was emptied.
	ClearedEngines []string
	// Partial reports engines that could not be cleared, leaving stale
	// read-model state behind. Every first-party engine clears, so a
	// partial reset means the retention promise is unenforceable and the
	// caller must fail.
	Partial bool
}

// ResetForRetention drops the whole materialized index — read model,
// journal, idempotency window — so re-ingest under a retention cutoff
// rebuilds only what the policy keeps. This is the index-side half of
// retention (T23): the sources are read-only and never touched, and the
// index is disposable by doctrine, so "purge" means reset plus a
// cutoff-respecting re-derivation — every ingest path filters through
// FilterWithinCutoff, so the index converges to "nothing older than the
// cutoff" and stays there. Callers follow with RebuildSearchIndex (the FTS
// table drops and rebuilds over the now-empty read model) and re-run their
// ingest commands with -keep.
func ResetForRetention(ctx context.Context, sys *system.System) (RetentionReport, error) {
	views, err := readAllViews(ctx, sys)
	if err != nil {
		return RetentionReport{}, err
	}

	result, err := sys.MetaEngine().Reset(ctx)
	if err != nil {
		return RetentionReport{}, err
	}

	return RetentionReport{
		Views:          len(views),
		ClearedEngines: result.ClearedEngines,
		Partial:        result.Partial(),
	}, nil
}

// FilterWithinCutoff keeps observations that started at or after the
// cutoff; a zero cutoff keeps everything unchanged. It is the policy point
// of retention: one-shot commands and the serve poller run their
// observations through this, so a re-derivation under a policy can never
// resurrect calls the policy dropped.
func FilterWithinCutoff(observations []calls.Observation, cutoff time.Time) (kept []calls.Observation, dropped int) {
	if cutoff.IsZero() {
		return observations, 0
	}

	for _, obs := range observations {
		if obs.StartedAt.Before(cutoff) {
			dropped++

			continue
		}

		kept = append(kept, obs)
	}

	return kept, dropped
}
