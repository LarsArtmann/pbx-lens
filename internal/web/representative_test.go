package web

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/system/v4"
	"github.com/larsartmann/pbx-lens/internal/calls"
	"github.com/larsartmann/pbx-lens/internal/cli"
)

// newTestSystem opens a fresh sqlite index and ingests the observations,
// drained so the read model is queryable. Query-level tests need the system
// itself; the HTTP tests build their own through newTestServer.
func newTestSystem(t *testing.T, observations ...calls.Observation) *system.System {
	t.Helper()

	ctx := t.Context()

	sys, cleanup, err := cli.OpenSystem(ctx, filepath.Join(t.TempDir(), "representative.db"))
	if err != nil {
		t.Fatalf("OpenSystem: %v", err)
	}

	t.Cleanup(cleanup)

	if len(observations) > 0 {
		if stats := cli.Ingest(ctx, sys, observations); len(stats.Rejected) > 0 {
			t.Fatalf("ingest rejected: %v", stats.Rejected)
		}

		if err := cli.DrainProjections(ctx, sys); err != nil {
			t.Fatalf("drain: %v", err)
		}
	}

	return sys
}

// aiCallFixture is one real call as the index sees it: a FreeSWITCH CDR and
// the voice agent's transcript, both keyed by the same call uuid.
func aiCallFixture(nativeID string, started time.Time) (calls.Observation, calls.Observation) {
	cdr := calls.Observation{
		NativeID:     calls.NativeID(nativeID),
		Kind:         calls.SourceFreeSwitchCDR,
		Direction:    calls.DirectionInbound,
		StartedAt:    started,
		DurationSecs: 63,
		FromNumber:   "+4930123456",
		ToNumber:     "9100",
		HangupCause:  "normal_clearing",
		IsAI:         true,
	}

	transcript := calls.Observation{
		NativeID:          calls.NativeID(nativeID),
		Kind:              calls.SourceAgentTranscript,
		Direction:         calls.DirectionInbound,
		StartedAt:         started.Add(time.Second),
		DurationSecs:      63,
		FromNumber:        "+4930123456",
		ToNumber:          "9100",
		IsAI:              true,
		TranscriptExcerpt: "hello there",
		Transcript: []calls.TranscriptTurn{
			{Role: calls.TurnRoleUser, Text: "hello there"},
			{Role: calls.TurnRoleAssistant, Text: "how can I help"},
		},
	}

	return cdr, transcript
}

func TestListCollapsesSiblingSourceRows(t *testing.T) {
	t.Parallel()

	started := testCallStarted()
	cdr, transcript := aiCallFixture("uuid-ai-1", started)

	page, err := ListCalls(t.Context(), newTestSystem(t, cdr, transcript), ListCallsQuery{})
	if err != nil {
		t.Fatalf("ListCalls: %v", err)
	}

	if len(page.Rows) != 1 {
		t.Fatalf("one call described by two sources must render one row, got %d: %+v", len(page.Rows), page.Rows)
	}

	row := page.Rows[0]
	if row.Kind != calls.SourceFreeSwitchCDR {
		t.Errorf("representative kind = %q, want the CDR row", row.Kind)
	}

	// The CDR cannot know the conversation; the collapsed row carries it.
	if row.Excerpt != "hello there" {
		t.Errorf("excerpt = %q, want the transcript's opening line", row.Excerpt)
	}

	if row.Duration != 63*time.Second {
		t.Errorf("duration = %v, want 63s", row.Duration)
	}

	if !row.IsAI {
		t.Error("collapsed row lost the AI badge")
	}
}

func TestListKeepsTranscriptWithoutCDR(t *testing.T) {
	t.Parallel()

	transcript := calls.Observation{
		NativeID:     calls.NativeID("uuid-orphan"),
		Kind:         calls.SourceAgentTranscript,
		Direction:    calls.DirectionInbound,
		StartedAt:    testCallStarted(),
		DurationSecs: 12,
		IsAI:         true,
	}

	page, err := ListCalls(t.Context(), newTestSystem(t, transcript), ListCallsQuery{})
	if err != nil {
		t.Fatalf("ListCalls: %v", err)
	}

	if len(page.Rows) != 1 {
		t.Fatalf("a transcript with no CDR is the only evidence of the call, got %d rows", len(page.Rows))
	}

	if page.Rows[0].Kind != calls.SourceAgentTranscript {
		t.Errorf("kind = %q, want the transcript row", page.Rows[0].Kind)
	}
}

func TestListCollapseSurvivesPageBoundary(t *testing.T) {
	// The page seam must land inside a call's source pair: pageSize calls,
	// each contributing two rows, gives exactly one duplicate per page.
	seeded := make([]calls.Observation, 0, 2*(pageSize+1))
	base := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)

	for i := range pageSize + 1 {
		cdr, transcript := aiCallFixture(fmt.Sprintf("uuid-%03d", i), base.Add(time.Duration(i)*time.Hour))
		seeded = append(seeded, cdr, transcript)
	}

	page, err := ListCalls(t.Context(), newTestSystem(t, seeded...), ListCallsQuery{})
	if err != nil {
		t.Fatalf("ListCalls: %v", err)
	}

	if len(page.Rows) != pageSize {
		t.Fatalf("page rows = %d, want a full page of %d distinct calls", len(page.Rows), pageSize)
	}

	if page.NextOlder == "" {
		t.Fatal("a full page with one call left must offer the older walk")
	}

	seen := make(map[string]struct{}, len(page.Rows))

	for _, row := range page.Rows {
		if _, dup := seen[row.NativeID]; dup {
			t.Fatalf("call %q rendered twice on one page", row.NativeID)
		}

		seen[row.NativeID] = struct{}{}
	}

	// The older page must carry the remaining call, not a repeat.
	older, err := ListCalls(t.Context(), newTestSystem(t, seeded...), ListCallsQuery{
		Before: mustParseRFC3339(t, page.NextOlder),
	})
	if err != nil {
		t.Fatalf("ListCalls older: %v", err)
	}

	if len(older.Rows) != 1 {
		t.Fatalf("older page rows = %d, want the one remaining call", len(older.Rows))
	}

	if _, dup := seen[older.Rows[0].NativeID]; dup {
		t.Errorf("call %q repeated across the page boundary", older.Rows[0].NativeID)
	}
}

func TestListCollapseKeepsTimestampTiesWhole(t *testing.T) {
	// A timestamp group is never split by the batch seam: with a batch that
	// exactly holds the whole group, the probe row proves the walk is done and
	// every call still renders once (the +1 probe is what tells an exhausted
	// tie group from a walk with older rows).
	previousBatch := listScanBatch

	listScanBatch = 6

	t.Cleanup(func() { listScanBatch = previousBatch })

	// Three calls at the SAME instant, each with a CDR and a transcript.
	started := testCallStarted()
	seeded := make([]calls.Observation, 0, 6)

	for i := range 3 {
		cdr, transcript := aiCallFixture(fmt.Sprintf("uuid-tie-%d", i), started)
		transcript.StartedAt = started // the tie under test
		seeded = append(seeded, cdr, transcript)
	}

	page, err := ListCalls(t.Context(), newTestSystem(t, seeded...), ListCallsQuery{})
	if err != nil {
		t.Fatalf("ListCalls: %v", err)
	}

	if len(page.Rows) != 3 {
		t.Fatalf("ties dropped or duplicated: rows = %d, want 3", len(page.Rows))
	}

	if page.NextOlder != "" {
		t.Error("an exhausted tie group must not offer an older walk")
	}
}

func TestListStallsLoudlyOnOversizedTieGroup(t *testing.T) {
	// A group wider than one batch cannot be walked by a started_at keyset;
	// the list must say so instead of silently truncating (the export's
	// documented failure mode, applied to the page).
	previousBatch := listScanBatch

	listScanBatch = 4

	t.Cleanup(func() { listScanBatch = previousBatch })

	started := testCallStarted()
	seeded := make([]calls.Observation, 0, 8)

	for i := range 4 {
		cdr, transcript := aiCallFixture(fmt.Sprintf("uuid-wide-%d", i), started)
		transcript.StartedAt = started
		seeded = append(seeded, cdr, transcript)
	}

	if _, err := ListCalls(t.Context(), newTestSystem(t, seeded...), ListCallsQuery{}); err == nil {
		t.Fatal("an unwalkable tie group must fail loudly")
	} else if !strings.Contains(err.Error(), "list_cursor_stalled") {
		t.Errorf("stall error = %v, want the list_cursor_stalled code", err)
	}
}

func TestDashboardStatsCollapseSiblingRows(t *testing.T) {
	t.Parallel()

	started := testCallStarted()
	cdr, transcript := aiCallFixture("uuid-ai-1", started)

	stats, err := DashboardStatsQuery(
		t.Context(),
		newTestSystem(t, cdr, transcript),
		started,
		ListCallsQuery{},
		dashboardWindowDays,
	)
	if err != nil {
		t.Fatalf("DashboardStatsQuery: %v", err)
	}

	if stats.Total != 1 {
		t.Errorf("Total = %d, want 1 (one call, not one row per source)", stats.Total)
	}

	if stats.AICalls != 1 || stats.HumanCalls != 0 {
		t.Errorf("AI/human = %d/%d, want 1/0", stats.AICalls, stats.HumanCalls)
	}

	if stats.AvgDuration != 63*time.Second {
		t.Errorf("AvgDuration = %v, want 63s", stats.AvgDuration)
	}

	var dayCalls int
	for _, day := range stats.ByDay {
		dayCalls += day.Calls
	}

	if dayCalls != 1 {
		t.Errorf("chart volume = %d, want 1", dayCalls)
	}
}

func TestExportKeepsEverySourceRow(t *testing.T) {
	t.Parallel()

	started := testCallStarted()
	cdr, transcript := aiCallFixture("uuid-ai-1", started)

	views, err := ExportCalls(t.Context(), newTestSystem(t, cdr, transcript), ListCallsQuery{})
	if err != nil {
		t.Fatalf("ExportCalls: %v", err)
	}

	if len(views) != 2 {
		t.Fatalf("export is a per-source dump: views = %d, want 2", len(views))
	}
}

func TestListRendersCollapsedRowOnce(t *testing.T) {
	t.Parallel()

	started := testCallStarted()
	cdr, transcript := aiCallFixture("uuid-ai-1", started)

	mux := newTestServer(t, cdr, transcript)

	_, body := getBody(t, mux, "/")
	if got := strings.Count(body, "/calls/freeswitch-cdr/uuid-ai-1"); got != 1 {
		t.Errorf("CDR detail links = %d, want 1", got)
	}

	if strings.Contains(body, "/calls/agent-transcript/uuid-ai-1") {
		t.Error("superseded transcript row must not render")
	}

	if !strings.Contains(body, "hello there") {
		t.Error("collapsed row should carry the transcript's opening line")
	}
}

func mustParseRFC3339(t *testing.T, value string) time.Time {
	t.Helper()

	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("parse cursor %q: %v", value, err)
	}

	return parsed
}
