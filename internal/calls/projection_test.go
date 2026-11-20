package calls

import (
	"encoding/json"
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
	rebuilt := view.Observation

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
	rebuilt := view.Observation

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

// The read model is a COMPOSITION now (PC-1, 2026-11-20): the view embeds
// Observation and Enrichment instead of mirroring 17 fields by hand. JSON
// identity is the contract that makes the swap invisible: the flattened
// keys are exactly the pre-embed set (decode is key-based, so rows written
// before the embed read fine), and the embedded observation still hashes to
// the stored hash — the invariant the enrich pass's command addressing
// relies on.
func TestCallViewJSONIdentityAcrossTheEmbed(t *testing.T) {
	// A row written by the PRE-embed layout: mos sits AFTER the enrichment
	// keys (the old struct's field order), a position the embedded layout
	// no longer emits.
	legacy := `{
		"native_id": "conv-1",
		"kind": "telnyx-conversation",
		"direction": "inbound",
		"started_at": "2026-11-20T10:00:00Z",
		"duration_secs": 42,
		"from_number": "+17287289311",
		"to_number": "+15555550123",
		"hangup_cause": "normal_clearing",
		"is_ai": true,
		"transcript_excerpt": "hello",
		"transcript": [{"role":"user","text":"hello"}],
		"source_events": [],
		"recording_ref": "rec-1",
		"cost_micros": 2000,
		"cost_currency": "USD",
		"insight_summary": "eggs",
		"mos": 4.2,
		"content_hash": "stored-hash"
	}`

	var view CallView
	if err := json.Unmarshal([]byte(legacy), &view); err != nil {
		t.Fatalf("pre-embed row must decode: %v", err)
	}

	if view.CostMicros != 2000 || view.MOS != 4.2 || view.ContentHash != "stored-hash" ||
		len(view.Transcript) != 1 || view.FromNumber != "+17287289311" {
		t.Fatalf("pre-embed row decoded wrong: %+v", view)
	}

	// The embedded layout must flatten to EXACTLY the pre-embed key set —
	// a lost key is a lost column for every consumer of the stored JSON.
	fresh := ViewFrom(Observation{
		NativeID:          "conv-1",
		Kind:              SourceTelnyxConversation,
		Direction:         DirectionInbound,
		StartedAt:         time.Date(2026, 11, 20, 10, 0, 0, 0, time.UTC),
		DurationSecs:      42,
		FromNumber:        "+17287289311",
		ToNumber:          "+15555550123",
		HangupCause:       "normal_clearing",
		IsAI:              true,
		TranscriptExcerpt: "hello",
		Transcript:        []TranscriptTurn{{Role: TurnRoleUser, Text: "hello"}},
		SourceEvents:      []SourceEvent{{Role: "idle", Text: "[long silence]"}},
		RecordingRef:      "rec-1",
		MOS:               4.2,
	}, Enrichment{CostMicros: 2000, CostCurrency: "USD", InsightSummary: "eggs"})

	encoded, err := json.Marshal(fresh)
	if err != nil {
		t.Fatalf("marshal view: %v", err)
	}

	var keys map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &keys); err != nil {
		t.Fatalf("re-marshal decode: %v", err)
	}

	for _, want := range []string{
		"native_id", "kind", "direction", "started_at", "duration_secs",
		"from_number", "to_number", "hangup_cause", "is_ai",
		"transcript_excerpt", "transcript", "source_events", "recording_ref",
		"cost_micros", "cost_currency", "insight_summary", "mos", "content_hash",
	} {
		if _, ok := keys[want]; !ok {
			t.Errorf("embedded layout lost key %q", want)
		}
	}

	// The embed must not have INVENTED keys either.
	if len(keys) != 18 {
		t.Errorf("embedded layout emitted %d keys, want exactly the pre-embed 18", len(keys))
	}

	// The enrich invariant: the embedded observation hashes to the stored
	// hash (the decider addresses call.enrich through this snapshot).
	if fresh.Observation.ContentHash() != fresh.ContentHash {
		t.Fatalf("embedded observation hash %q != stored hash %q",
			fresh.Observation.ContentHash(), fresh.ContentHash)
	}
}
