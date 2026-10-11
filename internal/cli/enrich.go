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
	seeds []EnrichmentSeed,
	since time.Time,
) (EnrichStats, error) {
	stats := EnrichStats{Considered: len(seeds)}
	if len(seeds) == 0 {
		return stats, nil
	}

	reader := metaengine.NewReader[calls.CallView](sys.MetaEngine(), calls.Collection)

	pass := enrichPass{client: client, since: since, reader: reader}

	for _, seed := range seeds {
		pass.enrichConversation(ctx, sys, seed, &stats)
	}

	return stats, nil
}

// enrichPass carries the per-pass state of one enrichment run: the lazy
// detail-records list (fetched at most once, only when a seed still lacks
// cost) and the typed reader over the projection.
type enrichPass struct {
	client         *telnyx.Client
	since          time.Time
	reader         *metaengine.TypedReader[calls.CallView]
	records        []telnyx.DetailRecord
	recordsFetched bool
}

func (p *enrichPass) enrichConversation(
	ctx context.Context,
	sys *system.System,
	seed EnrichmentSeed,
	stats *EnrichStats,
) {
	view, found, err := p.reader.Get(ctx, seed.StreamKey)
	if err != nil {
		stats.Rejected = append(stats.Rejected, seed.ConversationID+": "+err.Error())

		return
	}

	if !found {
		stats.Missing++

		return
	}

	current := enrichmentOf(view)

	enr, reject := p.targetEnrichment(ctx, seed, view, current)
	if reject != "" {
		stats.Rejected = append(stats.Rejected, reject)

		return
	}

	if enr == current {
		stats.Unchanged++

		return
	}

	if reject := dispatchEnrich(ctx, sys, seed, view, enr); reject != "" {
		stats.Rejected = append(stats.Rejected, reject)

		return
	}

	stats.Enriched++
}

// targetEnrichment resolves what the sources currently report for one seed,
// starting from the view's enrichment (already-final fields carry forward
// unfetched) and overwriting the halves that still need fetching. A
// non-empty reject string reports a fetch failure.
func (p *enrichPass) targetEnrichment(
	ctx context.Context,
	seed EnrichmentSeed,
	view calls.CallView,
	current calls.Enrichment,
) (calls.Enrichment, string) {
	enr := current

	if view.CostMicros == 0 && seed.CallControlID != "" {
		records, err := p.detailRecords(ctx)
		if err != nil {
			return current, seed.ConversationID + ": " + err.Error()
		}

		if rec, ok := telnyx.DetailRecordByCallControl(records, seed.CallControlID); ok {
			if micros, currency, costOK := telnyx.CostEnrichment(rec); costOK {
				enr.CostMicros = micros
				enr.CostCurrency = currency
			}
		}
	}

	if view.InsightSummary == "" {
		insights, err := p.client.ListConversationInsights(ctx, seed.ConversationID)
		if err != nil {
			return current, seed.ConversationID + ": " + err.Error()
		}

		enr.InsightSummary = telnyx.InsightSummary(insights)
	}

	return enr, ""
}

// detailRecords lazily fetches the pass's Detail Records list.
func (p *enrichPass) detailRecords(ctx context.Context) ([]telnyx.DetailRecord, error) {
	if p.recordsFetched {
		return p.records, nil
	}

	records, err := p.client.ListDetailRecords(ctx, p.since)
	if err != nil {
		return nil, err
	}

	p.records = records
	p.recordsFetched = true

	return records, nil
}

// dispatchEnrich appends call.enriched for one stream; a non-empty string
// reports the refusal.
func dispatchEnrich(
	ctx context.Context,
	sys *system.System,
	seed EnrichmentSeed,
	view calls.CallView,
	enr calls.Enrichment,
) string {
	cmd, err := calls.NewEnrichCommand(view.Observation(), enr)
	if err != nil {
		return seed.ConversationID + ": " + err.Error()
	}

	if err := sys.CommandDispatcher().Dispatch(ctx, cmd); err != nil {
		return seed.ConversationID + ": " + err.Error()
	}

	return ""
}

// enrichmentOf reads the view's current enrichment payload.
func enrichmentOf(view calls.CallView) calls.Enrichment {
	return calls.Enrichment{
		CostMicros:     view.CostMicros,
		CostCurrency:   view.CostCurrency,
		InsightSummary: view.InsightSummary,
	}
}
