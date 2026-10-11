package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	appkit "github.com/larsartmann/go-appkit"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/larsartmann/pbx-lens/internal/calls"
	"github.com/larsartmann/pbx-lens/internal/cli"
	"github.com/larsartmann/pbx-lens/internal/ingest/cdr"
	"github.com/larsartmann/pbx-lens/internal/ingest/telnyx"
	"github.com/larsartmann/pbx-lens/internal/ingest/webhook"
	"github.com/larsartmann/pbx-lens/internal/web"
)

const (
	version = "0.1.0"

	minArgs   = 2
	exitUsage = 2

defaultAddr = "127.0.0.1:8300"
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
			" pbx-lens serve -db <dsn> [-addr 127.0.0.1:8300]",
	)
}

func ingestCDR(args []string) {
	fs := flag.NewFlagSet("ingest-cdr", flag.ExitOnError)
	db := fs.String("db", "", "sqlite DSN (required)")
	aiDest := fs.String("ai-dest", cdr.DefaultAIDestination,
		"comma-separated AI destinations (dialplan badge): destination_number matches badge IsAI")
	_ = fs.Parse(args)

	if *db == "" {
		fatal(errorfamily.NewRejection("lens.cli.missing_db", "ingest-cdr needs -db <dsn>"))
	}

	if fs.NArg() != 1 {
		fatal(errorfamily.NewRejection("lens.cli.bad_args", "ingest-cdr needs exactly one Master.csv path"))
	}

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

	if *db == "" {
		fatal(errorfamily.NewRejection("lens.cli.missing_db", "ingest-telnyx needs -db <dsn>"))
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

	ctx, cancel := context.WithTimeout(context.Background(), ingestTimeout)
	defer cancel()

	sys, cleanup, err := cli.OpenSystem(ctx, *db)
	if err != nil {
		fatal(err)
	}

	defer cleanup()

	client := telnyx.NewClient(key, "")

	observations, fetchRejects, err := cli.FetchTelnyxObservations(ctx, client, cli.TelnyxOptions{
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

	stats.Rejected = append(fetchRejects, stats.Rejected...)
	stats.Total += len(fetchRejects)

	fmt.Printf(
		"ingest-telnyx: %d conversations fetched, %d recorded, %d corrected, %d rejected\n",
		stats.Total, stats.Recorded, stats.Corrected, len(stats.Rejected),
	)

	printRejected(stats.Rejected)
}

func ingestWebhook(args []string) {
	fs := flag.NewFlagSet("ingest-webhook", flag.ExitOnError)
	db := fs.String("db", "", "sqlite DSN (required)")
	_ = fs.Parse(args)

	if *db == "" {
		fatal(errorfamily.NewRejection("lens.cli.missing_db", "ingest-webhook needs -db <dsn>"))
	}

	if fs.NArg() != 1 {
		fatal(errorfamily.NewRejection("lens.cli.bad_args", "ingest-webhook needs exactly one inbound.jsonl path"))
	}

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

	fmt.Printf(
		"ingest-webhook: %d calls aggregated, %d lines skipped, %d recorded, %d corrected, %d rejected\n",
			stats.Total, parsed.Skipped, stats.Recorded, stats.Corrected, len(stats.Rejected),
	)

	printRejected(stats.Rejected)
}

func ingestAgent(args []string) {
	fs := flag.NewFlagSet("ingest-agent", flag.ExitOnError)
	db := fs.String("db", "", "sqlite DSN (required)")
	dir := fs.String("dir", "/var/lib/telephony/recordings/transcripts", "transcripts directory ({call-uuid}.jsonl files)")
	_ = fs.Parse(args)

	if *db == "" {
		fatal(errorfamily.NewRejection("lens.cli.missing_db", "ingest-agent needs -db <dsn>"))
	}

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
	_ = fs.Parse(args)

	if *db == "" {
		fatal(errorfamily.NewRejection("lens.cli.missing_db", "serve needs -db <dsn>"))
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

	svc, err := appkit.NewService(appkit.ServiceConfig{Addr: *addr})
	if err != nil {
		fatal(err)
	}

	recordings := web.RecordingOptions{RecordingsDir: *recordingsDir}

	if *keyFile != "" {
		key, err := cli.ResolveKey(*keyFile)
		if err != nil {
			fatal(err)
		}

		recordings.Telnyx = telnyx.NewClient(key, "")
	}

	web.Register(svc.Mux, sys, web.Options{Recordings: recordings})

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
