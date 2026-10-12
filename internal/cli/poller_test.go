package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/pbx-lens/internal/search"
)

// pollTestLog records poller output for assertions. Structured key-value
// args (R14) render as space-joined k=v pairs.
type pollTestLog struct{ lines []string }

func (l *pollTestLog) printf(msg string, args ...any) {
	var kv strings.Builder

	for i := 0; i+1 < len(args); i += 2 {
		kv.WriteString(fmt.Sprintf(" %v=%v", args[i], args[i+1]))
	}

	l.lines = append(l.lines, strings.TrimSpace(msg+kv.String()))
}

func TestPollOnceIngestsAndRebuildsSearch(t *testing.T) {
	ctx := context.Background()

	sys, cleanup, err := OpenSystem(ctx, "")
	if err != nil {
		t.Fatalf("OpenSystem: %v", err)
	}

	t.Cleanup(cleanup)

	webhookLog := filepath.Join(t.TempDir(), "webhook.jsonl")

	line := `{"received_at":"2026-10-10T09:00:00Z","remote":"198.51.100.7","body":{"data":` +
		`{"event_type":"call.initiated","occurred_at":"2026-10-10T09:00:00Z","payload":` +
		`{"call_control_id":"poll-1","direction":"incoming","from":"+17287289311","to":"+15555550123"}}}}` +
		"\n" +
		`{"received_at":"2026-10-10T09:00:26Z","remote":"198.51.100.7","body":{"data":` +
		`{"event_type":"call.hangup","occurred_at":"2026-10-10T09:00:26Z","payload":` +
		`{"call_control_id":"poll-1","hangup_cause":"normal_clearing"}}}}`
	if err := os.WriteFile(webhookLog, []byte(line), 0o600); err != nil {
		t.Fatalf("write webhook log: %v", err)
	}

	index, err := search.Open(ctx, filepath.Join(t.TempDir(), "poll.db"))
	if err != nil {
		t.Fatalf("search.Open: %v", err)
	}

	t.Cleanup(func() { _ = index.Close() })

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
	ctx := context.Background()

	sys, cleanup, err := OpenSystem(ctx, "")
	if err != nil {
		t.Fatalf("OpenSystem: %v", err)
	}

	t.Cleanup(cleanup)

	webhookLog := filepath.Join(t.TempDir(), "webhook.jsonl")

	line := `{"received_at":"2026-10-10T09:00:00Z","remote":"198.51.100.7","body":{"data":` +
		`{"event_type":"call.initiated","occurred_at":"2026-10-10T09:00:00Z","payload":` +
		`{"call_control_id":"poll-1","direction":"incoming","from":"+17287289311","to":"+15555550123"}}}}` +
		"\n" +
		`{"received_at":"2026-10-10T09:00:26Z","remote":"198.51.100.7","body":{"data":` +
		`{"event_type":"call.hangup","occurred_at":"2026-10-10T09:00:26Z","payload":` +
		`{"call_control_id":"poll-1","hangup_cause":"normal_clearing"}}}}`
	if err := os.WriteFile(webhookLog, []byte(line), 0o600); err != nil {
		t.Fatalf("write webhook log: %v", err)
	}

	index, err := search.Open(ctx, filepath.Join(t.TempDir(), "poll.db"))
	if err != nil {
		t.Fatalf("search.Open: %v", err)
	}

	t.Cleanup(func() { _ = index.Close() })

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
	ctx := context.Background()

	sys, cleanup, err := OpenSystem(ctx, "")
	if err != nil {
		t.Fatalf("OpenSystem: %v", err)
	}

	t.Cleanup(cleanup)

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
	ctx := context.Background()

	sys, cleanup, err := OpenSystem(ctx, "")
	if err != nil {
		t.Fatalf("OpenSystem: %v", err)
	}

	t.Cleanup(cleanup)

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

// webhookPollLine renders one initiated+hangup webhook pair as JSONL for a
// call with the given id and start time (the hangup carries no distinct
// timestamp; the parser keys the observation off the initiated event).
func webhookPollLine(callID string, at time.Time, from, to string) string {
	stamp := at.Format(time.RFC3339)

	return `{"received_at":"` + stamp + `","remote":"198.51.100.7","body":{"data":` +
		`{"event_type":"call.initiated","occurred_at":"` + stamp + `","payload":` +
		`{"call_control_id":"` + callID + `","direction":"incoming","from":"` + from + `","to":"` + to + `"}}}}`
}

func TestPollOnceRecordsPerSourceCounters(t *testing.T) {
	// R15: a pass folds each source's stats into the counter set the ops
	// endpoint serves — latest pass wins, sources without data stay zeroed.
	ctx := context.Background()

	sys, cleanup, err := OpenSystem(ctx, "")
	if err != nil {
		t.Fatalf("OpenSystem: %v", err)
	}

	t.Cleanup(cleanup)

	webhookLog := filepath.Join(t.TempDir(), "webhook.jsonl")

	line := `{"received_at":"2026-10-10T09:00:00Z","remote":"198.51.100.7","body":{"data":` +
		`{"event_type":"call.initiated","occurred_at":"2026-10-10T09:00:00Z","payload":` +
		`{"call_control_id":"ops-1","direction":"incoming","from":"+17287289311","to":"+15555550123"}}}}` +
		"\n"
	if err := os.WriteFile(webhookLog, []byte(line), 0o600); err != nil {
		t.Fatalf("write webhook log: %v", err)
	}

	counters := NewSourceCounters()
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
