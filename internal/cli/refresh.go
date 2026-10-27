package cli

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/pbx-lens/internal/calls"
)

// RefreshOutcome classifies one fetched observation against the stored read
// model. It exists so an operator can see — before writing anything — how many
// streams a ContentHash schema bump (or a source-side change) would rewrite.
type RefreshOutcome string

const (
	// RefreshNew means the index has no row for the stream yet.
	RefreshNew RefreshOutcome = "new"
	// RefreshStale means the source's content hash differs from the stored
	// row's, so a re-ingest will emit call.corrected (one event per stream).
	RefreshStale RefreshOutcome = "stale"
	// RefreshAdditiveOnly is the silent-drift alarm: the hashes match, but the
	// source carries additive data the stored row lacks. A re-ingest would
	// replay it as a no-op (the decider answers an equal hash with nil), so
	// this row would never heal without a schema bump.
	RefreshAdditiveOnly RefreshOutcome = "additive-only"
	// RefreshUnchanged means the stored row already reflects the source.
	RefreshUnchanged RefreshOutcome = "unchanged"
	// RefreshUnbuildable means the observation cannot address a stream (a
	// mapper bug or a source row without identity); it is reported, never
	// dispatched.
	RefreshUnbuildable RefreshOutcome = "unbuildable"
)

// RefreshFinding is one stream's classification, with the reason a human
// needs to judge whether the rewrite is expected.
type RefreshFinding struct {
	Stream  string
	Outcome RefreshOutcome
	Reason  string
}

// RefreshReport is one classification pass over a source batch.
type RefreshReport struct {
	Total    int
	Findings []RefreshFinding
}

// Count reports how many findings carry an outcome.
func (r RefreshReport) Count(outcome RefreshOutcome) int {
	count := 0

	for _, finding := range r.Findings {
		if finding.Outcome == outcome {
			count++
		}
	}

	return count
}

// Rewrites reports how many streams a re-ingest would actually write: new
// rows plus corrections. Unchanged and unbuildable streams cost nothing.
func (r RefreshReport) Rewrites() int {
	return r.Count(RefreshNew) + r.Count(RefreshStale)
}

func (r RefreshReport) String() string {
	return fmt.Sprintf(
		"%d streams: %d new, %d stale, %d additive-only, %d unchanged, %d unbuildable (a re-ingest would write %d)",
		r.Total,
		r.Count(RefreshNew),
		r.Count(RefreshStale),
		r.Count(RefreshAdditiveOnly),
		r.Count(RefreshUnchanged),
		r.Count(RefreshUnbuildable),
		r.Rewrites(),
	)
}

// RefreshReadError reports a failed read-model lookup while classifying one
// stream: the measurement itself failed, so the report is incomplete. The
// per-stream outcomes are data in RefreshReport, not errors; this type carries
// only the read failure, with the stream named for the operator.
type RefreshReadError struct {
	Stream string
	Err    error
}

func (e *RefreshReadError) Error() string {
	return fmt.Sprintf("refresh: read view for %s: %v", e.Stream, e.Err)
}

func (e *RefreshReadError) Unwrap() error { return e.Err }

func newRefreshReadError(stream string, err error) *RefreshReadError {
	return &RefreshReadError{Stream: stream, Err: err}
}

// ClassifyRefresh compares each fetched observation with its stored view
// without dispatching anything. This is the measurement half of the
// additive-field doctrine (see calls.ObservationSchemaVersion): a schema bump
// turns every stream stale, and an operator can size that wave before the
// first write.
//
// Lookups go through calls.LookupView, so a row written under the retired
// marker-prefixed key is found instead of being misreported as new.
func ClassifyRefresh(
	ctx context.Context,
	reader *metaengine.TypedReader[calls.CallView],
	observations []calls.Observation,
) (RefreshReport, error) {
	report := RefreshReport{Total: len(observations), Findings: make([]RefreshFinding, 0, len(observations))}

	for _, obs := range observations {
		stream := streamLabel(obs)

		sid, err := obs.StreamID()
		if err != nil {
			report.Findings = append(report.Findings, RefreshFinding{
				Stream:  stream,
				Outcome: RefreshUnbuildable,
				Reason:  err.Error(),
			})

			continue
		}

		view, found, err := calls.LookupView(ctx, reader, sid)
		if err != nil {
			return report, newRefreshReadError(stream, err)
		}

		if !found {
			report.Findings = append(report.Findings, RefreshFinding{
				Stream:  stream,
				Outcome: RefreshNew,
				Reason:  "no stored row for this stream",
			})

			continue
		}

		if view.ContentHash != obs.ContentHash() {
			report.Findings = append(report.Findings, RefreshFinding{
				Stream:  stream,
				Outcome: RefreshStale,
				Reason:  "content hash differs from the stored row",
			})

			continue
		}

		if reason := additiveDrift(obs, view); reason != "" {
			report.Findings = append(report.Findings, RefreshFinding{
				Stream:  stream,
				Outcome: RefreshAdditiveOnly,
				Reason:  reason,
			})

			continue
		}

		report.Findings = append(report.Findings, RefreshFinding{
			Stream:  stream,
			Outcome: RefreshUnchanged,
			Reason:  "stored row matches the source",
		})
	}

	return report, nil
}

// additiveDrift names the additive (deliberately unhashed) data the source
// carries but the stored view does not. It is the one place that knows which
// observation fields live outside ContentHash: a future additive field must be
// added here too, or the refresh diagnostic under-reports the drift it exists
// to catch (the schema-version bump is the repair; this is the detector).
func additiveDrift(obs calls.Observation, view calls.CallView) string {
	if len(obs.SourceEvents) > len(view.SourceEvents) {
		return fmt.Sprintf(
			"source carries %d source event(s), stored row has %d — unhashed, so a re-ingest is a no-op",
			len(obs.SourceEvents),
			len(view.SourceEvents),
		)
	}

	return ""
}
