package web

import (
	"net/http"
	"strings"
	"testing"

	"github.com/larsartmann/templ-components/display"
)

// TestDashboardWindowPresets pins plan T12: ?window=7|30 changes the chart
// span honestly (title + poll URL), garbage rejects with 400, and both
// presets render with the active one visible.
func TestDashboardWindowPresets(t *testing.T) {
	mux := newTestServer(t)

	status, body := getBody(t, mux, "/dashboard")
	if status != http.StatusOK {
		t.Fatalf("dashboard status = %d, want 200", status)
	}

	if !strings.Contains(body, "Calls per day (last 30 days)") {
		t.Error("default window should be 30 days")
	}

	for _, preset := range []string{"Last 7 days", "Last 30 days"} {
		if !strings.Contains(body, preset) {
			t.Errorf("preset %q missing", preset)
		}
	}

	status, body = getBody(t, mux, "/dashboard?window=7")
	if status != http.StatusOK {
		t.Fatalf("window=7 status = %d, want 200", status)
	}

	if !strings.Contains(body, "Calls per day (last 7 days)") {
		t.Error("window=7 should retitle the volume chart")
	}

	if !strings.Contains(body, `hx-get="/frag/dashboard?window=7"`) {
		t.Errorf("polled region must carry window=7, got: %s", pollURLOf(body))
	}

	status, _ = getBody(t, mux, "/dashboard?window=9")
	if status != http.StatusBadRequest {
		t.Errorf("window=9 status = %d, want 400", status)
	}

	// The polled fragment honors the same window.
	status, frag := getBody(t, mux, "/frag/dashboard?window=7")
	if status != http.StatusOK {
		t.Fatalf("frag window=7 status = %d, want 200", status)
	}

	if !strings.Contains(frag, "last 7 days") {
		t.Error("fragment should re-render the 7-day chart")
	}
}

// TestSearchHintProse pins the digits-only affordance (plan T12, trimmed
// 2026-10-15): the search page advertises the phone-number shortcut as one
// prose hint under the form — no floating chip beside it.
func TestSearchHintProse(t *testing.T) {
	mux := newTestServer(t)

	_, body := getBody(t, mux, "/search")
	if !strings.Contains(body, "Matches transcript words and phone numbers") {
		t.Error("search page missing the phone-number shortcut hint")
	}

	if strings.Contains(body, "digits-only finds numbers") {
		t.Error("search page still renders the redundant hint chip")
	}
}

// TestWeekChangeRendering pins the StatCard Change/Trend contract (plan
// T13): the delta renders library-native with an explicit sign and the
// trend arrow follows the percentage, not the baseline flag alone.
func TestWeekChangeRendering(t *testing.T) {
	if got := weekChangeLabel(WeekDelta{Pct: 12.4, HasBaseline: true}); got != "+12% vs last week" {
		t.Errorf("weekChangeLabel(up) = %q", got)
	}

	if got := weekChangeLabel(WeekDelta{Pct: -3.6, HasBaseline: true}); got != "-4% vs last week" {
		t.Errorf("weekChangeLabel(down) = %q", got)
	}

	cases := []struct {
		delta WeekDelta
		want  display.TrendDirection
	}{
		{WeekDelta{Pct: 5, HasBaseline: true}, display.TrendUp},
		{WeekDelta{Pct: -5, HasBaseline: true}, display.TrendDown},
		{WeekDelta{Pct: 0, HasBaseline: true}, display.TrendNone},
		{WeekDelta{Pct: 5, HasBaseline: false}, display.TrendNone},
	}

	for _, tc := range cases {
		if got := weekTrend(tc.delta); got != tc.want {
			t.Errorf("weekTrend(%+v) = %q, want %q", tc.delta, got, tc.want)
		}
	}
}
