package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/larsartmann/pbx-lens/internal/calls"
	"github.com/larsartmann/pbx-lens/internal/cli"
	"github.com/larsartmann/pbx-lens/internal/ingest/cdr"
	"github.com/larsartmann/pbx-lens/internal/ingest/telnyx"
)

const (
	version = "0.1.0"

	minArgs = 2
	exitUsage = 2

	ingestTimeout = 5 * time.Minute
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
	default:
		usage()
		os.Exit(exitUsage)
	}
}

func usage() {
	fmt.Fprintln(
		os.Stderr,
		"usage: pbx-lens version | pbx-lens smoke [-db path] | pbx-lens ingest-cdr -db <dsn> <Master.csv> |"+
			" pbx-lens ingest-telnyx -db <dsn> [-key-file path] [-since 48h] [-direction outbound] [-limit N]",
	)
}

func ingestCDR(args []string) {
	fs := flag.NewFlagSet("ingest-cdr", flag.ExitOnError)
	db := fs.String("db", "", "sqlite DSN (required)")
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

	parsed, err := cdr.Parse(file)
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
	stats.Rejected = append(fetchRejects, stats.Rejected...)
	stats.Total += len(fetchRejects)

	fmt.Printf(
		"ingest-telnyx: %d conversations fetched, %d recorded, %d corrected, %d rejected\n",
		stats.Total, stats.Recorded, stats.Corrected, len(stats.Rejected),
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

func printRejected(rejected []string) {
	for _, reason := range rejected {
		fmt.Println("  rejected:", reason)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "pbx-lens:", err)
	os.Exit(1)
}
