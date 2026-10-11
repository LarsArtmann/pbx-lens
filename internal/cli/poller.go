package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/larsartmann/go-cqrs-lite/system/v4"
	"github.com/larsartmann/pbx-lens/internal/calls"
	"github.com/larsartmann/pbx-lens/internal/ingest/cdr"
	"github.com/larsartmann/pbx-lens/internal/ingest/telnyx"
	"github.com/larsartmann/pbx-lens/internal/ingest/webhook"
	"github.com/larsartmann/pbx-lens/internal/search"
)

// DefaultPollEvery is the serve poller cadence when -poll-every is unset.
const DefaultPollEvery = 15 * time.Minute

// DefaultPollSince is the Telnyx look-back window when -poll-since is unset.
const DefaultPollSince = 48 * time.Hour

// PollConfig names the sources one serve poller pass refreshes. Every
// source is optional; configured ones run in a fixed order (cheap local
// files first, then the network fetch) and a failure in one logs and skips
// to the next — a poller pass never takes the server down.
type PollConfig struct {
	Every time.Duration

	// Telnyx polls conversations+recordings when the client is set.
	Telnyx    *telnyx.Client
	SinceAgo  time.Duration
	Direction calls.Direction

	// CDRFile re-parses a FreeSWITCH Master.csv (idempotent re-ingest).
	CDRFile string

	// WebhookLog re-parses the receiver's JSONL tail file.
	WebhookLog string

	// TranscriptsDir re-scans agent transcript files.
	TranscriptsDir string

	// MaxAge gates ingest-side retention: an observation older than
	// now-MaxAge is refused with retention.ReasonAgeCutoff instead of
	// recorded, so over-age rows cannot re-enter an index the periodic
	// purge already cleaned. Zero keeps everything — retention is opt-in,
	// never silent.
	MaxAge time.Duration
}

// PollLoop runs poll passes on the configured cadence until ctx ends. The
// first pass fires immediately so a restarted serve converges at once;
// every pass drains projections and rebuilds the FTS index, so the read
// model and search can never drift from the event log.
func PollLoop(
	ctx context.Context,
	sys *system.System,
	index *search.Index,
	cfg PollConfig,
	log func(msg string, args ...any),
) {
	if cfg.Every <= 0 || log == nil {
		return
	}

	PollOnce(ctx, sys, index, cfg, log)

	ticker := time.NewTicker(cfg.Every)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			PollOnce(ctx, sys, index, cfg, log)
		}
	}
}

// PollOnce refreshes every configured source once.
func PollOnce(
	ctx context.Context,
	sys *system.System,
	index *search.Index,
	cfg PollConfig,
	log func(msg string, args ...any),
) {
	var stats IngestStats

	stats = mergeStats(stats, pollCDR(ctx, sys, cfg, log))
	stats = mergeStats(stats, pollWebhook(ctx, sys, cfg, log))
	stats = mergeStats(stats, pollTranscripts(ctx, sys, cfg, log))
	stats = mergeStats(stats, pollTelnyx(ctx, sys, cfg, log))

	if err := DrainProjections(ctx, sys); err != nil {
		log("poll drain: %v", err)

		return
	}

	if index != nil {
		if err := RebuildOpenIndex(ctx, sys, index); err != nil {
			log("poll search rebuild: %v", err)
		}
	}

	log("poll pass: %d offered, %d recorded, %d corrected, %d rejected",
		stats.Total, stats.Recorded, stats.Corrected, len(stats.Rejected))

	for _, reason := range stats.Rejected {
		log("  rejected: %s", reason)
	}
}

// pollCDR re-ingests the FreeSWITCH CDR file when configured, logging and
// swallowing any failure so one dead source never aborts a pass.
func pollCDR(
	ctx context.Context,
	sys *system.System,
	cfg PollConfig,
	log func(msg string, args ...any),
) IngestStats {
	if cfg.CDRFile == "" {
		return IngestStats{}
	}

	pass, err := pollFile(cfg.CDRFile, func(path *os.File) (IngestStats, error) {
		parsed, err := cdr.Parse(path)
		if err != nil {
			return IngestStats{}, err
		}

		return GateIngest(ctx, sys, cfg.MaxAge, parsed.Observations), nil
	})
	if err != nil {
		log("poll cdr: %v", err)

		return IngestStats{}
	}

	return pass
}

// pollWebhook re-ingests the receiver webhook JSONL when configured.
func pollWebhook(
	ctx context.Context,
	sys *system.System,
	cfg PollConfig,
	log func(msg string, args ...any),
) IngestStats {
	if cfg.WebhookLog == "" {
		return IngestStats{}
	}

	pass, err := pollFile(cfg.WebhookLog, func(path *os.File) (IngestStats, error) {
		parsed, err := webhook.Parse(path)
		if err != nil {
			return IngestStats{}, err
		}

		return GateIngest(ctx, sys, cfg.MaxAge, parsed.Observations), nil
	})
	if err != nil {
		log("poll webhook log: %v", err)

		return IngestStats{}
	}

	return pass
}

// pollTranscripts re-scans the agent transcript directory when configured.
func pollTranscripts(
	ctx context.Context,
	sys *system.System,
	cfg PollConfig,
	log func(msg string, args ...any),
) IngestStats {
	if cfg.TranscriptsDir == "" {
		return IngestStats{}
	}

	observations, rejects, err := FetchAgentObservations(ctx, sys, cfg.TranscriptsDir)
	if err != nil {
		log("poll transcripts: %v", err)

		return IngestStats{}
	}

	pass := GateIngest(ctx, sys, cfg.MaxAge, observations)
	pass.Rejected = append(rejects, pass.Rejected...)

	return pass
}

// pollTelnyx polls conversations+recordings when a client is configured,
// then runs the enrichment pass (Detail Records cost + conversation
// insights) over the same conversations.
func pollTelnyx(
	ctx context.Context,
	sys *system.System,
	cfg PollConfig,
	log func(msg string, args ...any),
) IngestStats {
	if cfg.Telnyx == nil {
		return IngestStats{}
	}

	since := time.Now().UTC().Add(-cfg.SinceAgo)

	observations, seeds, rejects, err := FetchTelnyxObservations(ctx, cfg.Telnyx, TelnyxOptions{
		Since:     since,
		Direction: cfg.Direction,
	})
	if err != nil {
		log("poll telnyx: %v", err)

		return IngestStats{}
	}

	now := time.Now().UTC()

	admitted, refusals := AgeGate{MaxAge: cfg.MaxAge}.Apply(observations, now)
	pass := ingestStatsFor(ctx, sys, observations, admitted, refusals)
	pass.Rejected = append(rejects, pass.Rejected...)

	// An observation refused by the age gate must not be enriched either:
	// its stream never lands, so keeping the seed would only burn a fetch
	// and retry as Missing every pass.
	keptSeeds := SeedsFor(admitted, seeds)

	// Enrichment reads the read model, so this pass's recordings must be
	// projected before it runs; the pass-end drain then covers the
	// enrichment events themselves.
	if err := DrainProjections(ctx, sys); err != nil {
		log("poll telnyx drain: %v", err)

		return pass
	}

	estats, err := EnrichTelnyxCalls(ctx, sys, cfg.Telnyx, keptSeeds, since)
	if err != nil {
		log("poll telnyx enrich: %v", err)

		return pass
	}

	log("poll telnyx enrich: %d considered, %d enriched, %d unchanged, %d missing, %d rejected",
		estats.Considered, estats.Enriched, estats.Unchanged, estats.Missing, len(estats.Rejected))

	for _, reason := range estats.Rejected {
		log("  enrichment rejected: %s", reason)
	}

	return pass
}

// ingestStatsFor ingests the admitted set and reports the refusals against
// the full offered batch — the one finisher the gated telnyx source shares
// with GateIngest.
func ingestStatsFor(
	ctx context.Context,
	sys *system.System,
	offered, admitted []calls.Observation,
	refusals []string,
) IngestStats {
	stats := Ingest(ctx, sys, admitted)
	stats.Total = len(offered)
	stats.Rejected = append(stats.Rejected, refusals...)

	return stats
}

// pollFile opens, feeds, and closes a file source.
func pollFile(path string, feed func(*os.File) (IngestStats, error)) (IngestStats, error) {
	file, err := os.Open(path)
	if err != nil {
		return IngestStats{}, fmt.Errorf("opening %s: %w", path, err)
	}

	stats, feedErr := feed(file)
	closeErr := file.Close()

	if feedErr != nil {
		return IngestStats{}, feedErr
	}

	return stats, closeErr
}

func mergeStats(into, from IngestStats) IngestStats {
	into.Total += from.Total
	into.Recorded += from.Recorded
	into.Corrected += from.Corrected
	into.Rejected = append(into.Rejected, from.Rejected...)

	return into
}
