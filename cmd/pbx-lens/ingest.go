package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/larsartmann/go-cqrs-lite/system/v4"
	"github.com/larsartmann/pbx-lens/internal/calls"
	"github.com/larsartmann/pbx-lens/internal/cli"
	"github.com/larsartmann/pbx-lens/internal/ingest/cdr"
	"github.com/larsartmann/pbx-lens/internal/ingest/telnyx"
	"github.com/larsartmann/pbx-lens/internal/ingest/webhook"
)

func ingestCDR(args []string) {
	fs := flag.NewFlagSet("ingest-cdr", flag.ExitOnError)
	db := fs.String("db", "", "sqlite DSN (required)")
	aiDest := fs.String("ai-dest", cdr.DefaultAIDestination,
		"comma-separated AI destinations (dialplan badge): destination_number matches badge IsAI")
	maxAge := fs.Duration("max-age", 0, "refuse observations older than this (0 = keep everything)")
	dryRun := fs.Bool("dry-run", false, "parse and report without touching the index (no -db needed)")
	_ = fs.Parse(args)

	if !*dryRun {
		requireDB(fs, *db)
	}

	requireOneArg(fs, "Master.csv path")

	file, err := os.Open(fs.Arg(0))
	if err != nil {
		fatal(err)
	}

	parsed, err := cdr.ParseWithOptions(file, cdr.ParseOptions{AIDestinations: cdr.ParseAIDestinations(*aiDest)})
	closeErr := file.Close()

	if err != nil {
		fatal(err)
	}

	if closeErr != nil {
		fatal(closeErr)
	}

	if *dryRun {
		admitted, refusals := (cli.AgeGate{MaxAge: *maxAge}).Apply(parsed.Observations, time.Now().UTC())

		fmt.Printf(
			"ingest-cdr (dry-run): %d rows parsed, %d parse-skipped, %d would-record, %d age-refused — nothing written\n",
			len(parsed.Observations),
			parsed.Skipped,
			len(admitted),
			len(refusals),
		)

		printRejected(refusals)

		return
	}

	stats := gateAndFinalize(
		*db,
		*maxAge,
		func(context.Context, *system.System) ([]calls.Observation, []string, error) {
			return parsed.Observations, nil, nil
		},
	)

	fmt.Printf(
		"ingest-cdr: %d rows parsed, %d parse-skipped, %d recorded, %d corrected, %d rejected\n",
		stats.Total, parsed.Skipped, stats.Recorded, stats.Corrected, len(stats.Rejected),
	)

	printRejected(stats.Rejected)
}

func ingestTelnyx(args []string) {
	fs := flag.NewFlagSet("ingest-telnyx", flag.ExitOnError)
	db := fs.String("db", "", "sqlite DSN (required)")
	keyFile := fs.String("key-file", "", "Telnyx API key file (default: $TELNYX_API_KEY)")
	since := fs.String("since", "48h", "look-back window: Go duration or RFC3339 timestamp")
	direction := fs.String(
		"direction",
		"outbound",
		"conversation direction (Telnyx metadata exposes none): outbound|inbound",
	)
	limit := fs.Int("limit", 0, "max conversations to fetch (0 = all)")
	aiDID := fs.String(
		"ai-did",
		"",
		"comma-separated AI-facing DIDs: conversations touching one are inbound (empty = keep declared direction)",
	)
	telnyxMaxAge := fs.Duration("max-age", 0, "refuse conversations older than this (0 = keep everything)")
	dryRun := fs.Bool("dry-run", false, "fetch and report without touching the index (no -db needed)")
	apiBase := fs.String("api-base", "", "override the Telnyx API base URL (stub rehearsals; empty = production)")
	_ = fs.Parse(args)

	if !*dryRun {
		requireDB(fs, *db)
	}

	key, err := cli.ResolveKey(*keyFile)
	if err != nil {
		fatal(err)
	}

	sinceAt, err := cli.ParseSince(*since)
	if err != nil {
		fatal(err)
	}

	dir, err := cli.ParseDirection(*direction)
	if err != nil {
		fatal(err)
	}

	opts := cli.TelnyxOptions{
		Since:     sinceAt,
		Direction: dir,
		AIDIDs:    telnyx.ParseAIDIDs(*aiDID),
		Limit:     *limit,
	}

	if *dryRun {
		// Report-only fetch (T14 parity with ingest-cdr): the counts ARE
		// the deliverable — conversations would fetch, which would record,
		// which the age gate would refuse — but no index opens, so neither
		// ingest nor enrichment runs (enrichment reads the read model).
		ctx, cancel := context.WithTimeout(context.Background(), ingestTimeout)
		defer cancel()

		observations, _, fetchRejects, err := cli.FetchTelnyxObservations(ctx, telnyx.NewClient(key, *apiBase), opts)
		if err != nil {
			fatal(err)
		}

		admitted, refusals := (cli.AgeGate{MaxAge: *telnyxMaxAge}).Apply(observations, time.Now().UTC())

		fmt.Printf(
			"ingest-telnyx (dry-run): %d conversations fetched, %d would-record, %d age-refused, %d fetch-rejected — nothing written\n",
			len(observations),
			len(admitted),
			len(refusals),
			len(fetchRejects),
		)

		printRejected(fetchRejects)
		printRejected(refusals)

		return
	}

	ctx, sys, cleanup, err := openIndexSession(*db)
	if err != nil {
		fatal(err)
	}

	defer cleanup()

	client := telnyx.NewClient(key, *apiBase)

	observations, seeds, fetchRejects, err := cli.FetchTelnyxObservations(ctx, client, opts)
	if err != nil {
		fatal(err)
	}

	admitted, ageRefusals := (cli.AgeGate{MaxAge: *telnyxMaxAge}).Apply(observations, time.Now().UTC())

	stats := cli.Ingest(ctx, sys, admitted)
	stats.Total = len(observations)
	stats.Rejected = append(stats.Rejected, ageRefusals...)

	if err := cli.DrainProjections(ctx, sys); err != nil {
		fatal(err)
	}

	// Enrichment reads the read model, so it runs after the drain that
	// projected this pass's recordings; the second drain covers the
	// call.enriched events the pass appends. Seeds whose observation the
	// age gate refused are dropped: enriching a stream that never lands
	// would count Missing and retry forever.
	estats, err := cli.EnrichTelnyxCalls(ctx, sys, client, cli.SeedsFor(admitted, seeds), sinceAt)
	if err != nil {
		fatal(err)
	}

	if err := finalizeIngest(ctx, sys, *db); err != nil {
		fatal(err)
	}

	stats.Rejected = append(fetchRejects, stats.Rejected...)
	stats.Total += len(fetchRejects)

	fmt.Printf(
		"ingest-telnyx: %d conversations fetched, %d recorded, %d corrected, %d rejected\n",
		stats.Total, stats.Recorded, stats.Corrected, len(stats.Rejected),
	)

	fmt.Printf(
		"enrichment: %d considered, %d enriched, %d unchanged, %d missing, %d rejected\n",
		estats.Considered, estats.Enriched, estats.Unchanged, estats.Missing, len(estats.Rejected),
	)

	printRejected(stats.Rejected)
	printRejected(estats.Rejected)
}

func ingestWebhook(args []string) {
	fs := flag.NewFlagSet("ingest-webhook", flag.ExitOnError)
	db := fs.String("db", "", "sqlite DSN (required)")
	webhookMaxAge := fs.Duration("max-age", 0, "refuse observations older than this (0 = keep everything)")
	_ = fs.Parse(args)

	requireDB(fs, *db)
	requireOneArg(fs, "inbound.jsonl path")

	file, err := os.Open(fs.Arg(0))
	if err != nil {
		fatal(err)
	}

	parsed, err := webhook.Parse(file)
	closeErr := file.Close()

	if err != nil {
		fatal(err)
	}

	if closeErr != nil {
		fatal(closeErr)
	}

	stats := gateAndFinalize(
		*db,
		*webhookMaxAge,
		func(context.Context, *system.System) ([]calls.Observation, []string, error) {
			return parsed.Observations, nil, nil
		},
	)

	fmt.Printf(
		"ingest-webhook: %d calls aggregated, %d lines skipped, %d recorded, %d corrected, %d rejected\n",
		stats.Total, parsed.Skipped, stats.Recorded, stats.Corrected, len(stats.Rejected),
	)

	printRejected(stats.Rejected)
}

func ingestAgent(args []string) {
	fs := flag.NewFlagSet("ingest-agent", flag.ExitOnError)
	db := fs.String("db", "", "sqlite DSN (required)")
	dir := fs.String(
		"dir",
		"/var/lib/telephony/recordings/transcripts",
		"transcripts directory ({call-uuid}.jsonl files)",
	)
	agentMaxAge := fs.Duration("max-age", 0, "refuse transcripts older than this (0 = keep everything)")
	_ = fs.Parse(args)

	requireDB(fs, *db)

	// The fetch needs the live session (the parser joins the CDR view by
	// call uuid), so the agent verb rides gateAndFinalize's fetch hook;
	// its per-file rejections fold into the stats inside the helper.
	stats := gateAndFinalize(
		*db,
		*agentMaxAge,
		func(ctx context.Context, sys *system.System) ([]calls.Observation, []string, error) {
			return cli.FetchAgentObservations(ctx, sys, *dir)
		},
	)

	fmt.Printf(
		"ingest-agent: %d transcripts ingested, %d recorded, %d corrected, %d rejected\n",
		stats.Total, stats.Recorded, stats.Corrected, len(stats.Rejected),
	)

	printRejected(stats.Rejected)
}
