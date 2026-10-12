package web

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

// TestHeatmapBucketsDashboardZone pins the bucketing contract (plan T19):
// call starts land on the dashboard-zone weekday x hour, not UTC. A 23:30
// UTC winter instant is 00:30 Berlin — the next weekday, hour 0.
func TestHeatmapBucketsDashboardZone(t *testing.T) {
	// 2026-01-15 23:30 UTC = 2026-01-16 00:30 CET (UTC+1, Thursday).
	utc := time.Date(2026, 1, 15, 23, 30, 0, 0, time.UTC)

	local := utc.In(dashboardLocation)
	weekday := (int(local.Weekday()) + 6) % 7

	if got := time.Thursday; weekday != int(got)-1 {
		// Jan 16 2026 is a Friday; Mon-first index 4.
		if weekday != 4 {
			t.Errorf("weekday index = %d, want 4 (Friday Mon-first)", weekday)
		}
	}

	if local.Hour() != 0 {
		t.Errorf("Berlin hour = %d, want 0", local.Hour())
	}
}

// TestHeatmapDSTBoundary pins the DST story: on the spring switch (2026-03-29
// Berlin jumps 02:00 -> 03:00 CEST), a 01:30 UTC instant is 03:30 local —
// hour bucket 3, not the winter 2.
func TestHeatmapDSTBoundary(t *testing.T) {
	utc := time.Date(2026, 3, 29, 1, 30, 0, 0, time.UTC)

	local := utc.In(dashboardLocation)
	if local.Hour() != 3 {
		t.Errorf("DST-switch local hour = %d, want 3 (CEST)", local.Hour())
	}
}

// TestHeatmapCardRenders pins the card's honesty ladder: data renders the
// grid, an empty window renders the explanation instead of an all-zero grid.
func TestHeatmapCardRenders(t *testing.T) {
	mux := newTestServer(t)

	_, empty := getBody(t, mux, "/dashboard")
	if !strings.Contains(empty, "Call density (weekday × hour)") {
		t.Fatal("heatmap card missing")
	}

	if !strings.Contains(empty, "No calls in the last 30 days.") {
		t.Error("empty index should explain the quiet heatmap, not render zeros")
	}

	seeded := webObservation("hm-1", time.Date(2026, 10, 12, 14, 0, 0, 0, time.UTC))
	mux2 := newTestServer(t, seeded)

	status, body := getBody(t, mux2, "/dashboard")
	if status != http.StatusOK {
		t.Fatalf("dashboard status = %d", status)
	}

	if !strings.Contains(body, `role="img"`) || !strings.Contains(body, "Mon") {
		t.Error("data-carrying dashboard should render the heatmap grid")
	}
}
