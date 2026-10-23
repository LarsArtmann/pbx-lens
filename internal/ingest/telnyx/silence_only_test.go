package telnyx

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/larsartmann/pbx-lens/internal/calls"
	"github.com/larsartmann/pbx-lens/internal/wire"
)

// The silence-only fixture is the 2026-10-23 forensic call (see
// testdata/silence-only/README.md): a real two-sided conversation whose
// message log carries ZERO user turns and two `[long silence]` system rows,
// while the recording holds a human greeting at 0.3 s and a closing line at
// 39.9 s. These tests pin what the mapper must do with it.

type silenceOnlyFixture struct {
	conv Conversation
	msgs []Message
	rec  Recording
}

func loadSilenceOnly(t *testing.T) silenceOnlyFixture {
	t.Helper()

	read := func(name string, dst any) {
		t.Helper()

		raw, err := os.ReadFile(filepath.Join("testdata", "silence-only", name))
		if err != nil {
			t.Fatalf("read fixture %s: %v", name, err)
		}

		if err := json.Unmarshal(raw, dst); err != nil {
			t.Fatalf("parse fixture %s: %v", name, err)
		}
	}

	var fixture silenceOnlyFixture

	read("conversation.json", &fixture.conv)
	read("messages.json", &fixture.msgs)
	read("recording.json", &fixture.rec)

	return fixture
}

func mapSilenceOnly(t *testing.T) calls.Observation {
	t.Helper()

	fixture := loadSilenceOnly(t)

	obs, err := MapConversation(fixture.conv, fixture.msgs, &fixture.rec, DefaultMapOptions())
	if err != nil {
		t.Fatalf("map the silence-only fixture: %v", err)
	}

	return obs
}

func TestSilenceOnlyKeepsMarkersOutOfTheTranscript(t *testing.T) {
	t.Parallel()

	obs := mapSilenceOnly(t)

	if len(obs.Transcript) != 2 {
		t.Fatalf("transcript must carry the two assistant turns, got %d: %+v", len(obs.Transcript), obs.Transcript)
	}

	if userTurns := calls.CountRole(obs.Transcript, calls.TurnRoleUser); userTurns != 0 {
		t.Fatalf("the source logged no user turn, got %d", userTurns)
	}

	for _, turn := range obs.Transcript {
		if turn.Role != calls.TurnRoleAssistant {
			t.Fatalf("every transcript turn is the assistant's, got %q", turn.Role)
		}
	}
}

func TestSilenceOnlyPreservesTheIdleMarkersAsSourceEvents(t *testing.T) {
	t.Parallel()

	obs := mapSilenceOnly(t)

	if len(obs.SourceEvents) != 2 {
		t.Fatalf("both [long silence] rows must survive as source events, got %d: %+v",
			len(obs.SourceEvents), obs.SourceEvents)
	}

	for i, event := range obs.SourceEvents {
		if event.Role != "system" {
			t.Errorf("event %d role = %q, want system", i, event.Role)
		}

		// The wording is explicitly NOT a stable contract (Telnyx docs,
		// ai-assistant-start): match the ROLE and the shape, log the text.
		if event.Text == "" {
			t.Errorf("event %d lost its text; the marker text is the whole point", i)
		}
	}

	if !obs.SourceEvents[0].At.Before(obs.SourceEvents[1].At) {
		t.Fatalf("source events must be chronological: %v then %v",
			obs.SourceEvents[0].At, obs.SourceEvents[1].At)
	}
}

// TestSilenceOnlyDropsNothing is the anti-regression for the actual bug: the
// old mapper `continue`d on every non user/assistant row, so four source
// messages became two turns and the admission of silence vanished.
func TestSilenceOnlyDropsNothing(t *testing.T) {
	t.Parallel()

	fixture := loadSilenceOnly(t)
	obs := mapSilenceOnly(t)

	got := len(obs.Transcript) + len(obs.SourceEvents)
	if got != len(fixture.msgs) {
		t.Fatalf("mapped %d of %d messages; every message must land in exactly one lane:\n%+v",
			got, len(fixture.msgs), obs)
	}
}

// TestSilenceOnlyAnchorsTurnsOnSentAt pins the timestamp source. Telnyx stamps
// created_at at PERSISTENCE (all four rows of this call landed inside one
// second, a minute after the speech) and sent_at at dispatch. Anchoring on
// created_at made turn times unmatchable against the recording.
func TestSilenceOnlyAnchorsTurnsOnSentAt(t *testing.T) {
	t.Parallel()

	obs := mapSilenceOnly(t)

	want := []string{"2026-10-18T03:23:30Z", "2026-10-18T03:23:45Z"}

	for i, turn := range obs.Transcript {
		if got := turn.At.UTC().Format(time.RFC3339); got != want[i] {
			t.Errorf("turn %d At = %s, want the sent_at %s", i, got, want[i])
		}
	}

	// The event offsets are the ones the report measured, within the
	// whole-second truncation the domain enforces.
	if got := obs.SourceEvents[1].At.UTC().Format(time.RFC3339); got != "2026-10-18T03:24:03Z" {
		t.Errorf("final idle marker At = %s, want 2026-10-18T03:24:03Z", got)
	}
}

// TestSpokeAtFallsBackToCreatedAt keeps sources (and fixtures) without sent_at
// working exactly as before.
func TestSpokeAtFallsBackToCreatedAt(t *testing.T) {
	t.Parallel()

	created := parseFlexTime(t, "2026-10-09T19:50:12Z")
	msg := Message{Role: "assistant", Text: "hi", CreatedAt: created}

	if got := msg.SpokeAt().UTC().Format(time.RFC3339); got != "2026-10-09T19:50:12Z" {
		t.Fatalf("SpokeAt without sent_at = %s, want created_at", got)
	}
}

// TestSilenceOnlyHashIgnoresSourceEvents is the storage-safety pin: persisting
// markers must not fork a stored call_views row.
func TestSilenceOnlyHashIgnoresSourceEvents(t *testing.T) {
	t.Parallel()

	obs := mapSilenceOnly(t)

	stripped := obs
	stripped.SourceEvents = nil

	if obs.ContentHash() != stripped.ContentHash() {
		t.Fatalf("source events entered the content hash: %s vs %s",
			obs.ContentHash(), stripped.ContentHash())
	}

	if len(obs.SourceEvents) == 0 {
		t.Fatal("fixture precondition: markers must exist for this pin to mean anything")
	}
}

// TestSilenceOnlyMarksTheSelfDial pins the second anomaly the page surfaces:
// both participants are the agent's own DID, because the bench dialled the
// number that terminates in the agent.
func TestSilenceOnlyMarksTheSelfDial(t *testing.T) {
	t.Parallel()

	obs := mapSilenceOnly(t)

	if obs.FromNumber == "" || obs.FromNumber != obs.ToNumber {
		t.Fatalf("fixture precondition: both parties are +17287289311, got %q -> %q",
			obs.FromNumber, obs.ToNumber)
	}
}

// parseFlexTime decodes one RFC3339 stamp through the wire type the API
// decoder uses, so the test exercises the real path.
func parseFlexTime(t *testing.T, stamp string) wire.FlexTime {
	t.Helper()

	var decoded wire.FlexTime
	if err := json.Unmarshal([]byte(`"`+stamp+`"`), &decoded); err != nil {
		t.Fatalf("decode %s: %v", stamp, err)
	}

	return decoded
}
