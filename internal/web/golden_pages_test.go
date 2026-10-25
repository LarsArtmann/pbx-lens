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
	// Staleness poll-region CONTENT: the banner renders only once the
	// newest indexed call crosses staleAfter, so its presence flips with
	// the wall clock (the page_calls golden silently captured the
	// under-6h state and broke at 17:00 local). The region's attributes
	// stay pinned; the banner itself is covered by the staleness handler
	// tests, not the goldens.
	regexp.MustCompile(`(?s)(<div hx-get="/frag/staleness"[^>]*>).*?(</div>\s*<div class="mb-6">)`),
	// Bare calendar dates: the dashboard's day-bucket labels and
	// today-relative filter links carry whichever date "today" resolved
	// to, so they shift at midnight Europe/Berlin. Only BARE dates scrub
	// — the next character guard spares the fixed-seed rows'
	// datetime="…T…" attributes, which stay pinned.
	regexp.MustCompile(`(20\d{2}-\d{2}-\d{2})([^T0-9-]|$)`),
}

var goldenReplacements = []string{
	`datetime="SCRUBBED">Updated `,
	"Updated HH:MM:SS",
	"RELATIVE_TIME",
	"RELATIVE_TIME",
	"call is AGE old",
	`${1}STALENESS_REGION${2}`,
	`DATE$2`,
}

func scrubVolatileTime(body string) string {
	for i, re := range goldenVolatile {
		body = re.ReplaceAllString(body, goldenReplacements[i])
	}

	return body
}

// TestGoldenScrubberCoversFooterStamp pins the scrubber's contract against
// the PolledRegion footer copy (T18/F71): if the copy drifts (wording or
// attribute order), the volatile patterns silently stop matching and the
// goldens would capture raw wall-clock stamps — flaking on seconds or
// baking the minute into a "structure" snapshot. This test fails instead,
// pointing at the pattern+replacement pair to update alongside the copy.
func TestGoldenScrubberCoversFooterStamp(t *testing.T) {
	body := `<time datetime="2026-10-18T01:23:45+02:00" tabindex="-1">Updated 01:23:45</time>`

	// The optional-tabindex group is CONSUMED by the pattern and the
	// replacement re-emits the element without it — the goldens' established
	// shape (changing that is a golden -update sweep, not a scrub fix).
	want := `<time datetime="SCRUBBED">Updated HH:MM:SS</time>`

	if got := scrubVolatileTime(body); got != want {
		t.Errorf(
			"footer stamp scrub mismatch — a goldenVolatile pattern no longer matches the footer copy; update the pattern AND its goldenReplacements entry together with the copy change\n got %q\nwant %q",
			got,
			want,
		)
	}
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
	seeds := goldenSeed()

	// The dashboard buckets by the wall clock (today-count, day-bucket
	// position, trend peak), so the clock is FROZEN here: the original
	// golden captured the seeds as "today" on 2026-10-12 and flipped at
	// midnight Berlin. The frozen now sits the day AFTER the seeds —
	// deterministic forever, staleness banner predictably visible.
	mux := newTestServerWith(t,
		Options{
			Search: sqliteSearcher(t, seeds...),
			Now:    func() time.Time { return time.Date(2026, 10, 13, 10, 0, 0, 0, time.UTC) },
		},
		seeds...,
	)

	goldenPage(t, mux, "page_calls", "/")
	goldenPage(t, mux, "page_calls_filtered_empty", "/?direction=inbound")
	goldenPage(t, mux, "page_search", "/search?q=hello")
	goldenPage(t, mux, "page_search_transcript", "/search?q=hello&mode=transcript")
	goldenPage(t, mux, "page_dashboard", "/dashboard")
	goldenPage(t, mux, "page_detail", "/calls/telnyx-conversation/conv-in")
	goldenPage(t, mux, "page_detail_q", "/calls/telnyx-conversation/conv-in?q=how&at=1")

	// The ops surface rides its own mux under THIS entry point — a partial
	// -run of the golden suite must not be able to skip it (05-49 §f6).
	goldenOpsPage(t)
}

// page_ops pins the human ops surface (05-40 M9): poller verdict, sorted
// sources, cache section. The RelativeTime "last pass" is scrubbed like
// every other wall-clock string. It lives INSIDE TestGoldenPages (its own
// mux, same entry point) so `-run TestGoldenPages` can never skip the ops
// golden — the exact trap 05-49 §f6 recorded when this was a separate
// top-level test.
func goldenOpsPage(t *testing.T) {
	t.Helper()

	mux := newTestServerWith(t, Options{Ops: opsPollerPayload(), Cache: opsCounters(t)})
	goldenPage(t, mux, "page_ops", "/ops")
}

// The ?q= golden pins the match log's affordances beyond its markup: the
// first matching turn carries its anchor id and the amber highlight, and
// unmatched turns stay unmarked (plan T1.2 — the hand-rolled branch had
// unit tests but no structural snapshot).
func TestGoldenDetailQueryCarriesMatchLog(t *testing.T) {
	mux := newTestServerWithSearch(t, goldenSeed()...)

	status, body := getBody(t, mux, "/calls/telnyx-conversation/conv-in?q=how&at=1")
	if status != http.StatusOK {
		t.Fatalf("detail?q status = %d, want 200", status)
	}

	if !strings.Contains(body, `id="turn-1"`) {
		t.Error("matched turn missing its anchor id in the ?q= rendering")
	}

	if !strings.Contains(body, "bg-amber-100") {
		t.Error("matched turn missing the highlight mark in the ?q= rendering")
	}

	if strings.Contains(body, `id="turn-0"`) {
		t.Error("unmatched turn must not carry an anchor id")
	}
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
