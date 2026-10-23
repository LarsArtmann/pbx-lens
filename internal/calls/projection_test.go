package calls

import (
	"testing"
	"time"
)

func TestViewObservationRoundTripHashesIdentically(t *testing.T) {
	obs := Observation{
		NativeID:          "conv-roundtrip",
		Kind:              SourceTelnyxConversation,
		Direction:         DirectionOutbound,
		StartedAt:         time.Date(2026, 10, 9, 19, 51, 25, 0, time.UTC),
		DurationSecs:      26,
		FromNumber:        "+17287289311",
		ToNumber:          "+14084757593",
		HangupCause:       "normal_clearing",
		IsAI:              true,
		TranscriptExcerpt: "Woolly, did you take my eggs?",
		Transcript: []TranscriptTurn{
			{Role: "user", Text: "Woolly, did you take my eggs?", At: time.Unix(10, 0).UTC()},
			{Role: "assistant", Text: "I did NOT take your eggs!", At: time.Unix(20, 0).UTC()},
		},
		RecordingRef: "rec-1",
		MOS:          4.2,
	}

	view := ViewFrom(obs, Enrichment{CostMicros: 2000, CostCurrency: "USD", InsightSummary: "eggs"})
	rebuilt := view.Observation()

	if rebuilt.ContentHash() != obs.ContentHash() {
		t.Fatalf(
			"rebuild must hash identically: %s != %s",
			rebuilt.ContentHash(), obs.ContentHash(),
		)
	}

	if rebuilt.ContentHash() != view.ContentHash {
		t.Fatalf(
			"rebuild must match the stored view hash: %s != %s",
			rebuilt.ContentHash(),
			view.ContentHash,
		)
	}

	if _, err := rebuilt.StreamID(); err != nil {
		t.Fatalf("rebuild must stay stream-addressable: %v", err)
	}
}

// TestViewRoundTripPreservesSourceEventsWithTheSameHash pins the two halves of
// the SourceEvent contract in one place: the markers survive the view round
// trip (so the page can render them from a stored row), and they stay out of
// ContentHash (so persisting them cannot fork a stored row).
func TestViewRoundTripPreservesSourceEventsWithTheSameHash(t *testing.T) {
	t.Parallel()

	obs := Observation{
		NativeID:     "conv-silence",
		Kind:         SourceTelnyxConversation,
		Direction:    DirectionOutbound,
		StartedAt:    time.Date(2026, 10, 18, 3, 23, 29, 0, time.UTC),
		DurationSecs: 45,
		FromNumber:   "+17287289311",
		ToNumber:     "+17287289311",
		IsAI:         true,
		Transcript: []TranscriptTurn{
			{Role: TurnRoleAssistant, Text: "Guten Tag.", At: time.Date(2026, 10, 18, 3, 23, 30, 0, time.UTC)},
		},
		SourceEvents: []SourceEvent{
			{Role: "system", Text: "[long silence]", At: time.Date(2026, 10, 18, 3, 23, 44, 0, time.UTC)},
			{Role: "system", Text: "[long silence]", At: time.Date(2026, 10, 18, 3, 24, 3, 0, time.UTC)},
		},
	}

	withoutMarkers := obs
	withoutMarkers.SourceEvents = nil

	if obs.ContentHash() != withoutMarkers.ContentHash() {
		t.Fatal("source events must not enter the content hash")
	}

	view := ViewFrom(obs, Enrichment{})
	rebuilt := view.Observation()

	if len(rebuilt.SourceEvents) != 2 {
		t.Fatalf("round trip lost the markers: %+v", rebuilt.SourceEvents)
	}

	if rebuilt.SourceEvents[1].Text != "[long silence]" {
		t.Fatalf("marker text changed in the round trip: %+v", rebuilt.SourceEvents[1])
	}

	if rebuilt.ContentHash() != view.ContentHash {
		t.Fatalf("rebuilt hash %s != stored %s", rebuilt.ContentHash(), view.ContentHash)
	}

	if CountRole(rebuilt.Transcript, TurnRoleUser) != 0 {
		t.Fatal("precondition: the fixture has no user turn")
	}
}
