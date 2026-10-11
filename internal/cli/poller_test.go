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

// pollTestLog records poller output for assertions.
type pollTestLog struct{ lines []string }

func (l *pollTestLog) printf(msg string, args ...any) {
	l.lines = append(l.lines, strings.TrimSpace(fmt.Sprintf(msg, args...)))
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
