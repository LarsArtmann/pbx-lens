package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/pbx-lens/internal/app"
	"github.com/larsartmann/pbx-lens/internal/calls"
	"github.com/larsartmann/pbx-lens/internal/ingest/cdr"
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
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(
		os.Stderr,
		"usage: pbx-lens version | pbx-lens smoke [-db path] | pbx-lens ingest-cdr -db <dsn> <Master.csv>",
	)
}

func ingestCDR(args []string) {
	fs := flag.NewFlagSet("ingest-cdr", flag.ExitOnError)
	db := fs.String("db", "", "sqlite DSN (required)")
	_ = fs.Parse(args)

	if *db == "" {
		fatal(errors.New("ingest-cdr needs -db <dsn>"))
	}

	if fs.NArg() != 1 {
		fatal(errors.New("ingest-cdr needs exactly one Master.csv path"))
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
			fatal(errors.New("projection did not converge within 5s"))
		}

		time.Sleep(20 * time.Millisecond)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "pbx-lens:", err)
	os.Exit(1)
}
