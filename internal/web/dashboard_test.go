package web

import (
	"context"
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
		"50.0% AI",
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
	// in-window); the legend carries the exact AI/human split.
	for _, want := range []string{"AI 1 · Human 2", "33.3% AI"} {
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

	stats, err := DashboardStatsQuery(t.Context(), sys, time.Date(2026, 10, 11, 15, 0, 0, 0, time.UTC))
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
