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
