package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	appkit "github.com/larsartmann/go-appkit"
	"github.com/larsartmann/go-appkit/systemd"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/larsartmann/pbx-lens/internal/backup"
	"github.com/larsartmann/pbx-lens/internal/calls"
	"github.com/larsartmann/pbx-lens/internal/cli"
	"github.com/larsartmann/pbx-lens/internal/ingest/cdr"
	"github.com/larsartmann/pbx-lens/internal/ingest/telnyx"
	"github.com/larsartmann/pbx-lens/internal/ingest/webhook"
	"github.com/larsartmann/pbx-lens/internal/retention"
	"github.com/larsartmann/pbx-lens/internal/search"
	"github.com/larsartmann/pbx-lens/internal/web"
)

const (
	version = "0.1.0"

	minArgs   = 2
	exitUsage = 2

	defaultAddr   = "127.0.0.1:8300"
	ingestTimeout = 5 * time.Minute
	serveTimeout  = 5 * time.Minute
	smokeTimeout  = 30 * time.Second

	projectionDeadline     = 5 * time.Second
	projectionPollInterval = 20 * time.Millisecond

	smokeDurationSecs = 26
)

func main() {
	if len(os.Args) < minArgs {
		usage()
		os.Exit(exitUsage)
	}

	switch os.Args[1] {
	case "version":
		fmt.Println("pbx-lens " + version)
	case "smoke":
		smoke(os.Args[minArgs:])
	case "ingest-cdr":
		ingestCDR(os.Args[minArgs:])
	case "ingest-telnyx":
		ingestTelnyx(os.Args[minArgs:])
	case "ingest-webhook":
		ingestWebhook(os.Args[minArgs:])
	case "ingest-agent":
		ingestAgent(os.Args[minArgs:])
	case "serve":
		serve(os.Args[minArgs:])
	case "backup":
		backupCmd(os.Args[minArgs:])
	case "purge":
		purgeCmd(os.Args[minArgs:])
	default:
		usage()
		os.Exit(exitUsage)
	}
}

func usage() {
	fmt.Fprintln(
		os.Stderr,
		"usage: pbx-lens version | pbx-lens smoke [-db path] | pbx-lens ingest-cdr -db <dsn> <Master.csv> |"+
			" pbx-lens ingest-telnyx -db <dsn> [-key-file path] [-since 48h] [-direction outbound] [-limit N] |"+
			" pbx-lens ingest-webhook -db <dsn> <inbound.jsonl> |"+
			" pbx-lens ingest-agent -db <dsn> -dir /var/lib/telephony/recordings/transcripts |"+
			" pbx-lens serve -db <dsn> [-addr 127.0.0.1:8300] |"+
			" pbx-lens backup -db <dsn> -out <snapshot.db>",
	)
}

// requireDB enforces the mandatory -db flag every index-opening subcommand
// shares.
func requireDB(fs *flag.FlagSet, db string) {
	if db == "" {
		fatal(errorfamily.NewRejection("lens.cli.missing_db", fs.Name()+" needs -db <dsn>"))
	}
}

// requireOneArg enforces the single positional argument of the file-ingesting
// subcommands.
func requireOneArg(fs *flag.FlagSet, what string) {
	if fs.NArg() != 1 {
		fatal(errorfamily.NewRejection("lens.cli.bad_args", fs.Name()+" needs exactly one "+what))
	}
}

func ingestCDR(args []string) {
	fs := flag.NewFlagSet("ingest-cdr", flag.ExitOnError)
	db := fs.String("db", "", "sqlite DSN (required)")
	aiDest := fs.String("ai-dest", cdr.DefaultAIDestination,
		"comma-separated AI destinations (dialplan badge): destination_number matches badge IsAI")
	_ = fs.Parse(args)

	requireDB(fs, *db)
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

	ctx, cancel := context.WithTimeout(context.Background(), ingestTimeout)
	defer cancel()

	sys, cleanup, err := cli.OpenSystem(ctx, *db)
	if err != nil {
		fatal(err)
	}

	defer cleanup()

	stats := cli.Ingest(ctx, sys, parsed.Observations)

	if err := cli.DrainProjections(ctx, sys); err != nil {
		fatal(err)
	}

	if err := cli.RebuildSearchIndex(ctx, sys, *db); err != nil {
		fatal(err)
	}

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
	_ = fs.Parse(args)

	requireDB(fs, *db)

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

	ctx, cancel := context.WithTimeout(context.Background(), ingestTimeout)
	defer cancel()

	sys, cleanup, err := cli.OpenSystem(ctx, *db)
	if err != nil {
		fatal(err)
	}

	defer cleanup()

	client := telnyx.NewClient(key, "")

	observations, seeds, fetchRejects, err := cli.FetchTelnyxObservations(ctx, client, cli.TelnyxOptions{
		Since:     sinceAt,
		Direction: dir,
		Limit:     *limit,
	})
	if err != nil {
		fatal(err)
	}

	stats := cli.Ingest(ctx, sys, observations)

	if err := cli.DrainProjections(ctx, sys); err != nil {
		fatal(err)
	}

	// Enrichment reads the read model, so it runs after the drain that
	// projected this pass's recordings; the second drain covers the
	// call.enriched events the pass appends.
	estats, err := cli.EnrichTelnyxCalls(ctx, sys, client, seeds, sinceAt)
	if err != nil {
		fatal(err)
	}

	if err := cli.DrainProjections(ctx, sys); err != nil {
		fatal(err)
	}

	if err := cli.RebuildSearchIndex(ctx, sys, *db); err != nil {
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

	ctx, cancel := context.WithTimeout(context.Background(), ingestTimeout)
	defer cancel()

	sys, cleanup, err := cli.OpenSystem(ctx, *db)
	if err != nil {
		fatal(err)
	}

	defer cleanup()

	stats := cli.Ingest(ctx, sys, parsed.Observations)

	if err := cli.DrainProjections(ctx, sys); err != nil {
		fatal(err)
	}

	if err := cli.RebuildSearchIndex(ctx, sys, *db); err != nil {
		fatal(err)
	}

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
	_ = fs.Parse(args)

	requireDB(fs, *db)

	ctx, cancel := context.WithTimeout(context.Background(), ingestTimeout)
	defer cancel()

	sys, cleanup, err := cli.OpenSystem(ctx, *db)
	if err != nil {
		fatal(err)
	}

	defer cleanup()

	observations, rejects, err := cli.FetchAgentObservations(ctx, sys, *dir)
	if err != nil {
		fatal(err)
	}

	stats := cli.Ingest(ctx, sys, observations)

	if err := cli.DrainProjections(ctx, sys); err != nil {
		fatal(err)
	}

	if err := cli.RebuildSearchIndex(ctx, sys, *db); err != nil {
		fatal(err)
	}

	stats.Rejected = append(rejects, stats.Rejected...)

	fmt.Printf(
		"ingest-agent: %d transcripts ingested, %d recorded, %d corrected, %d rejected\n",
		len(observations), stats.Recorded, stats.Corrected, len(stats.Rejected),
	)

	printRejected(stats.Rejected)
}

func smoke(args []string) {
	fs := flag.NewFlagSet("smoke", flag.ExitOnError)
	db := fs.String("db", "", "sqlite DSN (default: in-memory)")
	_ = fs.Parse(args)

	ctx, cancel := context.WithTimeout(context.Background(), smokeTimeout)
	defer cancel()

	sys, cleanup, err := cli.OpenSystem(ctx, *db)
	if err != nil {
		fatal(err)
	}

	defer cleanup()

	obs := calls.Observation{
		NativeID:          "smoke-0001",
		Kind:              calls.SourceTelnyxConversation,
		Direction:         calls.DirectionOutbound,
		StartedAt:         time.Now().UTC().Truncate(time.Second),
		DurationSecs:      smokeDurationSecs,
		FromNumber:        "+17287289311",
		ToNumber:          "+15555550123",
		HangupCause:       "normal_clearing",
		IsAI:              true,
		TranscriptExcerpt: "Hi Lars! This is Woolly, calling from your own PBX.",
	}

	cmd, err := calls.NewRecordCommand(obs)
	if err != nil {
		fatal(err)
	}

	if err := sys.CommandDispatcher().Dispatch(ctx, cmd); err != nil {
		fatal(err)
	}

	if err := sys.CommandDispatcher().Dispatch(ctx, cmd); err != nil {
		fatal(err)
	}

	if err := cli.DrainProjections(ctx, sys); err != nil {
		fatal(err)
	}

	reader := metaengine.NewReader[calls.CallView](sys.MetaEngine(), calls.Collection)
	deadline := time.Now().Add(projectionDeadline)

	for {
		view, found, err := reader.Get(ctx, cmd.StreamID().String())
		if err != nil {
			fatal(err)
		}

		if found {
			out, err := json.MarshalIndent(view, "", "  ")
			if err != nil {
				fatal(err)
			}

			fmt.Println(string(out))

			return
		}

		if time.Now().After(deadline) {
			fatal(errorfamily.NewTransient("lens.cli.projection_timeout", "projection did not converge within 5s"))
		}

		time.Sleep(projectionPollInterval)
	}
}

// backupCmd writes and verifies a consistent sqlite snapshot for the nightly
// backup staging (C4/T14). It is safe to run against a live serve database.
func backupCmd(args []string) {
	fs := flag.NewFlagSet("backup", flag.ExitOnError)
	db := fs.String("db", "", "sqlite DSN (required)")
	out := fs.String("out", "", "snapshot output path (required)")
	_ = fs.Parse(args)

	requireDB(fs, *db)

	if *out == "" {
		fatal(errorfamily.NewRejection("lens.cli.missing_out", "backup needs -out <path>"))
	}

	ctx, cancel := context.WithTimeout(context.Background(), ingestTimeout)
	defer cancel()

	if err := backup.Snapshot(ctx, *db, *out); err != nil {
		fatal(err)
	}

	if err := backup.Verify(ctx, *out); err != nil {
		fatal(err)
	}

	fmt.Printf("snapshot written and verified: %s\n", *out)
}

// serve runs the read-only web UI until SIGINT/SIGTERM (appkit Run owns the
// signal handling and the graceful drain). One drain at startup catches the
// read model up with whatever the ingest commands left unprojected.
func serve(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	db := fs.String("db", "", "sqlite DSN (required)")
	addr := fs.String("addr", defaultAddr, "listen address")
	keyFile := fs.String("telnyx-key-file", "", "Telnyx API key file for the recording proxy (default: disabled)")
	recordingsDir := fs.String(
		"recordings-dir",
		"/var/lib/telephony/recordings",
		"local FreeSWITCH recordings directory ({uuid}_{dest}.wav files; empty = disabled)",
	)
	htpasswdPath := fs.String(
		"htpasswd",
		"",
		"bcrypt htpasswd file gating every route (htpasswd -B; empty = no auth for localhost dev)",
	)
	pollEvery := fs.Duration(
		"poll-every",
		cli.DefaultPollEvery,
		"poll cadence for keeping the index fresh (0 = disabled)",
	)
	pollSince := fs.Duration(
		"poll-since",
		cli.DefaultPollSince,
		"Telnyx look-back window per poll",
	)
	pollDirection := fs.String(
		"poll-direction",
		"outbound",
		"caller-declared direction for polled Telnyx conversations",
	)
	cdrFile := fs.String("cdr-file", "", "FreeSWITCH Master.csv re-ingested every poll (empty = skip)")
	webhookLog := fs.String("webhook-log", "", "receiver webhook JSONL re-ingested every poll (empty = skip)")
	transcriptsDir := fs.String("transcripts-dir", "", "agent transcripts dir re-scanned every poll (empty = skip)")
	_ = fs.Parse(args)

	requireDB(fs, *db)

	pollDir, err := cli.ParseDirection(*pollDirection)
	if err != nil {
		fatal(err)
	}

	setupCtx, cancel := context.WithTimeout(context.Background(), serveTimeout)
	defer cancel()

	sys, cleanup, err := cli.OpenSystem(setupCtx, *db)
	if err != nil {
		fatal(err)
	}

	defer cleanup()

	if err := cli.DrainProjections(setupCtx, sys); err != nil {
		fatal(err)
	}

	index, err := search.Open(setupCtx, *db)
	if err != nil {
		fatal(err)
	}

	defer func() { _ = index.Close() }()

	if err := cli.RebuildIndex(setupCtx, sys, index); err != nil {
		fatal(err)
	}

	serviceConfig := appkit.ServiceConfig{Addr: *addr}

	// sd_notify (T7/F23): READY=1 after the listener binds, STOPPING=1 at
	// the drain, WATCHDOG keepalive while serving. A no-op outside systemd
	// (NOTIFY_SOCKET unset), so the same binary serves localhost dev and
	// the service manager unchanged.
	systemd.Install(&serviceConfig)

	if *htpasswdPath != "" {
		auth, err := web.HtpasswdMiddleware(*htpasswdPath)
		if err != nil {
			fatal(err)
		}

		serviceConfig.ExtraMiddlewares = append(serviceConfig.ExtraMiddlewares, auth)
	}

	svc, err := appkit.NewService(serviceConfig)
	if err != nil {
		fatal(err)
	}

	recordings := web.RecordingOptions{RecordingsDir: *recordingsDir}

	var telnyxClient *telnyx.Client

	if *keyFile != "" {
		key, err := cli.ResolveKey(*keyFile)
		if err != nil {
			fatal(err)
		}

		telnyxClient = telnyx.NewClient(key, "")
		recordings.Telnyx = telnyxClient
	}

	web.Register(svc.Mux, sys, web.Options{Recordings: recordings, Search: index})

	// The poller keeps the index fresh until shutdown; each pass drains
	// projections and rebuilds the FTS index so / and /search never serve
	// yesterday's state while serve runs.
	pollCtx, stopPoller := context.WithCancel(context.Background())
	defer stopPoller()

	go cli.PollLoop(pollCtx, sys, index, cli.PollConfig{
		Every:          *pollEvery,
		Telnyx:         telnyxClient,
		SinceAgo:       *pollSince,
		Direction:      pollDir,
		CDRFile:        *cdrFile,
		WebhookLog:     *webhookLog,
		TranscriptsDir: *transcriptsDir,
	}, log.Printf)

	// The serve lifetime is signal-owned (Run handles SIGINT/SIGTERM) — no
	// timeout context here, or the server would die with it.
	if err := svc.Run(context.Background()); err != nil {
		fatal(err)
	}
}

func printRejected(rejected []string) {
	for _, reason := range rejected {
		fmt.Println("  rejected:", reason)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "pbx-lens:", err)
	os.Exit(1)
}

func purgeCmd(args []string) {
	fs := flag.NewFlagSet("purge", flag.ExitOnError)
	db := fs.String("db", "", "sqlite DSN (required)")
	maxAge := fs.String(
		"max-age",
		"",
		"retention window: Go duration (e.g. 2160h = 90 days); calls observed before now-minus-window are retired (required)",
	)
	dryRun := fs.Bool("dry-run", false, "list what would be retired without changing anything")
	_ = fs.Parse(args)

	requireDB(fs, *db)

	if *maxAge == "" {
		fatal(errorfamily.NewRejection(
			"lens.purge.missing_max_age",
			fs.Name()+" needs -max-age <duration> (e.g. 720h, 2160h); retention must be explicit",
		))
	}

	age, err := time.ParseDuration(*maxAge)
	if err != nil || age <= 0 {
		fatal(errorfamily.NewRejection(
			"lens.purge.bad_max_age",
			fmt.Sprintf("-max-age %q must be a positive Go duration (e.g. 720h, 2160h)", *maxAge),
		))
	}

	ctx, cancel := context.WithTimeout(context.Background(), ingestTimeout)
	defer cancel()

	sys, cleanup, err := cli.OpenSystem(ctx, *db)
	if err != nil {
		fatal(err)
	}

	defer cleanup()

	policy := retention.Policy{MaxAge: age}

	if *dryRun {
		views, err := retention.OldViews(ctx, sys, policy, time.Now())
		if err != nil {
			fatal(err)
		}

		fmt.Printf("purge (dry run): %d call(s) older than %s\n", len(views), *maxAge)

		for _, view := range views {
			fmt.Printf("  %s:%s started %s\n", view.Kind, view.NativeID, view.StartedAt.UTC().Format(time.RFC3339))
		}

		return
	}

	report, err := retention.Retire(ctx, sys, policy, time.Now())
	if err != nil {
		fatal(err)
	}

	if err := cli.DrainProjections(ctx, sys); err != nil {
		fatal(err)
	}

	if err := cli.RebuildSearchIndex(ctx, sys, *db); err != nil {
		fatal(err)
	}

	fmt.Printf("purge: retired %d of %d over-age call(s)\n", report.Retired, report.OverAge)
	printRejected(report.Rejections)
}
