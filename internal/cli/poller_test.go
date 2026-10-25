package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/system/v4"
	"github.com/larsartmann/pbx-lens/internal/search"
)

// pollTestLog records poller output for assertions. Structured key-value
// args (R14) render as space-joined k=v pairs.
type pollTestLog struct{ lines []string }

func (l *pollTestLog) printf(msg string, args ...any) {
	var kv strings.Builder

	for i := 0; i+1 < len(args); i += 2 {
		_, _ = fmt.Fprintf(&kv, " %v=%v", args[i], args[i+1])
	}

	l.lines = append(l.lines, strings.TrimSpace(msg+kv.String()))
}

func TestPollOnceIngestsAndRebuildsSearch(t *testing.T) {
	ctx, sys, index, webhookLog := openPollHarness(t)

	logs := &pollTestLog{}
	PollOnce(ctx, sys, index, PollConfig{WebhookLog: webhookLog}, logs.printf)

	if len(logs.lines) == 0 {
		t.Fatal("poller should log its pass summary")
	}

	hits, err := index.Query(ctx, "+17287289311", 10)
	if err != nil {
		t.Fatalf("query after poll: %v", err)
	}

	if len(hits) != 1 || hits[0].NativeID != "poll-1" {
		t.Fatalf("FTS index should carry the polled call, got %+v", hits)
	}
}

func TestPollOnceIsIdempotentAcrossPasses(t *testing.T) {
	ctx, sys, index, webhookLog := openPollHarness(t)

	PollOnce(ctx, sys, index, PollConfig{WebhookLog: webhookLog}, func(string, ...any) {})
	PollOnce(ctx, sys, index, PollConfig{WebhookLog: webhookLog}, func(string, ...any) {})

	hits, err := index.Query(ctx, "+17287289311", 10)
	if err != nil {
		t.Fatalf("query after two passes: %v", err)
	}

	if len(hits) != 1 {
		t.Fatalf("second pass must replay to zero new events (1 hit, got %d)", len(hits))
	}
}

func TestPollOnceSkipsMissingSourcesWithoutFailing(t *testing.T) {
	sys := openTestSystem(t)
	ctx := context.Background()

	logs := &pollTestLog{}
	PollOnce(ctx, sys, nil, PollConfig{CDRFile: filepath.Join(t.TempDir(), "missing.csv")}, logs.printf)

	if len(logs.lines) == 0 || !strings.Contains(logs.lines[len(logs.lines)-1], "poll pass") {
		t.Fatalf("a failing source must log and still finish the pass, got %v", logs.lines)
	}
}

func TestPollLoopStopsWithCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	sys, cleanup, err := OpenSystem(ctx, "")
	if err != nil {
		t.Fatalf("OpenSystem: %v", err)
	}

	t.Cleanup(cleanup)

	done := make(chan struct{})

	go func() {
		PollLoop(ctx, sys, nil, PollConfig{Every: time.Hour}, func(string, ...any) {})
		close(done)
	}()

	cancel()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("PollLoop must return once ctx is cancelled")
	}
}

func TestPollOnceAgeCutoffRejectsOverAgeObservations(t *testing.T) {
	sys := openTestSystem(t)
	ctx := context.Background()

	webhookLog := filepath.Join(t.TempDir(), "webhook.jsonl")

	freshAt := time.Now().UTC().Add(-1 * time.Hour)
	overAgeAt := time.Now().UTC().Add(-25 * time.Hour)

	line := webhookPollLine("poll-fresh", freshAt, "+17287289311", "+15555550123") + "\n" +
		webhookPollLine("poll-overage", overAgeAt, "+17287289312", "+15555550124")
	if err := os.WriteFile(webhookLog, []byte(line), 0o600); err != nil {
		t.Fatalf("write webhook log: %v", err)
	}

	index, err := search.Open(ctx, filepath.Join(t.TempDir(), "poll.db"))
	if err != nil {
		t.Fatalf("search.Open: %v", err)
	}

	t.Cleanup(func() { _ = index.Close() })

	logs := &pollTestLog{}
	PollOnce(ctx, sys, index, PollConfig{WebhookLog: webhookLog, MaxAge: 24 * time.Hour}, logs.printf)

	joined := strings.Join(logs.lines, "\n")
	if !strings.Contains(joined, "retention:age-cutoff") {
		t.Fatalf("over-age observation must be rejected with retention:age-cutoff, got %v", logs.lines)
	}

	if !strings.Contains(joined, "webhook-log:poll-overage: retention:age-cutoff") {
		t.Fatalf("rejection must carry the stream label, got %v", logs.lines)
	}

	fresh, err := index.Query(ctx, "+17287289311", 10)
	if err != nil {
		t.Fatalf("query fresh call: %v", err)
	}

	if len(fresh) != 1 || fresh[0].NativeID != "poll-fresh" {
		t.Fatalf("fresh call must be indexed, got %+v", fresh)
	}

	overAge, err := index.Query(ctx, "+17287289312", 10)
	if err != nil {
		t.Fatalf("query over-age call: %v", err)
	}

	if len(overAge) != 0 {
		t.Fatalf("over-age call must never enter the index, got %+v", overAge)
	}
}

// webhookPollLine renders the initiated event of one polled call as a JSONL
// line (the parser keys the observation off the initiated event, so a bare
// initiated line is already a complete observation; pair it with
// webhookHangupLine for a finished call).
func webhookPollLine(callID string, at time.Time, from, to string) string {
	stamp := at.Format(time.RFC3339)

	return `{"received_at":"` + stamp + `","remote":"198.51.100.7","body":{"data":` +
		`{"event_type":"call.initiated","occurred_at":"` + stamp + `","payload":` +
		`{"call_control_id":"` + callID + `","direction":"incoming","from":"` + from + `","to":"` + to + `"}}}}`
}

// webhookHangupLine completes a polled call with its hangup event.
func webhookHangupLine(callID string, at time.Time) string {
	stamp := at.Format(time.RFC3339)

	return `{"received_at":"` + stamp + `","remote":"198.51.100.7","body":{"data":` +
		`{"event_type":"call.hangup","occurred_at":"` + stamp + `","payload":` +
		`{"call_control_id":"` + callID + `","hangup_cause":"normal_clearing"}}}}`
}

// openPollHarness gives a poll test the full stack: a memory system, a search
// index, and a webhook log holding one completed call (poll-1). Shared so the
// pass tests exercise the same fixture instead of verbatim copies that drift.
func openPollHarness(t *testing.T) (context.Context, *system.System, *search.Index, string) {
	t.Helper()

	sys := openTestSystem(t)
	ctx := context.Background()

	webhookLog := filepath.Join(t.TempDir(), "webhook.jsonl")

	start := time.Date(2026, 10, 10, 9, 0, 0, 0, time.UTC)

	line := webhookPollLine("poll-1", start, "+17287289311", "+15555550123") + "\n" +
		webhookHangupLine("poll-1", start.Add(26*time.Second))
	if err := os.WriteFile(webhookLog, []byte(line), 0o600); err != nil {
		t.Fatalf("write webhook log: %v", err)
	}

	index, err := search.Open(ctx, filepath.Join(t.TempDir(), "poll.db"))
	if err != nil {
		t.Fatalf("search.Open: %v", err)
	}

	t.Cleanup(func() { _ = index.Close() })

	return ctx, sys, index, webhookLog
}

func TestPollOnceRecordsPerSourceCounters(t *testing.T) {
	// R15: a pass folds each source's stats into the counter set the ops
	// endpoint serves — latest pass wins, sources without data stay zeroed.
	sys := openTestSystem(t)
	ctx := context.Background()

	webhookLog := filepath.Join(t.TempDir(), "webhook.jsonl")

	line := webhookPollLine("ops-1", time.Date(2026, 10, 10, 9, 0, 0, 0, time.UTC),
		"+17287289311", "+15555550123") + "\n"
	if err := os.WriteFile(webhookLog, []byte(line), 0o600); err != nil {
		t.Fatalf("write webhook log: %v", err)
	}

	counters := NewSourceCounters(DefaultPollEvery)
	PollOnce(ctx, sys, nil, PollConfig{WebhookLog: webhookLog, Counters: counters}, func(string, ...any) {})

	sources, lastPass := counters.Snapshot()

	webhook := sources["webhook-log"]
	if webhook.Offered != 1 || webhook.Recorded != 1 {
		t.Errorf("webhook counters = %+v, want offered=1 recorded=1", webhook)
	}

	if lastPass.IsZero() {
		t.Error("lastPass must be stamped after a pass")
	}

	if _, exists := sources["telnyx-conversation"]; !exists {
		t.Error("unconfigured sources must still appear (zeroed), so ops sees the whole picture")
	}
}

func TestStalePass(t *testing.T) {
	// The pass-age verdict behind the ops payload's stale flag: one missed
	// tick is tolerated, two is a verdict, and "never ran" is stale now.
	now := time.Date(2026, 10, 15, 5, 0, 0, 0, time.UTC)
	every := 15 * time.Minute

	tests := []struct {
		name      string
		lastPass  time.Time
		every     time.Duration
		wantAge   time.Duration
		wantStale bool
	}{
		{
			name:      "never recorded is stale with sentinel age",
			lastPass:  time.Time{},
			wantAge:   -1 * time.Second,
			wantStale: true,
		},
		{
			name:      "fresh pass inside one cadence is healthy",
			lastPass:  now.Add(-5 * time.Minute),
			wantAge:   5 * time.Minute,
			wantStale: false,
		},
		{
			name:      "one missed tick is tolerated",
			lastPass:  now.Add(-2*every + time.Minute),
			wantAge:   2*every - time.Minute,
			wantStale: false,
		},
		{
			name:      "two cadences is a verdict",
			lastPass:  now.Add(-2 * every),
			wantAge:   2 * every,
			wantStale: false,
		},
		{
			name:      "past two cadences is stale",
			lastPass:  now.Add(-2*every - time.Second),
			wantAge:   2*every + time.Second,
			wantStale: true,
		},
		{
			name:      "zero cadence falls back to the default",
			lastPass:  now.Add(-2*DefaultPollEvery - time.Minute),
			every:     0,
			wantAge:   2*DefaultPollEvery + time.Minute,
			wantStale: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			age, stale := stalePass(tt.lastPass, tt.every, now)
			if age != tt.wantAge {
				t.Errorf("age = %v, want %v", age, tt.wantAge)
			}

			if stale != tt.wantStale {
				t.Errorf("stale = %v, want %v", stale, tt.wantStale)
			}
		})
	}
}

func TestSnapshotAnyReportsStaleVerdict(t *testing.T) {
	// The payload answers "is the poller alive" directly, so a monitor
	// never re-derives staleness from the timestamp and a cadence it has
	// to know out of band. The clock is injected — no sleeps.
	counters := NewSourceCounters(15 * time.Minute)
	freeze := time.Date(2026, 10, 15, 5, 0, 0, 0, time.UTC)
	counters.now = func() time.Time { return freeze }

	payload := counters.SnapshotAny().(map[string]any)

	if stale, ok := payload["stale"].(bool); !ok || !stale {
		t.Errorf("stale = %v (%T), want true before the first pass", payload["stale"], payload["stale"])
	}

	if age, ok := payload["last_pass_age_seconds"].(float64); !ok || age != -1 {
		t.Errorf("last_pass_age_seconds = %v, want -1 before the first pass", payload["last_pass_age_seconds"])
	}

	counters.Record("webhook-log", IngestStats{Total: 1, Recorded: 1})

	payload = counters.SnapshotAny().(map[string]any)

	if stale, ok := payload["stale"].(bool); !ok || stale {
		t.Errorf("stale = %v, want false right after a pass", payload["stale"])
	}

	if _, ok := payload["last_pass"].(time.Time); !ok {
		t.Errorf("last_pass = %T, want time.Time", payload["last_pass"])
	}

	counters.lastPass = freeze.Add(-45 * time.Minute)
	payload = counters.SnapshotAny().(map[string]any)

	if stale, ok := payload["stale"].(bool); !ok || !stale {
		t.Errorf("stale = %v, want true at three missed cadences", payload["stale"])
	}
}
