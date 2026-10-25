package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/pbx-lens/internal/calls"
	"github.com/larsartmann/pbx-lens/internal/cli"
	"github.com/larsartmann/pbx-lens/internal/ingest/telnyx"
)

// refreshFindingLimit caps the per-finding detail a refresh pass prints. The
// counts always carry the whole truth; the list is orientation, and a
// thousand-line terminal dump is not.
const refreshFindingLimit = 20

// refresh is the one-shot backfill verb of the additive-field doctrine (see
// calls.ObservationSchemaVersion): fetch the source, classify every stream
// against the stored read model, and — only with -apply — re-ingest what the
// classification says would change.
//
// Unlike ingest-telnyx it defaults to an UNBOUNDED look-back: its whole reason
// to exist is history the 48 h poll window can no longer reach. It also skips
// the enrichment pass, which owns its own hash and its own API budget.
func refresh(args []string) {
	if err := runRefresh(args, os.Stdout); err != nil {
		fatal(err)
	}
}

func runRefresh(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("refresh", flag.ExitOnError)
	db := fs.String("db", "", "sqlite DSN (required)")
	keyFile := fs.String("key-file", "", "Telnyx API key file (default: $TELNYX_API_KEY)")
	since := fs.String("since", "0", "look-back window: Go duration or RFC3339 timestamp (0 = every conversation)")
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
	apiBase := fs.String("api-base", "", "override the Telnyx API base URL (stub rehearsals; empty = production)")
	apply := fs.Bool("apply", false, "dispatch the corrections (default: classify and report only)")
	_ = fs.Parse(args)

	if *db == "" {
		return errDBRequired
	}

	key, err := cli.ResolveKey(*keyFile)
	if err != nil {
		return err
	}

	sinceAt, err := cli.ParseSince(*since)
	if err != nil {
		return err
	}

	dir, err := cli.ParseDirection(*direction)
	if err != nil {
		return err
	}

	ctx, sys, cleanup, err := openIndexSession(*db)
	if err != nil {
		return err
	}

	defer cleanup()

	observations, _, fetchRejects, err := cli.FetchTelnyxObservations(
		ctx,
		telnyx.NewClient(key, *apiBase),
		cli.TelnyxOptions{
			Since:     sinceAt,
			Direction: dir,
			AIDIDs:    telnyx.ParseAIDIDs(*aiDID),
			Limit:     *limit,
		},
	)
	if err != nil {
		return err
	}

	reader := metaengine.NewReader[calls.CallView](sys.MetaEngine(), calls.Collection)

	report, err := cli.ClassifyRefresh(ctx, reader, observations)
	if err != nil {
		return err
	}

	printRefreshReport(out, report, fetchRejects)

	if !*apply {
		_, _ = fmt.Fprintf(
			out,
			"refresh: classify-only — nothing written. %d stream(s) would be written; re-run with -apply to backfill.\n",
			report.Rewrites(),
		)

		return nil
	}

	stats := cli.Ingest(ctx, sys, observations)
	if err := finalizeIngest(ctx, sys, *db); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(
		out,
		"refresh: applied — %d recorded, %d corrected, %d rejected (search index rebuilt)\n",
		stats.Recorded, stats.Corrected, len(stats.Rejected),
	)

	printRejectedTo(out, stats.Rejected)

	return nil
}

// printRefreshReport prints the classification counts first (they are the
// deliverable) and then at most refreshFindingLimit findings that are worth an
// operator's attention — unchanged streams are pure noise.
func printRefreshReport(out io.Writer, report cli.RefreshReport, fetchRejects []string) {
	_, _ = fmt.Fprintf(out, "refresh: %d conversations fetched, %s\n", report.Total, report.String())

	printed := 0

	for _, finding := range report.Findings {
		if finding.Outcome == cli.RefreshUnchanged {
			continue
		}

		if printed == refreshFindingLimit {
			_, _ = fmt.Fprintf(
				out,
				"  … %d further finding(s) not shown\n",
				countActionable(report)-refreshFindingLimit,
			)

			break
		}

		_, _ = fmt.Fprintf(out, "  %s %s — %s\n", finding.Outcome, finding.Stream, finding.Reason)
		printed++
	}

	printRejectedTo(out, fetchRejects)
}

func countActionable(report cli.RefreshReport) int {
	count := 0

	for _, finding := range report.Findings {
		if finding.Outcome != cli.RefreshUnchanged {
			count++
		}
	}

	return count
}

func printRejectedTo(out io.Writer, rejected []string) {
	for _, reason := range rejected {
		_, _ = fmt.Fprintf(out, "  rejected: %s\n", reason)
	}
}
