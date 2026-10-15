package cli

import (
	"context"
	"fmt"
	"maps"
	"os"
	"sync"
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
	// AIDIDs is the deployment's AI-facing DID set; non-empty switches the
	// mapper to DID-based direction resolution (docs/did-direction-design.md).
	AIDIDs map[string]bool

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

	// Counters accumulates per-source ingest health for the /api/ops
	// endpoint (R15). Nil (tests, one-shot ingests) records nothing.
	Counters *SourceCounters
}

// SourceCounters accumulates the latest pass's per-source ingest health
// (R15): offered/recorded/corrected/rejected per source, served at
// /api/ops so ingest health is visible without shelling into the host.
// Every carries the poll cadence so the snapshot can say whether the
// reported pass is stale instead of leaving every consumer to derive it
// from a timestamp and out-of-band knowledge of the interval.
type SourceCounters struct {
	mu       sync.Mutex
	sources  map[string]SourceStat
	lastPass time.Time
	every    time.Duration
	now      func() time.Time
}

// SourceStat is one source's latest-pass ingest counters.
type SourceStat struct {
	Offered   int
	Recorded  int
	Corrected int
	Rejected  []string
}

// NewSourceCounters builds an empty counter set. every is the poll cadence
// the staleness verdict is measured against; zero or negative falls back to
// DefaultPollEvery.
func NewSourceCounters(every time.Duration) *SourceCounters {
	return &SourceCounters{sources: make(map[string]SourceStat), every: every, now: time.Now}
}

// staleAfterPasses is the staleness threshold in cadences: a pass older
// than two intervals means the poller missed a tick — hung, crashed, or
// the host slept. One missed tick is tolerated; two is a verdict.
const staleAfterPasses = 2

// stalePass is the pass-age verdict behind the ops payload's stale flag:
// how long ago the latest pass ran and whether that is already past the
// tolerated miss count. A zero lastPass (no pass ever recorded) is stale
// by definition — an ops endpoint answering before the first pass means
// the poller has not proven itself yet.
func stalePass(lastPass time.Time, every time.Duration, now time.Time) (age time.Duration, stale bool) {
	if every <= 0 {
		every = DefaultPollEvery
	}

	if lastPass.IsZero() {
		return -1, true
	}

	age = now.Sub(lastPass)

	return age, age > every*staleAfterPasses
}

// Record folds one pass's stats into the source's slot (latest pass wins;
// the ops endpoint answers "what did the poller last see").
func (c *SourceCounters) Record(source string, stats IngestStats) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.sources[source] = SourceStat{
		Offered:   stats.Total,
		Recorded:  stats.Recorded,
		Corrected: stats.Corrected,
		Rejected:  stats.Rejected,
	}
	c.lastPass = time.Now().UTC()
}

// Snapshot copies the counters for serving.
func (c *SourceCounters) Snapshot() (map[string]SourceStat, time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()

	out := make(map[string]SourceStat, len(c.sources))

	maps.Copy(out, c.sources)

	return out, c.lastPass
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

// PollOnce refreshes every configured source once. Logs are structured
// key-value records (R14) so an operator can grep the serve log.
func PollOnce(
	ctx context.Context,
	sys *system.System,
	index *search.Index,
	cfg PollConfig,
	log func(msg string, args ...any),
) {
	cdrStats := pollCDR(ctx, sys, cfg, log)
	webhookStats := pollWebhook(ctx, sys, cfg, log)
	transcriptStats := pollTranscripts(ctx, sys, cfg, log)
	telnyxStats := pollTelnyx(ctx, sys, cfg, log)

	if cfg.Counters != nil {
		cfg.Counters.Record("freeswitch-cdr", cdrStats)
		cfg.Counters.Record("webhook-log", webhookStats)
		cfg.Counters.Record("agent-transcript", transcriptStats)
		cfg.Counters.Record("telnyx-conversation", telnyxStats)
	}

	stats := mergeStats(
		mergeStats(mergeStats(mergeStats(IngestStats{}, cdrStats), webhookStats), transcriptStats),
		telnyxStats,
	)

	if err := DrainProjections(ctx, sys); err != nil {
		log("poll drain", "err", err)

		return
	}

	if index != nil {
		if err := RebuildOpenIndex(ctx, sys, index); err != nil {
			log("poll search rebuild", "err", err)
		}
	}

	log("poll pass", "offered", stats.Total, "recorded", stats.Recorded,
		"corrected", stats.Corrected, "rejected", len(stats.Rejected))

	for _, reason := range stats.Rejected {
		log("poll rejected", "reason", reason)
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
		log("poll cdr", "err", err)

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
		log("poll webhook", "err", err)

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
		log("poll transcripts", "err", err)

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
		AIDIDs:    cfg.AIDIDs,
	})
	if err != nil {
		log("poll telnyx", "err", err)

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
		log("poll telnyx drain", "err", err)

		return pass
	}

	estats, err := EnrichTelnyxCalls(ctx, sys, cfg.Telnyx, keptSeeds, since)
	if err != nil {
		log("poll telnyx enrich", "err", err)

		return pass
	}

	log("poll telnyx enrich", "considered", estats.Considered, "enriched", estats.Enriched,
		"unchanged", estats.Unchanged, "missing", estats.Missing, "rejected", len(estats.Rejected))

	for _, reason := range estats.Rejected {
		log("poll telnyx enrichment rejected", "reason", reason)
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

// SnapshotAny exposes the counters as the untyped payload the web layer
// serves at /api/ops (cli stays decoupled from the web package). The age
// and stale keys carry the pass-health verdict so consumers get an honest
// answer instead of deriving one from the timestamp: age is -1 when no
// pass has ever been recorded (and stale is then true by definition).
func (c *SourceCounters) SnapshotAny() any {
	c.mu.Lock()

	sources := make(map[string]SourceStat, len(c.sources))
	maps.Copy(sources, c.sources)
	lastPass, every, now := c.lastPass, c.every, c.now
	c.mu.Unlock()

	age, stale := stalePass(lastPass, every, now())

	return map[string]any{
		"sources":                sources,
		"last_pass":              lastPass,
		"last_pass_age_seconds":  age.Seconds(),
		"stale":                  stale,
	}
}
