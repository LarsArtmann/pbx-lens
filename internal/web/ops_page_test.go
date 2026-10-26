package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/pbx-lens/internal/calls"
	"github.com/larsartmann/pbx-lens/internal/cli"
)

// opsCounters exercises the ETag middleware once per outcome (tag, 304) so
// the cache section has real numbers to show.
func opsCounters(t *testing.T) *CacheCounters {
	t.Helper()

	stats := NewCacheCounters()

	generate := ETagMiddleware(stats, nil)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("stable body"))
	}))

	first := httptest.NewRecorder()
	generate.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/", nil))

	replay := httptest.NewRecorder()
	conditional := httptest.NewRequest(http.MethodGet, "/", nil)
	conditional.Header.Set("If-None-Match", first.Header().Get("ETag"))
	generate.ServeHTTP(replay, conditional)

	if replay.Code != http.StatusNotModified {
		t.Fatalf("conditional replay status = %d, want 304", replay.Code)
	}

	return stats
}

func opsPollerPayload() func() any {
	age := 90.0
	stale := false

	return func() any {
		return map[string]any{
			"sources": map[string]any{
				"webhook-log": map[string]any{
					"offered": 2, "recorded": 2, "corrected": 0, "rejected": []string{},
				},
				"agent-transcript": map[string]any{
					"offered": 3, "recorded": 2, "corrected": 1,
					"rejected": []string{"age-cutoff", "parse-error"},
				},
				"freeswitch-cdr": map[string]any{
					"offered": 7, "recorded": 7, "corrected": 0, "rejected": []string{},
				},
				"telnyx-conversation": map[string]any{
					"offered": 4, "recorded": 4, "corrected": 0, "rejected": []string{},
				},
			},
			"last_pass":             "2026-10-12T04:00:00Z",
			"last_pass_age_seconds": age,
			"stale":                 stale,
		}
	}
}

func TestOpsPageRendersCounters(t *testing.T) {
	// The human surface shows the same counters /api/ops serves: sources in
	// sorted order (map order would be nondeterministic), the pass verdict,
	// and the cache section — including the rejection reasons.
	mux := newTestServerWith(t, Options{Ops: opsPollerPayload(), Cache: opsCounters(t)})

	status, body := getBody(t, mux, "/ops")
	if status != http.StatusOK {
		t.Fatalf("GET /ops status = %d, want 200 (body: %s)", status, body)
	}

	for _, want := range []string{
		"Ingest poller",
		"Response cache (ETag)",
		"agent-transcript", "freeswitch", "telnyx", "webhook-log",
		"healthy",
		"1m30s ago",
		"Rejected: age-cutoff · parse-error",
		"Hit ratio",
		"50.0%",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("/ops missing %q (body: %s)", want, body)
		}
	}

	// Sorted sources: agent-transcript must precede webhook-log.
	if before := strings.Index(body, "agent-transcript"); before == -1 || before > strings.Index(body, "webhook-log") {
		t.Error("/ops sources must render in sorted order")
	}

	// Rejection count rides the table cell.
	if !strings.Contains(body, ">"+strconv.Itoa(2)+"</td>") {
		t.Errorf("/ops rejection cell missing count 2 (body: %s)", body)
	}
}

func TestOpsPageStaleVerdictAlerts(t *testing.T) {
	// A stale verdict is the page's whole point: the alert names the age
	// and the fix, the badge turns error-colored.
	payload := opsPollerPayload()

	mux := newTestServerWith(t, Options{Ops: payload})

	status, body := getBody(t, mux, "/ops")

	if status != http.StatusOK || strings.Contains(body, "Poller pass is stale") {
		t.Fatalf("healthy payload must not alert (status %d)", status)
	}

	age := 3 * 15 * 60.0

	payload = func() any {
		p := opsPollerPayload()()
		p.(map[string]any)["last_pass_age_seconds"] = age

		p.(map[string]any)["stale"] = true

		return p
	}

	mux = newTestServerWith(t, Options{Ops: payload})

	status, body = getBody(t, mux, "/ops")
	if status != http.StatusOK {
		t.Fatalf("GET /ops status = %d, want 200", status)
	}

	for _, want := range []string{"Poller pass is stale", "stale", "check the serve logs"} {
		if !strings.Contains(body, want) {
			t.Errorf("/ops stale page missing %q (body: %s)", want, body)
		}
	}
}

func TestOpsPageUnknownVerdictNeverLies(t *testing.T) {
	// An older payload shape without the verdict keys must render
	// "unknown", not a fake healthy.
	mux := newTestServerWith(t, Options{Ops: func() any {
		return map[string]any{
			"sources":   map[string]any{},
			"last_pass": "2026-10-12T04:00:00Z",
		}
	}})

	status, body := getBody(t, mux, "/ops")
	if status != http.StatusOK {
		t.Fatalf("GET /ops status = %d, want 200 (body: %s)", status, body)
	}

	if got := strings.Count(body, "unknown"); got < 2 {
		t.Errorf("/ops should render unknown verdict and age (got %d occurrences, body: %s)", got, body)
	}

	if strings.Contains(body, "Poller pass is stale") {
		t.Error("unknown verdict must not render the stale alert")
	}
}

func TestOpsPageNeverRanPassSaysNever(t *testing.T) {
	// A poller that has never completed a pass carries no last_pass at all:
	// the page must say "never", not render the zero time as "Jan 1, 0001"
	// (templ-components#52 has no empty escape).
	mux := newTestServerWith(t, Options{Ops: func() any {
		return map[string]any{
			"sources": map[string]any{
				"webhook-log": map[string]any{
					"offered": 1, "recorded": 1, "corrected": 0, "rejected": []string{},
				},
			},
		}
	}})

	status, body := getBody(t, mux, "/ops")
	if status != http.StatusOK {
		t.Fatalf("GET /ops status = %d, want 200 (body: %s)", status, body)
	}

	if !strings.Contains(body, "never") {
		t.Errorf("/ops never-ran page missing \"never\" (body: %s)", body)
	}

	for _, noise := range []string{"0001", "Jan 1"} {
		if strings.Contains(body, noise) {
			t.Errorf("/ops never-ran page renders zero-time noise %q (body: %s)", noise, body)
		}
	}
}

func TestOpsPageNothingWiredExplains(t *testing.T) {
	// Without any counters the page stays a 200 with the honest empty
	// state (the JSON twin 503s with the same guidance).
	status, body := getBody(t, newTestServer(t), "/ops")
	if status != http.StatusOK {
		t.Fatalf("GET /ops status = %d, want 200 (body: %s)", status, body)
	}

	for _, want := range []string{"No counters are wired", "running poller", "/api/ops"} {
		if !strings.Contains(body, want) {
			t.Errorf("/ops empty state missing %q (body: %s)", want, body)
		}
	}
}

func TestOpsPageCacheOnlyRendersCacheSection(t *testing.T) {
	// The cache middleware runs in every composition — the page must stand
	// on it alone, like the JSON endpoint does.
	mux := newTestServerWith(t, Options{Cache: opsCounters(t)})

	status, body := getBody(t, mux, "/ops")
	if status != http.StatusOK {
		t.Fatalf("GET /ops status = %d, want 200 (body: %s)", status, body)
	}

	if !strings.Contains(body, "Response cache (ETag)") {
		t.Error("/ops cache-only page missing the cache card")
	}

	if strings.Contains(body, "Ingest poller") {
		t.Error("/ops cache-only page must not render a poller card")
	}
}

func TestOpsPageBodyIsDeterministic(t *testing.T) {
	// The ETag/no-cache contract dies on any per-render entropy (the nav-id
	// lesson): two renders of the same counters must be byte-identical.
	mux := newTestServerWith(t, Options{Ops: opsPollerPayload(), Cache: opsCounters(t)})

	_, first := getBody(t, mux, "/ops")
	_, second := getBody(t, mux, "/ops")

	if first != second {
		t.Error("/ops body differs between identical renders — ETag revalidation would never 304")
	}
}

func TestOpsPageNavLinkAndRelativeTimeScrub(t *testing.T) {
	// The nav carries the Ops link on every page (the ops page is the
	// surface's entry), and the RelativeTime rendering is the only
	// wall-clock-driven text (the goldens scrub it).
	mux := newTestServerWith(t, Options{Ops: opsPollerPayload()})

	_, body := getBody(t, mux, "/ops")

	if !strings.Contains(body, `href="/ops"`) {
		t.Error("/ops nav must link to itself so the active state renders")
	}

	if !strings.Contains(body, `datetime="2026-10-12T04:00:00Z"`) {
		t.Error("/ops last-pass RelativeTime must carry the exact datetime attribute")
	}
}

func TestOpsPageRetentionOffSaysSo(t *testing.T) {
	// Retention must never be silent: with no -max-age the card states the
	// keep-everything posture instead of omitting the facts.
	mux := newTestServerWith(t, Options{Ops: opsPollerPayload()})

	status, body := getBody(t, mux, "/ops")
	if status != http.StatusOK {
		t.Fatalf("GET /ops status = %d, want 200", status)
	}

	if !strings.Contains(body, "keeping everything") {
		t.Error("/ops must state the retention-off posture explicitly")
	}

	if !strings.Contains(body, "no rows indexed yet") {
		t.Error("retention-off must still answer the oldest-view fact (here: an empty index)")
	}

	if strings.Contains(body, "Purge lag") {
		t.Error("retention-off must not render a purge-lag row (nothing is ever due)")
	}

	if strings.Contains(body, "Past the window now") {
		t.Error("retention-off must not render the live over-age row")
	}
}

func TestOpsPageRetentionCountsOverAge(t *testing.T) {
	// With a window the card shows the live would-retire count over the
	// seeded index: the seed's oldest call sits at 2026-10-12 09:00, so a
	// 24h window ending "now" (frozen clock) puts both seeds past it.
	seeds := goldenSeed()
	freeze := time.Date(2026, 10, 14, 9, 0, 0, 0, time.UTC)

	mux := newTestServerWith(t, Options{
		Search:          sqliteSearcher(t, seeds...),
		Now:             func() time.Time { return freeze },
		Ops:             opsPollerPayload(),
		RetentionMaxAge: 24 * time.Hour,
	}, seeds...)

	status, body := getBody(t, mux, "/ops")
	if status != http.StatusOK {
		t.Fatalf("GET /ops status = %d, want 200 (body: %s)", status, body)
	}

	for _, want := range []string{
		"Past the window now", "2 call(s) past the retention window", "call.retired",
		// P16 remainder: the oldest unretired view (the seed's oldest call
		// is exactly 48h old against the frozen clock) and the purge lag it
		// implies over the 24h window.
		"Oldest view", "48h old", "Purge lag", "24h past the window",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("/ops retention card missing %q (body: %s)", want, body)
		}
	}
}

func TestOpsPageAndAPICarryRetention(t *testing.T) {
	// P16: the age-posture SLO facts ride BOTH surfaces from the same
	// scans — the page card and the /api/ops "retention" section cannot
	// disagree. The frozen clock sits the seeds (oldest 2026-10-12 09:00)
	// exactly 48h back, so the 24h window puts both past it: over_age 2,
	// oldest age 172800s, purge lag 86400s.
	seeds := goldenSeed()
	freeze := time.Date(2026, 10, 14, 9, 0, 0, 0, time.UTC)

	mux := newTestServerWith(t, Options{
		Ops:             opsPollerPayload(),
		Now:             func() time.Time { return freeze },
		RetentionMaxAge: 24 * time.Hour,
	}, seeds...)

	_, page := getBody(t, mux, "/ops")
	for _, want := range []string{"Oldest view", "48h old", "Purge lag", "24h past the window"} {
		if !strings.Contains(page, want) {
			t.Errorf("/ops retention card missing %q", want)
		}
	}

	status, body := getBody(t, mux, "/api/ops")
	if status != http.StatusOK {
		t.Fatalf("GET /api/ops status = %d, want 200 (body: %s)", status, body)
	}

	for _, want := range []string{
		`"max_age_seconds":86400`, `"active":true`, `"over_age":2`,
		`"oldest_age_seconds":172800`, `"purge_lag_seconds":86400`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("/api/ops retention section missing %q (body: %s)", want, body)
		}
	}
}

func TestIngestCoverageQueryFoldsPerKind(t *testing.T) {
	const staleBoundary = 48 * time.Hour

	now := time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)

	seed := []calls.Observation{
		webObservation("conv-a", now.Add(-2*time.Hour)),
		webObservation("conv-b", now.Add(-staleBoundary)),
		func() calls.Observation {
			obs := webObservation("cdr-1", now.Add(-30*24*time.Hour))
			obs.Kind = calls.SourceFreeSwitchCDR

			return obs
		}(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	sys, cleanup, err := cli.OpenSystem(ctx, filepath.Join(t.TempDir(), "coverage.db"))
	if err != nil {
		t.Fatalf("OpenSystem: %v", err)
	}

	t.Cleanup(cleanup)

	if stats := cli.Ingest(ctx, sys, seed); len(stats.Rejected) > 0 {
		t.Fatalf("ingest rejected: %v", stats.Rejected)
	}

	if err := cli.DrainProjections(ctx, sys); err != nil {
		t.Fatalf("drain: %v", err)
	}

	rows, err := IngestCoverageQuery(ctx, sys, now)
	if err != nil {
		t.Fatalf("IngestCoverageQuery: %v", err)
	}

	if len(rows) != 2 {
		t.Fatalf("coverage rows = %d, want 2 kinds", len(rows))
	}

	byKind := map[calls.SourceKind]CoverageRow{}
	for _, row := range rows {
		byKind[row.Kind] = row
	}

	telnyx := byKind[calls.SourceTelnyxConversation]
	oldest := now.Add(-staleBoundary)
	newest := now.Add(-2 * time.Hour)

	if telnyx.Views != 2 || !telnyx.Newest.Equal(newest) || !telnyx.Oldest.Equal(oldest) {
		t.Errorf("telnyx coverage = %+v, want 2 views over [%v, %v]", telnyx, oldest, newest)
	}

	if telnyx.Stale(now) {
		t.Error("a 2h-old newest row is inside the SLO and must not flag")
	}

	cdr := byKind[calls.SourceFreeSwitchCDR]
	if cdr.Views != 1 || !cdr.Stale(now) {
		t.Errorf("cdr coverage = %+v, want 1 view flagged stale past the SLO", cdr)
	}
}

func TestOpsPageAndAPICarryCoverage(t *testing.T) {
	observation := webObservation("conv-a", time.Now().Add(-time.Hour))

	mux := newTestServerWith(t,
		Options{Ops: opsPollerPayload()},
		observation,
	)

	_, page := getBody(t, mux, "/ops")
	for _, want := range []string{"Index coverage", "telnyx-conversation", "Rows indexed"} {
		if !strings.Contains(page, want) {
			t.Errorf("ops page missing coverage element %q", want)
		}
	}

	status, body := getBody(t, mux, "/api/ops")
	if status != http.StatusOK {
		t.Fatalf("GET /api/ops status = %d, want 200", status)
	}

	for _, want := range []string{
		`"coverage":[`,
		`"kind":"telnyx-conversation"`,
		`"views":1`,
		`"stale":false`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("/api/ops missing coverage key %q", want)
		}
	}
}
