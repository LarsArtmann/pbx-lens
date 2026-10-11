package retention_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/larsartmann/pbx-lens/internal/calls"
	"github.com/larsartmann/pbx-lens/internal/cli"
	"github.com/larsartmann/pbx-lens/internal/retention"
	"github.com/larsartmann/pbx-lens/internal/web"
)

func TestPolicyAdmitsBoundaryInclusive(t *testing.T) {
	now := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
	policy := retention.Policy{MaxAge: 720 * time.Hour}

	if got := policy.Cutoff(now); !got.Equal(now.Add(-720 * time.Hour)) {
		t.Errorf("Cutoff = %v, want %v", got, now.Add(-720*time.Hour))
	}

	unbounded := retention.Policy{}
	if !unbounded.Cutoff(now).IsZero() {
		t.Error("unbounded policy must have a zero cutoff")
	}

	cases := []struct {
		name   string
		age    time.Duration
		accept bool
	}{
		{"fresh", 24 * time.Hour, true},
		{"exactly at cutoff", 720 * time.Hour, true},
		{"one second over", 720*time.Hour + time.Second, false},
		{"a year old", 365 * 24 * time.Hour, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			obs := calls.Observation{StartedAt: now.Add(-tc.age)}

			if got := policy.Admits(obs, now); got != tc.accept {
				t.Errorf("Admits(age=%s) = %t, want %t", tc.age, got, tc.accept)
			}
		})
	}
}

func TestRetireTombstonesOnlyOverAgeViews(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	sys, cleanup, err := cli.OpenSystem(ctx, filepath.Join(t.TempDir(), "retention-test.db"))
	if err != nil {
		t.Fatalf("OpenSystem: %v", err)
	}

	t.Cleanup(cleanup)

	now := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
	old1 := observation("old-1", now.Add(-30*24*time.Hour))
	old2 := observation("old-2", now.Add(-20*24*time.Hour))
	fresh := observation("fresh", now.Add(-2*time.Hour))

	if stats := cli.Ingest(ctx, sys, []calls.Observation{old1, old2, fresh}); len(stats.Rejected) > 0 {
		t.Fatalf("ingest rejected: %v", stats.Rejected)
	}

	if err := cli.DrainProjections(ctx, sys); err != nil {
		t.Fatalf("drain after ingest: %v", err)
	}

	policy := retention.Policy{MaxAge: 7 * 24 * time.Hour}

	report, err := retention.Retire(ctx, sys, policy, now)
	if err != nil {
		t.Fatalf("Retire: %v", err)
	}

	if report.OverAge != 2 || report.Retired != 2 || len(report.Rejections) > 0 {
		t.Fatalf("report = %+v, want over-age 2, retired 2, no rejections", report)
	}

	if err := cli.DrainProjections(ctx, sys); err != nil {
		t.Fatalf("drain after retire: %v", err)
	}

	views, err := web.ExportCalls(ctx, sys, web.ListCallsQuery{})
	if err != nil {
		t.Fatalf("ExportCalls: %v", err)
	}

	if len(views) != 1 || views[0].NativeID != "fresh" {
		t.Fatalf("remaining views = %+v, want only fresh", views)
	}

	rerun, err := retention.Retire(ctx, sys, policy, now)
	if err != nil {
		t.Fatalf("re-run Retire: %v", err)
	}

	if rerun.OverAge != 0 || rerun.Retired != 0 {
		t.Errorf("re-run report = %+v, want a converged no-op", rerun)
	}

	// The poller will re-offer the purged rows on every pass; the domain
	// must answer with a no-op replay (same ContentHash), never resurrect
	// the view.
	if stats := cli.Ingest(ctx, sys, []calls.Observation{old1, old2}); len(stats.Rejected) > 0 {
		t.Fatalf("re-ingest rejected: %v", stats.Rejected)
	}

	if err := cli.DrainProjections(ctx, sys); err != nil {
		t.Fatalf("drain after re-ingest: %v", err)
	}

	views, err = web.ExportCalls(ctx, sys, web.ListCallsQuery{})
	if err != nil {
		t.Fatalf("ExportCalls after re-ingest: %v", err)
	}

	if len(views) != 1 || views[0].NativeID != "fresh" {
		t.Fatalf("views after re-ingest = %+v, want purged calls to stay purged", views)
	}
}

func TestUnboundedPolicyRetiresNothing(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	sys, cleanup, err := cli.OpenSystem(ctx, filepath.Join(t.TempDir(), "retention-unbounded.db"))
	if err != nil {
		t.Fatalf("OpenSystem: %v", err)
	}

	t.Cleanup(cleanup)

	now := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
	ancient := observation("ancient", now.Add(-3*365*24*time.Hour))

	if stats := cli.Ingest(ctx, sys, []calls.Observation{ancient}); len(stats.Rejected) > 0 {
		t.Fatalf("ingest rejected: %v", stats.Rejected)
	}

	if err := cli.DrainProjections(ctx, sys); err != nil {
		t.Fatalf("drain: %v", err)
	}

	report, err := retention.Retire(ctx, sys, retention.Policy{}, now)
	if err != nil {
		t.Fatalf("Retire: %v", err)
	}

	if report.OverAge != 0 || report.Retired != 0 {
		t.Errorf("report = %+v, want the unbounded policy to touch nothing", report)
	}
}

func observation(nativeID string, started time.Time) calls.Observation {
	return calls.Observation{
		NativeID:     calls.NativeID(nativeID),
		Kind:         calls.SourceTelnyxConversation,
		Direction:    calls.DirectionOutbound,
		StartedAt:    started,
		DurationSecs: 42,
		FromNumber:   "+17287289311",
		ToNumber:     "+15555550123",
	}
}
