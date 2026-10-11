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

	if cfg.CDRFile != "" {
		pass, err := pollFile(cfg.CDRFile, func(path *os.File) (IngestStats, error) {
			parsed, err := cdr.Parse(path)
			if err != nil {
				return IngestStats{}, err
			}

			return Ingest(ctx, sys, parsed.Observations), nil
		})
		if err != nil {
			log("poll cdr: %v", err)
		} else {
			stats = mergeStats(stats, pass)
		}
	}

	if cfg.WebhookLog != "" {
		pass, err := pollFile(cfg.WebhookLog, func(path *os.File) (IngestStats, error) {
			parsed, err := webhook.Parse(path)
			if err != nil {
				return IngestStats{}, err
			}

			return Ingest(ctx, sys, parsed.Observations), nil
		})
		if err != nil {
			log("poll webhook log: %v", err)
		} else {
			stats = mergeStats(stats, pass)
		}
	}

	if cfg.TranscriptsDir != "" {
		observations, rejects, err := FetchAgentObservations(ctx, sys, cfg.TranscriptsDir)
		if err != nil {
			log("poll transcripts: %v", err)
		} else {
			pass := Ingest(ctx, sys, observations)
			pass.Rejected = append(rejects, pass.Rejected...)
			stats = mergeStats(stats, pass)
		}
	}

	if cfg.Telnyx != nil {
		since := time.Now().UTC().Add(-cfg.SinceAgo)
		observations, rejects, err := FetchTelnyxObservations(ctx, cfg.Telnyx, TelnyxOptions{
			Since:     since,
			Direction: cfg.Direction,
		})
		if err != nil {
			log("poll telnyx: %v", err)
		} else {
			pass := Ingest(ctx, sys, observations)
			pass.Rejected = append(rejects, pass.Rejected...)
			stats = mergeStats(stats, pass)
		}
	}

	if err := DrainProjections(ctx, sys); err != nil {
		log("poll drain: %v", err)

		return
	}

	if index != nil {
		if err := RebuildIndex(ctx, sys, index); err != nil {
			log("poll search rebuild: %v", err)
		}
	}

	log("poll pass: %d offered, %d recorded, %d corrected, %d rejected",
		stats.Total, stats.Recorded, stats.Corrected, len(stats.Rejected))

	for _, reason := range stats.Rejected {
		log("  rejected: %s", reason)
	}
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
