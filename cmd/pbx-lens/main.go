package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/larsartmann/pbx-lens/internal/app"
	"github.com/larsartmann/pbx-lens/internal/calls"
	"github.com/larsartmann/pbx-lens/internal/ingest/cdr"
	"github.com/larsartmann/pbx-lens/internal/ingest/telnyx"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "version":
		fmt.Println("pbx-lens 0.1.0")
	case "smoke":
		smoke(os.Args[2:])
	case "ingest-cdr":
		ingestCDR(os.Args[2:])
	case "ingest-telnyx":
		ingestTelnyx(os.Args[2:])
	default:
		usage()
		os.Exit(2)
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

	f, err := os.Open(fs.Arg(0))
	if err != nil {
		fatal(err)
	}
	defer f.Close()

	res, err := cdr.Parse(f)
	if err != nil {
		fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	sys, err := app.New(ctx, *db)
	if err != nil {
		fatal(err)
	}
	defer sys.Close()

	hostCtx, stopHost := context.WithCancel(ctx)
	defer stopHost()

	go func() { _ = sys.ProjectionHost().Start(hostCtx) }()

	rejected := 0

	for _, obs := range res.Observations {
		cmd, err := calls.NewRecordCommand(obs)
		if err != nil {
			rejected++

			continue
		}

		if err := sys.CommandDispatcher().Dispatch(ctx, cmd); err != nil {
			rejected++
		}
	}

	fmt.Printf("ingested %d, parse-skipped %d, rejected %d\n", len(res.Observations)-rejected, res.Skipped, rejected)
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

	key, err := resolveKey(*keyFile)
	if err != nil {
		fatal(err)
	}

	sinceAt, err := parseSince(*since)
	if err != nil {
		fatal(err)
	}

	dir := calls.Direction(*direction)
	if !dir.Valid() {
		fatal(
			errorfamily.NewRejection(
				"lens.cli.bad_direction",
				fmt.Sprintf("invalid -direction %q: must be outbound or inbound", *direction),
			),
		)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	sys, err := app.New(ctx, *db)
	if err != nil {
		fatal(err)
	}
	defer sys.Close()

	hostCtx, stopHost := context.WithCancel(ctx)
	defer stopHost()

	go func() { _ = sys.ProjectionHost().Start(hostCtx) }()

	client := telnyx.NewClient(key, "")

	convs, err := client.ListConversations(ctx, telnyx.ListOptions{Since: sinceAt, Limit: *limit})
	if err != nil {
		fatal(err)
	}

	recs, err := client.ListRecordings(ctx, 300)
	if err != nil {
		fatal(err)
	}

	ingested := 0

	var rejected []string

	for _, conv := range convs {
		msgs, err := client.Messages(ctx, conv.ID)
		if err != nil {
			rejected = append(rejected, conv.ID+": "+err.Error())

			continue
		}

		var rec *telnyx.Recording
		if r, ok := telnyx.RecordingByCallControl(recs, conv.MetadataString("call_control_id")); ok {
			rec = &r
		}

		obs, err := telnyx.MapConversation(conv, msgs, rec, telnyx.MapOptions{Direction: dir})
		if err != nil {
			rejected = append(rejected, conv.ID+": "+err.Error())

			continue
		}

		cmd, err := calls.NewRecordCommand(obs)
		if err != nil {
			rejected = append(rejected, conv.ID+": "+err.Error())

			continue
		}

		if err := sys.CommandDispatcher().Dispatch(ctx, cmd); err != nil {
			rejected = append(rejected, conv.ID+": "+err.Error())

			continue
		}

		ingested++
	}

	fmt.Printf("ingest-telnyx: %d conversations, %d ingested, %d rejected\n", len(convs), ingested, len(rejected))

	for _, r := range rejected {
		fmt.Println("  rejected:", r)
	}
}

func resolveKey(keyFile string) (string, error) {
	if keyFile != "" {
		raw, err := os.ReadFile(keyFile)
		if err != nil {
			return "", err
		}

		key := strings.TrimSpace(string(raw))
		if key == "" {
			return "", errorfamily.NewRejection(
				"lens.cli.empty_key_file",
				"key file is empty: "+keyFile,
			)
		}

		return key, nil
	}

	key := strings.TrimSpace(os.Getenv("TELNYX_API_KEY"))
	if key == "" {
		return "", errorfamily.NewRejection("lens.cli.no_api_key", "no API key: pass -key-file or set TELNYX_API_KEY")
	}

	return key, nil
}

func parseSince(s string) (time.Time, error) {
	if ts, err := time.Parse(time.RFC3339, s); err == nil {
		return ts, nil
	}

	d, err := time.ParseDuration(s)
	if err != nil {
		return time.Time{}, errorfamily.NewRejection(
			"lens.cli.bad_since",
			fmt.Sprintf("-since must be a duration (48h) or RFC3339 timestamp, got %q", s),
		)
	}

	return time.Now().UTC().Add(-d), nil
}

func smoke(args []string) {
	fs := flag.NewFlagSet("smoke", flag.ExitOnError)
	db := fs.String("db", "", "sqlite DSN (default: in-memory)")
	_ = fs.Parse(args)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	sys, err := app.New(ctx, *db)
	if err != nil {
		fatal(err)
	}
	defer sys.Close()

	hostCtx, stopHost := context.WithCancel(ctx)
	defer stopHost()

	go func() { _ = sys.ProjectionHost().Start(hostCtx) }()

	obs := calls.Observation{
		NativeID:          "smoke-0001",
		Kind:              calls.SourceTelnyxConversation,
		Direction:         calls.DirectionOutbound,
		StartedAt:         time.Now().UTC().Truncate(time.Second),
		DurationSecs:      26,
		FromNumber:        "+17287289311",
		ToNumber:          "+14084757593",
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
	deadline := time.Now().Add(5 * time.Second)

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

		time.Sleep(20 * time.Millisecond)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "pbx-lens:", err)
	os.Exit(1)
}
