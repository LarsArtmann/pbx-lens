package main

import (
	"bytes"
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/larsartmann/pbx-lens/internal/backup"
	"github.com/larsartmann/pbx-lens/internal/calls"
	"github.com/larsartmann/pbx-lens/internal/cli"
	"github.com/larsartmann/pbx-lens/internal/identity"
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
	age := fs.Duration("age", 0, "backdate the seeded call (staleness probes; verify-staleness-live.js uses -age 48h)")
	withRecording := fs.String("with-recording", "", "seed the recording reference (audio-card fixture for detail-rec)")
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
		StartedAt:         time.Now().UTC().Truncate(time.Second).Add(-*age),
		DurationSecs:      smokeDurationSecs,
		FromNumber:        "+17287289311",
		ToNumber:          "+15555550123",
		HangupCause:       "normal_clearing",
		IsAI:              true,
		TranscriptExcerpt: "Hi Lars! This is Woolly, calling from your own PBX.",
		RecordingRef:      *withRecording,
	}

	cmd, err := calls.NewRecordCommand(obs)
	if err != nil {
		fatal(err)
	}

	// The duplicate dispatch exercises the idempotency contract end-to-end:
	// the same stream id and ContentHash must replay without error and
	// converge to one view below.
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
		view, found, err := reader.Get(ctx, calls.RecordKey(cmd.StreamID()))
		if err != nil {
			fatal(err)
		}

		if found {
			// jsontext's encoder newline-terminates and indents, matching the
			// v1 MarshalIndent+Println output this used to print.
			var buf bytes.Buffer

			enc := jsontext.NewEncoder(&buf, jsontext.WithIndent("  "))

			if err := json.MarshalEncode(enc, view); err != nil {
				fatal(err)
			}

			fmt.Print(&buf)

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
	parseAndRequireDB(fs, db, args)

	if *out == "" {
		fatal(errMissingOut("backup"))
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

// identityExportCmd writes the identity DB's users as a plain-JSON record
// (email + profile metadata, no credentials at all). It is a readable
// record of who the identity database holds, NOT a restore path: the
// upstream import creates an account without a passkey and closes
// registration, which locks the operator out (verified 2026-11-21,
// pinned by TestImportedAccountIsLockedOut; the runbook carries the
// details), so every successful run says so.
//
// The identity database is passed as -auth-db — the serve verb's own name
// for it — never as -db: -db means the index everywhere else, and the
// identity opener creates its tables in whichever file it is handed. The
// verb opens an EXISTING identity database only (identity.OpenExisting): a
// missing file or a foreign database is refused before anything is written.
// The WebAuthn flags exist only because opening the identity store
// constructs the ceremony provider; an export never runs a ceremony, so
// the defaults are the dev values.
func identityExportCmd(args []string) {
	fs := flag.NewFlagSet("identity-export", flag.ExitOnError)
	authDB := fs.String("auth-db", "", "identity sqlite DSN — the serve verb's -auth-db file (required)")
	out := fs.String("out", "", "JSON output path (required)")
	rpid := fs.String("auth-rpid", "localhost", "WebAuthn RPID (ceremony-only; an export never runs one)")
	origin := fs.String("auth-origin", "http://localhost", "WebAuthn origin (ceremony-only; an export never runs one)")
	_ = fs.Parse(args)

	requireAuthDB(fs, *authDB)

	if *out == "" {
		fatal(errMissingOut("identity-export"))
	}

	ctx, cancel := context.WithTimeout(context.Background(), ingestTimeout)
	defer cancel()

	users, err := runIdentityExport(ctx, *authDB, *out, *rpid, *origin)
	if err != nil {
		fatal(err)
	}

	fmt.Printf("identity users exported: %d → %s\n", users, *out)

	fmt.Fprintln(os.Stderr,
		"note: this JSON is a readable record, not a restore path — importing it on a fresh host locks "+
			"the operator out; moving the account means restoring the identity.db snapshot "+
			"(docs/runbooks/identity-first-deploy.md)")

	if users == 0 {
		fmt.Fprintln(os.Stderr,
			"warning: the identity database holds no users — if the operator already registered, "+
				"this is the wrong -auth-db file")
	}
}

// runIdentityExport opens the EXISTING identity database (never creates
// one), streams the user array into out, and reports how many users it
// wrote. Opening is the full identity boot — the service owns the read
// model the export reads — so the verb is a short-lived second identity
// instance on the file serve holds open, running the same idempotent
// housekeeping serve runs (TestIdentityExportWhileAnotherInstanceIsLive).
func runIdentityExport(ctx context.Context, dsn, out, rpid, origin string) (int, error) {
	ident, err := identity.OpenIdentity(ctx, identity.IdentityConfig{
		DSN:       dsn,
		RPID:      rpid,
		RPOrigins: []string{origin},
		Policy:    identity.OpenExisting,
	})
	if err != nil {
		return 0, err
	}

	defer func() {
		if cerr := ident.Close(); cerr != nil {
			slog.Warn("identity-export: closing the identity store", "err", cerr)
		}
	}()

	if err := writeExportArtifact(ctx, out, ident.ExportUsersToJSON); err != nil {
		return 0, err
	}

	return ident.OpsSnapshot(ctx).Users, nil
}

// writeExportArtifact streams write's output into out through a private
// staging file beside it: the artifact appears under its name only once it
// is complete and synced, so a failed export leaves neither a truncated
// file nor a hole where the previous artifact was. Unlike the index
// snapshot the artifact is re-derivable (re-run the verb), so the rename's
// directory entry is not fsynced.
func writeExportArtifact(ctx context.Context, out string, write func(context.Context, io.Writer) error) error {
	staging, err := os.CreateTemp(filepath.Dir(out), filepath.Base(out)+".*.tmp")
	if err != nil {
		return errorfamily.WrapRejection(err, "lens.identity_export.out", "staging beside "+out).
			WithContext("out", out).
			WithContext("cause", err.Error())
	}

	staged := staging.Name()

	// artifactFailure names the step that failed and keeps the cause reachable
	// (errors.Is) and renderable (the CLI template reads both keys).
	artifactFailure := func(step string, err error) error {
		return errorfamily.WrapRejection(err, "lens.identity_export.out", step+" "+out).
			WithContext("out", out).
			WithContext("cause", err.Error())
	}

	discard := func() {
		_ = staging.Close()
		_ = os.Remove(staged)
	}

	if err := write(ctx, staging); err != nil {
		discard()

		return err
	}

	if err := staging.Sync(); err != nil {
		discard()

		return artifactFailure("syncing the staged export beside", err)
	}

	if err := staging.Close(); err != nil {
		_ = os.Remove(staged)

		return artifactFailure("writing the staged export for", err)
	}

	if err := os.Rename(staged, out); err != nil {
		_ = os.Remove(staged)

		return artifactFailure("moving the export into place at", err)
	}

	return nil
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
	parseAndRequireDB(fs, db, args)

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

			return json.MarshalEncode(jsontext.NewEncoder(out), map[string]any{
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
		return json.MarshalEncode(jsontext.NewEncoder(out), map[string]any{
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
