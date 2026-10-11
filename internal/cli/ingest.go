package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
	"github.com/larsartmann/pbx-lens/internal/calls"
	"github.com/larsartmann/pbx-lens/internal/retention"
)

// IngestStats reports what one dispatch pass did to the index.
type IngestStats struct {
	// Total is the number of observations offered to the loop.
	Total int
	// Recorded counts dispatches that stored a new stream or replayed an
	// identical snapshot (idempotent no-op).
	Recorded int
	// Corrected counts dispatches that superseded an older snapshot of an
	// existing stream with call.corrected.
	Corrected int
	// Rejected carries one entry per refused observation, prefixed with the
	// source label, so operators can trace a reject to its source row.
	Rejected []string
}

// Ingest dispatches call.record for every observation and classifies the
// outcome. Sources stay read-only: the only writes land on the lens's own
// event log, where re-dispatching an unchanged observation is a no-op.
//
// Corrected classification reads the projection before dispatch, so a
// lagging projection can under-count corrections on back-to-back passes;
// the count is operator display, never correctness.
func Ingest(ctx context.Context, sys *system.System, observations []calls.Observation) IngestStats {
	reader := metaengine.NewReader[calls.CallView](sys.MetaEngine(), calls.Collection)

	stats := IngestStats{Total: len(observations)}

	for _, obs := range observations {
		cmd, err := calls.NewRecordCommand(obs)
		if err != nil {
			stats.Rejected = append(stats.Rejected, streamLabel(obs)+": "+err.Error())

			continue
		}

		supersedes := false
		if view, found, viewErr := reader.Get(ctx, cmd.StreamID().String()); viewErr == nil && found {
			supersedes = view.ContentHash != obs.ContentHash()
		}

		if err := sys.CommandDispatcher().Dispatch(ctx, cmd); err != nil {
			stats.Rejected = append(stats.Rejected, streamLabel(obs)+": "+err.Error())

			continue
		}

		if supersedes {
			stats.Corrected++
		} else {
			stats.Recorded++
		}
	}

	return stats
}

func streamLabel(obs calls.Observation) string {
	return fmt.Sprintf("%s:%s", obs.Kind, obs.NativeID)
}

// AgeGate applies the retention window to a source batch at ingest:
// observations older than MaxAge are refused with a labeled rejection
// instead of recorded (the ingest-side half of the retention doctrine; the
// purge half is retention.Retire). The zero value keeps everything —
// retention is opt-in, matching the Policy zero value. Shared by the serve
// poller and the one-shot ingest commands so a CLI backfill cannot
// silently re-import rows the window refuses.
type AgeGate struct {
	MaxAge time.Duration
}

// Apply splits a source batch into admitted observations and labeled
// refusals (retention.ReasonAgeCutoff).
func (g AgeGate) Apply(observations []calls.Observation, now time.Time) ([]calls.Observation, []string) {
	if g.MaxAge <= 0 {
		return observations, nil
	}

	policy := retention.Policy{MaxAge: g.MaxAge}

	admitted := make([]calls.Observation, 0, len(observations))
	refusals := make([]string, 0)

	for _, obs := range observations {
		if policy.Admits(obs, now) {
			admitted = append(admitted, obs)

			continue
		}

		refusals = append(refusals, streamLabel(obs)+": "+string(retention.ReasonAgeCutoff))
	}

	return admitted, refusals
}

// GateIngest applies the age gate and ingests the survivors. Total counts
// everything offered, so refusals report against the real batch size.
func GateIngest(
	ctx context.Context,
	sys *system.System,
	maxAge time.Duration,
	observations []calls.Observation,
) IngestStats {
	admitted, refusals := AgeGate{MaxAge: maxAge}.Apply(observations, time.Now().UTC())

	stats := Ingest(ctx, sys, admitted)
	stats.Total = len(observations)
	stats.Rejected = append(stats.Rejected, refusals...)

	return stats
}
