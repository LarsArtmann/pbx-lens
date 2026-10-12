package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
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
	jsonOut := fs.Bool("json", false, "emit the report as JSON (works for both -dry-run and a real purge)")
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

	ctx, sys, cleanup, err := openIndexSession(*db)
	if err != nil {
		fatal(err)
	}

	defer cleanup()

	if err := runPurge(
		ctx,
		sys,
		*db,
		retention.Policy{MaxAge: age},
		*dryRun,
		*jsonOut,
		*maxAge,
		os.Stdout,
	); err != nil {
		fatal(err)
	}
}

// runPurge executes the retire-or-list pass and renders the report, text
// or JSON (-json, m27.8). Split from the flag wiring so the drill is
// testable without an os.Exit minefield.
func runPurge(
	ctx context.Context,
	sys *system.System,
	dsn string,
	policy retention.Policy,
	dryRun, asJSON bool,
	maxAgeLabel string,
	out io.Writer,
) error {
	if dryRun {
		views, err := retention.OldViews(ctx, sys, policy, time.Now())
		if err != nil {
			return err
		}

		if asJSON {
			candidates := make([]map[string]string, 0, len(views))
			for _, view := range views {
				candidates = append(candidates, map[string]string{
					"kind":       string(view.Kind),
					"native_id":  string(view.NativeID),
					"started_at": view.StartedAt.UTC().Format(time.RFC3339),
				})
			}

			return json.NewEncoder(out).Encode(map[string]any{
				"dry_run":      true,
				"max_age":      maxAgeLabel,
				"over_age":     len(views),
				"would_retire": candidates,
			})
		}

		return writeDryRunList(out, views, maxAgeLabel)
	}

	report, err := retention.Retire(ctx, sys, policy, time.Now())
	if err != nil {
		return err
	}

	if err := cli.DrainProjections(ctx, sys); err != nil {
		return err
	}

	if err := cli.RebuildSearchIndex(ctx, sys, dsn); err != nil {
		return err
	}

	if asJSON {
		return json.NewEncoder(out).Encode(map[string]any{
			"dry_run":    false,
			"max_age":    maxAgeLabel,
			"retired":    report.Retired,
			"over_age":   report.OverAge,
			"rejections": report.Rejections,
		})
	}

	if _, err := fmt.Fprintf(
		out,
		"purge: retired %d of %d over-age call(s)\n",
		report.Retired,
		report.OverAge,
	); err != nil {
		return err
	}

	printRejected(report.Rejections)

	return nil
}

// writeDryRunList renders the human dry-run report: a header line plus one
// line per over-age call.
func writeDryRunList(out io.Writer, views []calls.CallView, maxAgeLabel string) error {
	if _, err := fmt.Fprintf(
		out,
		"purge (dry run): %d call(s) older than %s\n",
		len(views),
		maxAgeLabel,
	); err != nil {
		return err
	}

	for _, view := range views {
		if _, err := fmt.Fprintf(
			out,
			"  %s:%s started %s\n",
			view.Kind,
			view.NativeID,
			view.StartedAt.UTC().Format(time.RFC3339),
		); err != nil {
			return err
		}
	}

	return nil
}
