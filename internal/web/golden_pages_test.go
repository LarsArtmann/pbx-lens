package web

import (
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/pbx-lens/internal/calls"
	"github.com/larsartmann/templ-components/utils/golden"
)

// Golden-HTML snapshots for the operator-facing pages (plan T22): copy
// drift and markup regressions fail `go test`, and an intentional change
// regenerates with `go test -update ./internal/web/ -run TestGoldenPages`.
// Wall-clock renderings (relative times, the poll-region footer, the
// staleness age) are scrubbed to placeholders before the compare — the
// goldens pin STRUCTURE, not the minute.

var goldenVolatile = []*regexp.Regexp{
	// PolledRegion footer: now-based datetime attribute + rendered time.
	regexp.MustCompile(`datetime="[^"]*"( tabindex="-1")?>Updated `),
	regexp.MustCompile(`Updated [0-9:]+`),
	// RelativeTime renderings (seeded times, humanized against time.Now).
	regexp.MustCompile(`\bjust now\b`),
	regexp.MustCompile(`\b\d+ (?:second|minute|hour|day|week)s? ago\b`),
	// Staleness banner age ("Newest indexed call is 2h5m old").
	regexp.MustCompile(`call is \S+( \S+)? old`),
}

var goldenReplacements = []string{
	`datetime="SCRUBBED">Updated `,
	"Updated HH:MM:SS",
	"RELATIVE_TIME",
	"RELATIVE_TIME",
	"call is AGE old",
}

func scrubVolatileTime(body string) string {
	for i, re := range goldenVolatile {
		body = re.ReplaceAllString(body, goldenReplacements[i])
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
	sample := `Updated 14:03:09 datetime="2026-10-12T14:03:09+02:00" tabindex="-1">Updated ` +
		`just now 5 minutes ago 2 hours ago call is 2h5m old every 30s &lt; 30s 30-60s`
	out := scrubVolatileTime(sample)

	for _, leaked := range []string{"Updated 14", "just now", "minutes ago", "hours ago", "call is 2h5m old", "2026-10-12T14:03"} {
		if strings.Contains(out, leaked) {
			t.Errorf("scrubber leaked %q -> %q", leaked, out)
		}
	}

	// Structural timings and chart labels must survive untouched.
	for _, kept := range []string{"every 30s", "&lt; 30s", "30-60s"} {
		if !strings.Contains(out, kept) {
			t.Errorf("scrubber ate structural text %q -> %q", kept, out)
		}
	}
}
