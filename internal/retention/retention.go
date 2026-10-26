// Package retention enforces the index age policy (T23/D2): a MaxAge
// window bounds how long a call stays indexed, OldViews lists what violates
// it, and Retire tombstones each violation through the domain (call.retired)
// so the removal replays consistently instead of being read-model surgery.
// The event store stays append-only by design; a tombstone frees every READ
// surface (list, search, export, dashboard), and a from-scratch rebuild
// converges to the same purged state because the tombstone is last in its
// stream. Erasing the underlying events entirely remains the disposable-DB
// move: drop the file and re-poll with the policy-gated poller.
package retention

import (
	"context"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
	"github.com/larsartmann/pbx-lens/internal/calls"
)

// ReasonRetention is the audit reason stamped onto every age-policy
// tombstone; the string is stable so exports of the event log group them.
const ReasonRetention = calls.RetentionReason("retention:age")

// ReasonAgeCutoff is the audit label for an observation refused at ingest
// because it predates the policy window (the serve poller's -max-age gate).
// Distinct from ReasonRetention so the purge (tombstone) and ingest-gate
// rejections group separately in event-log exports and poll summaries.
const ReasonAgeCutoff = calls.RetentionReason("retention:age-cutoff")

// maxScan bounds one policy pass. Personal-PBX volumes sit far below this;
// the cap only stops a runaway scan from exhausting memory (same doctrine
// as the export cap).
const maxScan = 100_000

// Policy is the retention window. The zero value keeps everything: retention
// must be opted into explicitly, never silent.
type Policy struct {
	MaxAge time.Duration
}

// Cutoff is the oldest StartedAt the policy keeps, or the zero time when
// the policy is unbounded.
func (p Policy) Cutoff(now time.Time) time.Time {
	if p.MaxAge <= 0 {
		return time.Time{}
	}

	return now.UTC().Add(-p.MaxAge)
}

// Admits reports whether the policy lets ingest keep an observation taken at
// now. The boundary is inclusive: a call observed exactly at the cutoff
// stays, so "max-age 720h" means "keep the last 720 hours", not one fewer.
func (p Policy) Admits(obs calls.Observation, now time.Time) bool {
	cutoff := p.Cutoff(now)
	if cutoff.IsZero() {
		return true
	}

	return !obs.StartedAt.UTC().Before(cutoff)
}

// Report summarizes one policy pass. OverAge counts the views the policy
// wants gone; Retired counts accepted tombstones (the two differ only when
// a dispatch is rejected).
type Report struct {
	OverAge    int
	Retired    int
	Rejections []string // per-stream dispatch failures (stream id: reason)
}

// OldViews returns the indexed calls the policy would retire, oldest first.
func OldViews(ctx context.Context, sys *system.System, policy Policy, now time.Time) ([]calls.CallView, error) {
	cutoff := policy.Cutoff(now)
	if cutoff.IsZero() {
		return nil, nil
	}

	reader := metaengine.NewReader[calls.CallView](sys.MetaEngine(), calls.Collection)

	views, err := reader.Scan(ctx,
		metaengine.WithSort("started_at", true),
		metaengine.WithLimit(maxScan),
	)
	if err != nil {
		return nil, err
	}

	overAge := make([]calls.CallView, 0)

	for _, view := range views {
		if view.StartedAt.UTC().Before(cutoff) {
			overAge = append(overAge, view)
		}
	}

	return overAge, nil
}

// OldestView returns the oldest view still indexed (the age-posture SLO's
// "oldest unretired view"): ok is false for an empty index, not for an
// error. The one-row scan is what the ops surfaces read their purge-lag
// figure from — how far that oldest view sits past the policy window is
// the time-dimension answer the over-age COUNT cannot give.
func OldestView(ctx context.Context, sys *system.System) (calls.CallView, bool, error) {
	reader := metaengine.NewReader[calls.CallView](sys.MetaEngine(), calls.Collection)

	views, err := reader.Scan(ctx,
		metaengine.WithSort("started_at", false),
		metaengine.WithLimit(1),
	)
	if err != nil || len(views) == 0 {
		return calls.CallView{}, false, err
	}

	return views[0], true, nil
}

// Retire tombstones every view OldViews would return. It dispatches the
// domain commands only; the caller drains projections and rebuilds the
// search index afterwards (one CLI orchestration, one place per concern).
func Retire(ctx context.Context, sys *system.System, policy Policy, now time.Time) (Report, error) {
	report := Report{}

	views, err := OldViews(ctx, sys, policy, now)
	if err != nil {
		return report, err
	}

	for _, view := range views {
		report.OverAge++

		cmd, err := calls.NewRetireCommand(view.Kind, view.NativeID, ReasonRetention)
		if err != nil {
			report.Rejections = append(report.Rejections, streamLabel(view)+": "+err.Error())

			continue
		}

		if err := sys.CommandDispatcher().Dispatch(ctx, cmd); err != nil {
			report.Rejections = append(report.Rejections, streamLabel(view)+": "+err.Error())

			continue
		}

		report.Retired++
	}

	return report, nil
}

func streamLabel(view calls.CallView) string {
	return string(view.Kind) + ":" + string(view.NativeID)
}
