package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/larsartmann/pbx-lens/internal/backup"
	"github.com/larsartmann/pbx-lens/internal/calls"
	"github.com/larsartmann/pbx-lens/internal/cli"
	"github.com/larsartmann/pbx-lens/internal/retention"
)

const (
	smokeTimeout = 30 * time.Second

	projectionDeadline     = 5 * time.Second
	projectionPollInterval = 20 * time.Millisecond

	smokeDurationSecs = 26
)

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
