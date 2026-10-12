package web

import (
	"context"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/pbx-lens/internal/calls"
	"github.com/larsartmann/pbx-lens/internal/cli"
)

func TestDashboardRendersStatsAndCharts(t *testing.T) {
	day := time.Date(2026, 10, 9, 10, 0, 0, 0, time.UTC) // inside the 30-day window
	ai := webObservation("conv-ai", day)
	ai.IsAI = true
	human := webObservation("conv-human", day.Add(time.Hour))
	human.IsAI = false

	mux := newTestServer(t, ai, human)

	status, body := getBody(t, mux, "/dashboard")
	if status != 200 {
		t.Fatalf("GET /dashboard status = %d, want 200 (body: %s)", status, body)
	}

	for _, want := range []string{
		"Dashboard", "Calls indexed", "Calls today", "AI share", "Avg duration",
		"AI vs human", "30-day trend", "Call quality (MOS)", "Not available yet",
		"50.0%",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("dashboard missing %q", want)
		}
	}
}

func TestDashboardCountsMatchFixture(t *testing.T) {
	day := time.Date(2026, 10, 9, 10, 0, 0, 0, time.UTC)
	ai := webObservation("conv-ai", day)
	ai.IsAI = true
	human := webObservation("conv-human", day.Add(time.Hour))
	human.IsAI = false

	older := webObservation("conv-old", day.AddDate(0, 0, -40)) // outside the window, inside totals

	mux := newTestServer(t, ai, human, older)
	_, body := getBody(t, mux, "/dashboard")

	// Totals cover the whole index (3), not just the chart window (2
	// in-window); the donut legend carries the exact AI/human split and the
	// center label the share.
	for _, want := range []string{"AI · 1", "Human · 2", "33.3%"} {
		if !strings.Contains(body, want) {
			t.Errorf("dashboard missing %q", want)
		}
	}

	if strings.Contains(body, "conv-") {
		t.Error("dashboard must not leak native ids into the page")
	}
}

func TestDashboardEmptyIndexIsHonest(t *testing.T) {
	status, body := getBody(t, newTestServer(t), "/dashboard")
	if status != 200 {
		t.Fatalf("GET /dashboard status = %d, want 200", status)
	}

	if !strings.Contains(body, "—") {
		t.Error("AI share must render an em dash on an empty index, not 0.0%")
	}

	if !strings.Contains(body, "No calls indexed yet") {
		t.Error("AI-vs-human card must explain the empty index")
	}
}

func TestDashboardMOSSummaryAndTrend(t *testing.T) {
	day := time.Date(2026, 10, 9, 10, 0, 0, 0, time.UTC)
	scoredA := webObservation("conv-mos-a", day)
	scoredA.MOS = 4.2
	scoredB := webObservation("conv-mos-b", day.Add(time.Hour))
	scoredB.MOS = 3.8
	unscored := webObservation("conv-mos-none", day.Add(2*time.Hour))

	status, body := getBody(t, newTestServer(t, scoredA, scoredB, unscored), "/dashboard")
	if status != 200 {
		t.Fatalf("GET /dashboard status = %d, want 200 (body: %s)", status, body)
	}

	for _, want := range []string{"Call quality (MOS)", "4.0", "mean of 2 scored calls"} {
		if !strings.Contains(body, want) {
			t.Errorf("dashboard missing %q", want)
		}
	}

	if strings.Contains(body, "Not available yet") {
		t.Error("scored calls must replace the MOS placeholder")
	}
}

func TestDashboardMOSOutsideWindowStillCounts(t *testing.T) {
	old := webObservation("conv-mos-old", time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC))
	old.MOS = 3.5

	_, body := getBody(t, newTestServer(t, old), "/dashboard")

	for _, want := range []string{"3.5", "mean of 1 scored call", "No MOS data in the last 30 days"} {
		if !strings.Contains(body, want) {
			t.Errorf("dashboard missing %q", want)
		}
	}
}

func TestDashboardStatsQueryMOSAggregation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	sys, cleanup, err := cli.OpenSystem(ctx, filepath.Join(t.TempDir(), "dashboard-mos-test.db"))
	if err != nil {
		t.Fatalf("OpenSystem: %v", err)
	}

	t.Cleanup(cleanup)

	day := time.Date(2026, 10, 9, 8, 0, 0, 0, time.UTC)
	scoredA := webObservation("conv-mos-1", day)
	scoredA.MOS = 4.0
	scoredB := webObservation("conv-mos-2", day.Add(time.Hour))
	scoredB.MOS = 2.0
	unscored := webObservation("conv-mos-3", day.Add(2*time.Hour))

	if stats := cli.Ingest(ctx, sys, []calls.Observation{scoredA, scoredB, unscored}); len(stats.Rejected) > 0 {
		t.Fatalf("ingest rejected: %v", stats.Rejected)
	}

	if err := cli.DrainProjections(ctx, sys); err != nil {
		t.Fatalf("drain: %v", err)
	}

	stats, err := DashboardStatsQuery(
		t.Context(),
		sys,
		time.Date(2026, 10, 11, 15, 0, 0, 0, time.UTC),
		ListCallsQuery{},
		dashboardWindowDays,
	)
	if err != nil {
		t.Fatalf("DashboardStatsQuery: %v", err)
	}

	if stats.MOSKnown != 2 {
		t.Errorf("MOSKnown = %d, want 2", stats.MOSKnown)
	}

	if stats.MOSSum != 6.0 {
		t.Errorf("MOSSum = %v, want 6", stats.MOSSum)
	}

	if got := stats.AvgMOS(); got != 3.0 {
		t.Errorf("AvgMOS = %v, want 3", got)
	}

	for _, dayVolume := range stats.ByDay {
		switch dayVolume.Day {
		case "2026-10-09":
			if dayVolume.MOSKnown != 2 || dayVolume.MOSSum != 6.0 || dayVolume.AvgMOS() != 3.0 {
				t.Errorf(
					"2026-10-09 MOS = %d/%v/%v, want 2/6/3",
					dayVolume.MOSKnown,
					dayVolume.MOSSum,
					dayVolume.AvgMOS(),
				)
			}
		default:
			if dayVolume.MOSKnown != 0 || dayVolume.AvgMOS() != 0 {
				t.Errorf("%s MOS = %d/%v, want 0/0", dayVolume.Day, dayVolume.MOSKnown, dayVolume.AvgMOS())
			}
		}
	}
}

func TestDashboardStatsQueryWindowMath(t *testing.T) {
	// Query-level check: in-window days aggregate, out-of-window calls only
	// feed the totals, and the window always spans exactly 30 day buckets.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	sys, cleanup, err := cli.OpenSystem(ctx, filepath.Join(t.TempDir(), "dashboard-test.db"))
	if err != nil {
		t.Fatalf("OpenSystem: %v", err)
	}

	t.Cleanup(cleanup)

	aiCall := webObservation("conv-a", time.Date(2026, 10, 9, 8, 0, 0, 0, time.UTC))
	aiCall.IsAI = true
	aiCallB := webObservation("conv-b", time.Date(2026, 10, 9, 9, 0, 0, 0, time.UTC))
	aiCallB.IsAI = true
	aiCallC := webObservation("conv-c", time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC))
	aiCallC.IsAI = true
	humanCall := webObservation("conv-old", time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC))

	if stats := cli.Ingest(
		ctx,
		sys,
		[]calls.Observation{aiCall, aiCallB, aiCallC, humanCall},
	); len(
		stats.Rejected,
	) > 0 {
		t.Fatalf("ingest rejected: %v", stats.Rejected)
	}

	if err := cli.DrainProjections(ctx, sys); err != nil {
		t.Fatalf("drain: %v", err)
	}

	stats, err := DashboardStatsQuery(
		t.Context(),
		sys,
		time.Date(2026, 10, 11, 15, 0, 0, 0, time.UTC),
		ListCallsQuery{},
		dashboardWindowDays,
	)
	if err != nil {
		t.Fatalf("DashboardStatsQuery: %v", err)
	}

	if stats.Total != 4 {
		t.Errorf("Total = %d, want 4", stats.Total)
	}

	if stats.Today != 0 {
		t.Errorf("Today = %d, want 0 (fixture days are not 2026-10-11)", stats.Today)
	}

	if len(stats.ByDay) != 30 {
		t.Fatalf("len(ByDay) = %d, want 30", len(stats.ByDay))
	}

	// conv-a + conv-b land on 10-09, conv-c on 09-20; conv-old is outside.
	for _, day := range stats.ByDay {
		switch day.Day {
		case "2026-10-09":
			if day.Calls != 2 {
				t.Errorf("2026-10-09 calls = %d, want 2", day.Calls)
			}
		case "2026-09-20":
			if day.Calls != 1 {
				t.Errorf("2026-09-20 calls = %d, want 1", day.Calls)
			}
		default:
			if day.Calls != 0 {
				t.Errorf("%s calls = %d, want 0", day.Day, day.Calls)
			}
		}
	}

	if stats.AICalls != 3 || stats.HumanCalls != 1 {
		t.Errorf("AI/Human = %d/%d, want 3/1", stats.AICalls, stats.HumanCalls)
	}
}

func TestDashboardBerlinDayBoundary(t *testing.T) {
	// A call at 22:30 UTC is 00:30 CEST: the dashboard counts it on the
	// operator's Berlin day (R2), so "Calls today" on the 10th includes it.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	sys, cleanup, err := cli.OpenSystem(ctx, filepath.Join(t.TempDir(), "dashboard-berlin-test.db"))
	if err != nil {
		t.Fatalf("OpenSystem: %v", err)
	}

	t.Cleanup(cleanup)

	nightCall := webObservation("conv-night", time.Date(2026, 10, 9, 22, 30, 0, 0, time.UTC))
	if stats := cli.Ingest(ctx, sys, []calls.Observation{nightCall}); len(stats.Rejected) > 0 {
		t.Fatalf("ingest rejected: %v", stats.Rejected)
	}

	if err := cli.DrainProjections(ctx, sys); err != nil {
		t.Fatalf("drain: %v", err)
	}

	stats, err := DashboardStatsQuery(
		t.Context(),
		sys,
		time.Date(2026, 10, 10, 10, 0, 0, 0, time.UTC),
		ListCallsQuery{},
		dashboardWindowDays,
	)
	if err != nil {
		t.Fatalf("DashboardStatsQuery: %v", err)
	}

	if stats.Today != 1 {
		t.Errorf("Today = %d, want 1 (the call crossed Berlin midnight into the 10th)", stats.Today)
	}

	if got := stats.ByDay[len(stats.ByDay)-1]; got.Day != "2026-10-10" || got.Calls != 1 {
		t.Errorf("newest bucket = %s/%d calls, want 2026-10-10/1", got.Day, got.Calls)
	}
}

func TestDashboardStatsQueryZoneIsInjectable(t *testing.T) {
	// UTC coverage (m2.2): pointing the zone at UTC puts the same call back
	// on the 9th — the zone decides the bucket, not the data.
	saved := dashboardLocation
	dashboardLocation = time.UTC

	t.Cleanup(func() { dashboardLocation = saved })

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	sys, cleanup, err := cli.OpenSystem(ctx, filepath.Join(t.TempDir(), "dashboard-utc-test.db"))
	if err != nil {
		t.Fatalf("OpenSystem: %v", err)
	}

	t.Cleanup(cleanup)

	nightCall := webObservation("conv-night", time.Date(2026, 10, 9, 22, 30, 0, 0, time.UTC))
	if stats := cli.Ingest(ctx, sys, []calls.Observation{nightCall}); len(stats.Rejected) > 0 {
		t.Fatalf("ingest rejected: %v", stats.Rejected)
	}

	if err := cli.DrainProjections(ctx, sys); err != nil {
		t.Fatalf("drain: %v", err)
	}

	stats, err := DashboardStatsQuery(
		t.Context(),
		sys,
		time.Date(2026, 10, 10, 10, 0, 0, 0, time.UTC),
		ListCallsQuery{},
		dashboardWindowDays,
	)
	if err != nil {
		t.Fatalf("DashboardStatsQuery: %v", err)
	}

	if stats.Today != 0 {
		t.Errorf("Today = %d, want 0 under UTC (the call stayed on the 9th)", stats.Today)
	}

	found := false

	for _, day := range stats.ByDay {
		if day.Day == "2026-10-09" {
			found = day.Calls == 1
		}
	}

	if !found {
		t.Error("UTC bucketing must land the call on 2026-10-09 with 1 call")
	}
}

func TestDashboardDurationHistogramAndWeekDelta(t *testing.T) {
	// M20.2 + R27: durations land in exactly one bucket, and a full previous
	// week yields a signed week-over-week delta on the volume card.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	sys, cleanup, err := cli.OpenSystem(ctx, filepath.Join(t.TempDir(), "dashboard-hist-test.db"))
	if err != nil {
		t.Fatalf("OpenSystem: %v", err)
	}

	t.Cleanup(cleanup)

	now := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)

	short := webObservation("conv-short", now)
	short.DurationSecs = 10
	medium := webObservation("conv-medium", now.Add(-time.Hour))
	medium.DurationSecs = 45
	long := webObservation("conv-long", now.Add(-2*time.Hour))
	long.DurationSecs = 200

	lastWeek := webObservation("conv-lastweek", now.AddDate(0, 0, -10))
	lastWeek.DurationSecs = 60

	if stats := cli.Ingest(ctx, sys, []calls.Observation{short, medium, long, lastWeek}); len(stats.Rejected) > 0 {
		t.Fatalf("ingest rejected: %v", stats.Rejected)
	}

	if err := cli.DrainProjections(ctx, sys); err != nil {
		t.Fatalf("drain: %v", err)
	}

	stats, err := DashboardStatsQuery(t.Context(), sys, now, ListCallsQuery{}, dashboardWindowDays)
	if err != nil {
		t.Fatalf("DashboardStatsQuery: %v", err)
	}

	wantCalls := map[string]int{"< 30s": 1, "30-60s": 2, "1-5m": 1}
	total := 0

	for _, bucket := range stats.Durations {
		total += bucket.Calls

		if want, ok := wantCalls[bucket.Label]; ok && bucket.Calls != want {
			t.Errorf("bucket %s = %d calls, want %d", bucket.Label, bucket.Calls, want)
		}
	}

	if total != 4 {
		t.Errorf("histogram must place every call exactly once, got %d placements for 4 calls", total)
	}

	if !stats.Week.HasBaseline {
		t.Fatal("a call 10 days back must give the window a previous-week baseline")
	}

	if stats.Week.Pct <= 0 {
		t.Errorf("3 this week vs 1 last week = +%v%%, got %v", stats.Week.Pct, stats.Week.Pct)
	}
}

func TestDashboardPageCarriesPollingAndHistogram(t *testing.T) {
	// R9/M20.2 page-level: the live region polls the fragment URL, the
	// duration card renders, and the sparkline carries its aria label (R27).
	mux := newTestServer(t, webObservation("conv-a", time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)))

	status, body := getBody(t, mux, "/dashboard")
	if status != 200 {
		t.Fatalf("GET /dashboard status = %d, want 200", status)
	}

	for _, want := range []string{
		`hx-get="/frag/dashboard"`,
		`hx-trigger="every 30s"`,
		`src="/static/htmx.min.js"`,
		"Call durations",
		"30-60s",
		"aria-label=",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("dashboard missing %q", want)
		}
	}
}

func TestDashboardFragmentRendersRegionOnly(t *testing.T) {
	// The polled payload must be the region alone: no shell, no doctype —
	// otherwise the outerHTML swap would nest whole documents.
	mux := newTestServer(t, webObservation("conv-a", time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)))

	status, body := getBody(t, mux, "/frag/dashboard")
	if status != 200 {
		t.Fatalf("GET /frag/dashboard status = %d, want 200", status)
	}

	if !strings.Contains(body, "Calls indexed") {
		t.Error("fragment must carry the stat cards")
	}

	if strings.Contains(strings.ToLower(body), "<!doctype") || strings.Contains(body, "<nav") {
		t.Error("fragment must not carry the page shell")
	}
}

func TestDashboardRangePickerFiltersAndCarriesRange(t *testing.T) {
	day := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	inRange := webObservation("conv-in", day)
	outOfRange := webObservation("conv-out", day.AddDate(0, 0, -20))

	mux := newTestServer(t, inRange, outOfRange)

	status, body := getBody(t, mux, "/dashboard?from=2026-10-08&to=2026-10-10")
	if status != 200 {
		t.Fatalf("GET /dashboard?range status = %d, want 200 (body: %s)", status, body)
	}

	if !strings.Contains(body, `hx-get="/frag/dashboard?from=2026-10-08&amp;to=2026-10-10"`) &&
		!strings.Contains(body, `hx-get="/frag/dashboard?from=2026-10-08&to=2026-10-10"`) {
		t.Errorf("polled region must carry the picker range (body: %s)", body[:400])
	}

	// The picker value echoes back into the form.
	if !strings.Contains(body, `value="2026-10-08"`) {
		t.Error("from input should echo the active range")
	}

	// Garbage and non-date-bounds reject with 400 (house style).
	for _, bad := range []string{"from=nope", "from=2026-10-01T00:00:00Z"} {
		status, _ = getBody(t, mux, "/dashboard?"+bad)
		if status != http.StatusBadRequest {
			t.Errorf("GET /dashboard?%s status = %d, want 400", bad, status)
		}
	}
}

func TestStalenessFragmentRechecksBanner(t *testing.T) {
	// The polled staleness region renders the banner server-side, so the
	// noscript fallback and the polled markup are the same contract (m9.4).
	old := webObservation("conv-old", time.Date(2026, 10, 9, 19, 51, 25, 0, time.UTC))
	mux := newTestServer(t, old)

	status, body := getBody(t, mux, "/frag/staleness")
	if status != 200 {
		t.Fatalf("GET /frag/staleness status = %d, want 200", status)
	}

	if !strings.Contains(body, "Index may be stale") {
		t.Error("freshness fragment must render the stale banner")
	}

	if !strings.Contains(body, `hx-get="/frag/staleness"`) {
		t.Error("fragment must keep the poll alive across swaps")
	}
}

func TestHtmxAssetServed(t *testing.T) {
	// The self-hosted runtime must exist in the build (R9): a missing asset
	// silently kills every polled region.
	status, body := getBody(t, newTestServer(t), "/static/htmx.min.js")
	if status != 200 {
		t.Fatalf("GET /static/htmx.min.js status = %d, want 200", status)
	}

	if !strings.Contains(body, "htmx") {
		t.Error("htmx asset should carry the runtime")
	}
}
