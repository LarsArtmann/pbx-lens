package web

import (
	"bytes"
	"context"
	"fmt"
	"hash/fnv"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/pbx-lens/internal/calls"
	"github.com/larsartmann/pbx-lens/internal/cli"
)

// Performance sanity rails (plan 05-40 M16/F49-F50): not gates, but the
// numbers a future "the list feels slow" report gets compared against.

// TestDetail500TurnTranscriptRenders pins the large-transcript render
// (F50): a 500-turn transcript renders completely, the ?q= redirect lands
// at the matched turn even at the far end of the log, and the payload
// stays proportional (500 turns, not 500 duplicated copies).
func TestDetail500TurnTranscriptRenders(t *testing.T) {
	started := testCallStarted()

	turns := make([]calls.TranscriptTurn, 0, 500)

	for i := range 500 {
		role := calls.TurnRoleUser

		if i%2 == 1 {
			role = calls.TurnRoleAssistant
		}

		text := fmt.Sprintf("turn %d: routine status exchange, nothing to see here", i)
		if i == 499 {
			text = "needle: the one matching turn at the far end"
		}

		turns = append(turns, calls.TranscriptTurn{Role: role, Text: text})
	}

	obs := webObservation("conv-500", started)
	obs.Transcript = turns

	mux := newTestServer(t, obs)

	recorder := httptestGet(t, mux, "/calls/telnyx-conversation/conv-500?q=needle")
	if recorder.Code != http.StatusFound {
		t.Fatalf("detail?q=needle status = %d, want 302", recorder.Code)
	}

	if got, want := recorder.Header().
		Get("Location"),
		"/calls/telnyx-conversation/conv-500?q=needle&at=499#turn-499"; got != want {
		t.Fatalf("redirect Location = %q, want %q", got, want)
	}

	status, body := getBody(t, mux, recorder.Header().Get("Location"))
	if status != http.StatusOK {
		t.Fatalf("anchored render status = %d", status)
	}

	if !strings.Contains(body, `id="turn-499"`) {
		t.Error("far-end matched turn missing its anchor")
	}

	for _, probe := range []string{"turn 0: routine", "turn 249: routine", "needle: the one matching turn"} {
		if !strings.Contains(body, probe) {
			t.Errorf("transcript truncated: %q missing from render", probe)
		}
	}

	if copies := strings.Count(body, "turn 249: routine status exchange"); copies != 1 {
		t.Errorf("turn 249 rendered %d times, want 1 (500 turns must not duplicate)", copies)
	}

	t.Logf("500-turn detail page renders %d bytes", len(body))
}

// BenchmarkListCalls10k is the 10k-row index rail (F49): the list query
// the calls table pays on every page view. p95 experience on the operator
// LAN stays dominated by wire+render; the query itself must stay in the
// low tens of milliseconds against real sqlite (run: go test -bench . ).
func BenchmarkListCalls10k(b *testing.B) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	sys, cleanup, err := cli.OpenSystem(ctx, filepath.Join(b.TempDir(), "bench-10k.db"))
	if err != nil {
		b.Fatalf("OpenSystem: %v", err)
	}

	defer cleanup()

	base := testCallStarted().Add(-30 * 24 * time.Hour)

	for batch := range 5 {
		observations := make([]calls.Observation, 0, 2000)

		for i := range 2000 {
			n := batch*2000 + i
			observations = append(observations, calls.Observation{
				NativeID:     calls.NativeID(fmt.Sprintf("bench-%05d", n)),
				Kind:         calls.SourceTelnyxConversation,
				Direction:    calls.DirectionOutbound,
				StartedAt:    base.Add(time.Duration(n) * 3 * time.Minute),
				DurationSecs: 30 + n%90,
				FromNumber:   "+17287289311",
				ToNumber:     "+15555550123",
				IsAI:         n%3 == 0,
			})
		}

		if stats := cli.Ingest(ctx, sys, observations); len(stats.Rejected) > 0 {
			b.Fatalf("ingest rejected: %v", stats.Rejected)
		}
	}

	if err := cli.DrainProjections(ctx, sys); err != nil {
		b.Fatalf("drain: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		if _, err := ListCalls(ctx, sys, ListCallsQuery{}); err != nil {
			b.Fatalf("ListCalls: %v", err)
		}
	}
}

// BenchmarkETagHashCost is the concurrent-poll CPU ceiling rail (F31): the
// ETag middleware body-hashes every cacheable 200, so each conditional
// poll from each client pays one hash of the body. Recorded at
// load-average ~74 on the 32-core dev box (2026-10-15):
//
//	25 KB page ≈ 22 µs/op    (≈45k polls/s per core)
//	122 KB page ≈ 104 µs/op  (≈9.6k polls/s per core)
//
// The dashboard's 30 s poll and the staleness banner's 60 s poll across a
// LAN of operators stay orders of magnitude below one core — the hash
// cost is not a scaling concern at any plausible operator count.
func BenchmarkETagHashCost(b *testing.B) {
	small := bytes.Repeat([]byte("x"), 25*1024)
	large := bytes.Repeat([]byte("x"), 122*1024)

	b.Run("page25KB", func(b *testing.B) {
		b.SetBytes(int64(len(small)))
		b.ReportAllocs()

		for b.Loop() {
			_ = pollHash(small)
		}
	})

	b.Run("page122KB", func(b *testing.B) {
		b.SetBytes(int64(len(large)))
		b.ReportAllocs()

		for b.Loop() {
			_ = pollHash(large)
		}
	})
}

// pollHash mirrors go-etag's defaultHashFunc (FNV-64a, lowercase hex;
// server@v0.6.1 etag.go:107-125) so the benchmark measures the exact
// per-poll cost the middleware pays without importing an unexported
// function. A custom HashFunc would change the constant — re-measure.
func pollHash(data []byte) string {
	h := fnv.New64a()
	_, _ = h.Write(data)

	return strconv.FormatUint(h.Sum64(), 16)
}
