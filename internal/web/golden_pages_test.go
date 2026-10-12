package web

import (
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/templ-components/utils/golden"
	"github.com/larsartmann/pbx-lens/internal/calls"
)

// Golden-HTML snapshots for the operator-facing pages (plan T22): copy
// drift and markup regressions fail `go test`, and an intentional change
// regenerates with `go test -update ./internal/web/ -run TestGoldenPages`.
// Wall-clock renderings (relative times, the poll-region footer, the
// staleness age) are scrubbed to placeholders before the compare — the
// goldens pin STRUCTURE, not the minute.

var goldenVolatile = []*regexp.Regexp{
	regexp.MustCompile(`Updated [0-9:]+`),
	regexp.MustCompile(`\bjust now\b`),
	regexp.MustCompile(`\b\d+ (?:second|minute|hour|day|week)s? ago\b`),
	regexp.MustCompile(`\b\d+d \d+h\b`),
	regexp.MustCompile(`\b\d+h \d+m\b`),
	regexp.MustCompile(`\b\d+m\b`),
	regexp.MustCompile(`\b\d+s\b`),
}

func scrubVolatileTime(body string) string {
	replacements := []string{"Updated HH:MM:SS", "RELATIVE_TIME", "RELATIVE_TIME", "AGE", "AGE", "AGE", "AGE"}

	for i, re := range goldenVolatile {
		body = re.ReplaceAllString(body, replacements[i])
	}

	return body
}

func goldenPage(t *testing.T, mux *http.ServeMux, name, path string) {
	t.Helper()

	status, body := getBody(t, mux, path)
	if status != http.StatusOK {
		t.Fatalf("GET %s status = %d, want 200", path, status)
	}

	golden.Assert(t, name, scrubVolatileTime(body))
}

func goldenSeed() []calls.Observation {
	base := time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC)

	inbound := webObservation("conv-in", base.Add(time.Minute))
	inbound.Direction = calls.DirectionInbound
	inbound.Transcript = []calls.TranscriptTurn{
		{Role: "user", Text: "hello there"},
		{Role: "assistant", Text: "how can I help"},
	}

	outbound := webObservation("conv-out", base)

	return []calls.Observation{outbound, inbound}
}

func TestGoldenPages(t *testing.T) {
	mux := newTestServerWithSearch(t, goldenSeed()...)

	goldenPage(t, mux, "page_calls", "/")
	goldenPage(t, mux, "page_calls_filtered_empty", "/?direction=inbound")
	goldenPage(t, mux, "page_search", "/search?q=hello")
	goldenPage(t, mux, "page_dashboard", "/dashboard")
	goldenPage(t, mux, "page_detail", "/calls/telnyx-conversation/conv-in")
}

// The scrubber itself stays honest: it must neutralize every wall-clock
// string the seeded pages can render, or the goldens flake by the minute.
func TestGoldenScrubberCoversVolatileStrings(t *testing.T) {
	sample := "Updated 14:03:09 just now 5 minutes ago 2 hours ago 3d 4h 2h 5m 43m 9s"
	out := scrubVolatileTime(sample)

	for _, leaked := range []string{"Updated 14", "just now", "minutes ago", "hours ago", "3d 4h", "2h 5m", "43m", "9s"} {
		if strings.Contains(out, leaked) {
			t.Errorf("scrubber leaked %q -> %q", leaked, out)
		}
	}
}
