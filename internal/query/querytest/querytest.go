// Package querytest is the query layer's shared test fixtures: one seeded
// call time, one corpus observation shape, one booted-and-drained system —
// so every suite (query, web, serve) derives from the same seams instead of
// 28 scattered literals. Test-only by convention; nothing in cmd/ imports it.
package querytest

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/system/v4"
	"github.com/larsartmann/pbx-lens/internal/calls"
	"github.com/larsartmann/pbx-lens/internal/cli"
)

// Fixture call durations, named once so no suite has to reverse-engineer a
// corpus: the standard conversation runs 42 s, the AI call pair 63 s, and the
// forensic join call 30 s (its live shape, 2026-10-23 12:09).
const (
	corpusCallSecs   = 42
	aiPairCallSecs   = 63
	forensicCallSecs = 30
)

// CallStarted is the shared seeded call time (plan T11.1): one fixed instant
// every test derives from, so changing the corpus seed is one line instead
// of 28 scattered literals.
func CallStarted() time.Time {
	return time.Date(2026, 10, 9, 19, 51, 25, 0, time.UTC)
}

// Observation is the shared corpus row shape: a two-turn Telnyx conversation
// with the standard parties and duration.
func Observation(nativeID string, started time.Time) calls.Observation {
	return calls.Observation{
		NativeID:     calls.NativeID(nativeID),
		Kind:         calls.SourceTelnyxConversation,
		Direction:    calls.DirectionOutbound,
		StartedAt:    started,
		DurationSecs: corpusCallSecs,
		FromNumber:   "+17287289311",
		ToNumber:     "+15555550123",
		Transcript: []calls.TranscriptTurn{
			{Role: "user", Text: "hello there"},
			{Role: "assistant", Text: "how can I help"},
		},
	}
}

// System opens a fresh sqlite index and ingests the observations, drained so
// the read model is queryable. Query-level tests need the system itself; the
// HTTP tests build their own server over it.
func System(t *testing.T, observations ...calls.Observation) *system.System {
	t.Helper()

	ctx := t.Context()

	sys, cleanup, err := cli.OpenSystem(ctx, filepath.Join(t.TempDir(), "querytest.db"))
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

// JoinUUID is the FreeSWITCH call uuid shared by every source that observed
// the forensic join call (the 2026-10-23 12:09 live instance): a 30 s call
// to the voice agent's extension 9100 whose CDR row carried the facts, whose
// transcript row carried the words, and whose recording lived at
// {uuid}_ai_agent.wav — none reachable from the CDR detail page.
const JoinUUID = "9d69060b-1003-401a-8a0a-f083a81c3b9c"

// JoinStarted is the forensic join call's start instant.
func JoinStarted() time.Time {
	return time.Date(2026, 10, 23, 10, 9, 51, 0, time.UTC)
}

// AgentCDR is the forensic join call's FreeSWITCH CDR row: the facts half.
func AgentCDR() calls.Observation {
	return calls.Observation{
		NativeID:     calls.NativeID(JoinUUID),
		Kind:         calls.SourceFreeSwitchCDR,
		Direction:    calls.DirectionOutbound,
		StartedAt:    JoinStarted(),
		DurationSecs: forensicCallSecs,
		FromNumber:   "4084757593",
		ToNumber:     "9100",
		HangupCause:  "NORMAL_CLEARING",
		IsAI:         true,
	}
}

// AgentTranscript is the forensic join call's agent-transcript row: the
// words half, keyed by the same uuid.
func AgentTranscript() calls.Observation {
	return calls.Observation{
		NativeID:     calls.NativeID(JoinUUID),
		Kind:         calls.SourceAgentTranscript,
		Direction:    calls.DirectionOutbound,
		StartedAt:    JoinStarted(),
		DurationSecs: forensicCallSecs,
		FromNumber:   "4084757593",
		ToNumber:     "9100",
		IsAI:         true,
		Transcript: []calls.TranscriptTurn{
			{Role: calls.TurnRoleAssistant, Text: "Guten Tag, hier ist Woolly."},
			{Role: calls.TurnRoleUser, Text: "Ich wollte wegen der Rechnung fragen."},
		},
	}
}

// AICallPair is one real call as the index sees it: a FreeSWITCH CDR and the
// voice agent's transcript, both keyed by the same call uuid (the collapse
// rule's canonical fixture).
func AICallPair(nativeID string, started time.Time) (calls.Observation, calls.Observation) {
	cdr := calls.Observation{
		NativeID:     calls.NativeID(nativeID),
		Kind:         calls.SourceFreeSwitchCDR,
		Direction:    calls.DirectionInbound,
		StartedAt:    started,
		DurationSecs: aiPairCallSecs,
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
		DurationSecs:      aiPairCallSecs,
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
