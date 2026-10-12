package web

import (
	"net/http"
	"strings"
	"testing"
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

// TestSearchHintChip pins the digits-only affordance (plan T12): the search
// page advertises the phone-number shortcut as a chip, not just prose.
func TestSearchHintChip(t *testing.T) {
	mux := newTestServer(t)

	_, body := getBody(t, mux, "/search")
	if !strings.Contains(body, "digits-only finds numbers") {
		t.Error("search page missing the digits-only hint chip")
	}

	if !strings.Contains(body, "Matches transcript words and phone numbers") {
		t.Error("search prose hint missing")
	}
}
