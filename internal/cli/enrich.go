package cli

import (
	"context"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
	"github.com/larsartmann/pbx-lens/internal/calls"
	"github.com/larsartmann/pbx-lens/internal/ingest/telnyx"
)

// EnrichStats reports what one enrichment pass did to the index.
type EnrichStats struct {
	// Considered is the number of seeds offered to the pass.
	Considered int
	// Enriched counts dispatches that stored a changed call.enriched.
	Enriched int
	// Unchanged counts streams whose enrichment already matches what the
	// sources currently report (the idempotent replay-to-zero case).
	Unchanged int
	// Missing counts seeds whose observation has not landed in the
	// projection yet — the enrich-before-observed rejection path, retried
	// by the next pass.
	Missing int
	// Rejected carries one entry per refused enrichment, prefixed with the
	// conversation id.
	Rejected []string
}

// EnrichTelnyxCalls pulls Telnyx Detail Records (one list, correlated
// in-process by call_control_id) and per-conversation insights, then
// dispatches call.enrich for streams whose enrichment differs from what the
// sources currently report. Already-enriched fields skip their fetch: cost
// and a completed summary are final snapshots, so re-polling the same
// window costs nothing once every conversation carries both (the detail
// list itself is fetched lazily, only when a seed still lacks cost).
//
// Callers must drain projections first: enrichment reads the read model, so
// observations recorded in the same pass must be projected before this runs
// (otherwise they count Missing and retry next pass).
func EnrichTelnyxCalls(
	ctx context.Context,
	sys *system.System,
	client *telnyx.Client,
	seeds []TelnyxSeed,
	since time.Time,
) (EnrichStats, error) {
	stats := EnrichStats{Considered: len(seeds)}
	if len(seeds) == 0 {
		return stats, nil
	}

	reader := metaengine.NewReader[calls.CallView](sys.MetaEngine(), calls.Collection)

	// The detail-records list is fetched lazily: only when a seed actually
	// lacks cost, and at most once per pass.
	var (
		records       []telnyx.DetailRecord
		detailFetched bool
	)

	fetchDetails := func() error {
		if detailFetched {
			return nil
		}

		recs, err := client.DetailRecords(ctx, since)
		if err != nil {
			return err
		}

		records = recs
		detailFetched = true

		return nil
	}

	for _, seed := range seeds {
		view, found, getErr := reader.Get(ctx, seed.StreamKey)
		if getErr != nil {
			stats.Rejected = append(stats.Rejected, seed.ConversationID+": "+getErr.Error())

			continue
		}

		if !found {
			stats.Missing++

			continue
		}

		current := calls.Enrichment{
			CostMicros:     view.CostMicros,
			CostCurrency:   view.CostCurrency,
			InsightSummary: view.InsightSummary,
		}

		enr := current

		if view.CostMicros == 0 && seed.CallControlID != "" {
			if fetchErr := fetchDetails(); fetchErr != nil {
				stats.Rejected = append(stats.Rejected, seed.ConversationID+": "+fetchErr.Error())

				continue
			}

			if rec, ok := telnyx.DetailRecordByCallControl(records, seed.CallControlID); ok {
				if micros, currency, costOK := telnyx.CostEnrichment(rec); costOK {
					enr.CostMicros = micros
					enr.CostCurrency = currency
				}
			}
		}

		if view.InsightSummary == "" {
			insights, insErr := client.ConversationInsights(ctx, seed.ConversationID)
			if insErr != nil {
				stats.Rejected = append(stats.Rejected, seed.ConversationID+": "+insErr.Error())

				continue
			}

			enr.InsightSummary = telnyx.InsightSummary(insights)
		}

		if enr == current {
			stats.Unchanged++

			continue
		}

		cmd, cmdErr := calls.NewEnrichCommand(view.Observation(), enr)
		if cmdErr != nil {
			stats.Rejected = append(stats.Rejected, seed.ConversationID+": "+cmdErr.Error())

			continue
		}

		if err := sys.CommandDispatcher().Dispatch(ctx, cmd); err != nil {
			stats.Rejected = append(stats.Rejected, seed.ConversationID+": "+err.Error())

			continue
		}

		stats.Enriched++
	}

	return stats, nil
}
