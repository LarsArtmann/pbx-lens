package cli

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
	"github.com/larsartmann/pbx-lens/internal/calls"
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
