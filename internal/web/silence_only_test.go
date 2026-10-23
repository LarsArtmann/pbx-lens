package web

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/pbx-lens/internal/calls"
)

// The 2026-10-23 forensic call, replayed through the real ingest -> projection
// -> page path: both parties are the agent's own DID, the message log has two
// assistant turns and zero user turns, and the only sign of the far end's
// audio is the source's own idle markers.
func silenceOnlyObservation() calls.Observation {
	started := time.Date(2026, 10, 18, 3, 23, 29, 0, time.UTC)

	return calls.Observation{
		NativeID:     "acea3e91-5464-42c1-9439-57296942687a",
		Kind:         calls.SourceTelnyxConversation,
		Direction:    calls.DirectionOutbound,
		StartedAt:    started,
		DurationSecs: 45,
		FromNumber:   "+17287289311",
		ToNumber:     "+17287289311",
		IsAI:         true,
		Transcript: []calls.TranscriptTurn{
			{
				Role: calls.TurnRoleAssistant,
				Text: "Guten Tag, hier ist Mara von Lars Artmann.",
				At:   started.Add(1 * time.Second),
			},
			{
				Role: calls.TurnRoleAssistant,
				Text: "Ich rufe wegen einer offenen Rechnung an.",
				At:   started.Add(16 * time.Second),
			},
		},
		SourceEvents: []calls.SourceEvent{
			{Role: "system", Text: "[long silence]", At: started.Add(15 * time.Second)},
			{Role: "system", Text: "[long silence]", At: started.Add(34 * time.Second)},
		},
	}
}

func TestDetailRendersTheSourceLogForASilentTranscript(t *testing.T) {
	mux := newTestServer(t, silenceOnlyObservation())

	status, body := getBody(t, mux, "/calls/telnyx-conversation/acea3e91-5464-42c1-9439-57296942687a")
	if status != http.StatusOK {
		t.Fatalf("GET detail status = %d, want 200", status)
	}

	for _, want := range []string{
		"Source log",     // the marker pane exists
		"[long silence]", // the source's own wording, verbatim
		"source markers", // the facts-card badge
		"no user turn",   // the zero-user-turn signal
		"every recorded turn is the assistant",
		"self-dial", // from == to
		"unlogged",  // the recording tail signal
		"aria-label=\"Source log\"",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("detail page missing %q", want)
		}
	}

	// The transcript is still honest about who spoke: two assistant lines,
	// no invented user line. The library scrollback (no ?q=) renders its own
	// tag classes, so count the tag TEXT rather than a class name.
	if got := strings.Count(body, ">assistant<"); got != 2 {
		t.Errorf("assistant turn tags = %d, want 2", got)
	}

	if got := strings.Count(body, ">user<"); got != 0 {
		t.Errorf("user turn tags = %d, want 0 (the source logged none)", got)
	}
}

func TestDetailEmptyStateDistinguishesMarkersFromNoTranscript(t *testing.T) {
	markers := silenceOnlyObservation()
	markers.Transcript = nil

	silent := newTestServer(t, markers)

	status, body := getBody(t, silent, "/calls/telnyx-conversation/acea3e91-5464-42c1-9439-57296942687a")
	if status != http.StatusOK {
		t.Fatalf("GET detail status = %d, want 200", status)
	}

	if !strings.Contains(body, "No spoken turns recorded") {
		t.Error("a source-only-markers call must say no SPOKEN turns were recorded")
	}

	if strings.Contains(body, "No source recorded a transcript") {
		t.Error("the generic empty state lies about a call whose source wrote markers")
	}

	// A truly empty call keeps the historical copy.
	bare := webObservation("conv-bare", time.Date(2026, 10, 12, 9, 1, 0, 0, time.UTC))
	bare.Transcript = nil
	dash := newTestServer(t, bare)

	status, body = getBody(t, dash, "/calls/telnyx-conversation/conv-bare")
	if status != http.StatusOK {
		t.Fatalf("GET detail status = %d, want 200", status)
	}

	if !strings.Contains(body, "No source recorded a transcript for this call yet") {
		t.Error("a call with no transcript and no markers must keep the plain copy")
	}
}

func TestDiagnosticItemsStayOffForACleanCall(t *testing.T) {
	clean := calls.CallView{
		FromNumber:   "+17287289311",
		ToNumber:     "+15555550123",
		StartedAt:    time.Date(2026, 10, 12, 9, 1, 0, 0, time.UTC),
		DurationSecs: 12,
		Transcript: []calls.TranscriptTurn{
			{Role: calls.TurnRoleUser, Text: "hello", At: time.Date(2026, 10, 12, 9, 1, 5, 0, time.UTC)},
			{Role: calls.TurnRoleAssistant, Text: "hi", At: time.Date(2026, 10, 12, 9, 1, 8, 0, time.UTC)},
		},
	}

	if items := diagnosticItems(clean); len(items) != 0 {
		t.Fatalf("a clean call must add no diagnostics, got %+v", items)
	}
}

// TestTranscriptTailMeasuredFromTheSpeech is the arithmetic pin for the signal
// that fires on the forensic call: the last turn started 29 s before the line
// dropped while the far end was still talking.
func TestTranscriptTailMeasuredFromTheSpeech(t *testing.T) {
	view := calls.CallView{
		StartedAt:    time.Date(2026, 10, 18, 3, 23, 29, 0, time.UTC),
		DurationSecs: 45,
		Transcript: []calls.TranscriptTurn{
			{Role: calls.TurnRoleAssistant, Text: "a", At: time.Date(2026, 10, 18, 3, 23, 30, 0, time.UTC)},
			{Role: calls.TurnRoleAssistant, Text: "b", At: time.Date(2026, 10, 18, 3, 23, 45, 0, time.UTC)},
		},
	}

	tail, ok := transcriptTail(view)
	if !ok {
		t.Fatal("a stamped transcript with a duration must yield a tail")
	}

	// 03:23:45 - 03:23:29 = 16 s offset; 45 s recording - 16 s = 29 s.
	if tail != 29*time.Second {
		t.Fatalf("tail = %s, want 29s (45s recording - the 16s offset of the last turn)", tail)
	}

	if tail < transcriptTailThreshold {
		t.Fatalf("the forensic call must cross the %s bar, got %s", transcriptTailThreshold, tail)
	}

	// An unstamped transcript (agent JSONL) yields no signal rather than a
	// guess.
	view.Transcript[0].At = time.Time{}
	view.Transcript[1].At = time.Time{}

	if _, ok := transcriptTail(view); ok {
		t.Fatal("an unstamped transcript must not produce a tail")
	}
}
