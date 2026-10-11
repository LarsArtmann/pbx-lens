package calls

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func validObservation() Observation {
	return Observation{
		NativeID:          "conv-1234",
		Kind:              SourceTelnyxConversation,
		Direction:         DirectionOutbound,
		StartedAt:         time.Date(2026, 10, 9, 19, 51, 25, 0, time.UTC),
		DurationSecs:      26,
		FromNumber:        "+17287289311",
		ToNumber:          "+14084757593",
		HangupCause:       "normal_clearing",
		IsAI:              true,
		TranscriptExcerpt: "The Bear Union does not negotiate",
		RecordingRef:      "c8daa4ef-020c-49c5-8a35-888b41e33900",
	}
}

func TestValidate(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*Observation)
		wantErr string
	}{
		{"valid", func(*Observation) {}, ""},
		{"empty native id", func(o *Observation) { o.NativeID = "" }, "call.record.empty_native_id"},
		{"unknown kind", func(o *Observation) { o.Kind = "carrier-pigeon" }, "call.record.unknown_source_kind"},
		{"unknown direction", func(o *Observation) { o.Direction = "sideways" }, "call.record.unknown_direction"},
		{"zero started at", func(o *Observation) { o.StartedAt = time.Time{} }, "call.record.zero_started_at"},
		{"negative duration", func(o *Observation) { o.DurationSecs = -1 }, "call.record.negative_duration"},
		{
			"both parties missing",
			func(o *Observation) { o.FromNumber = ""; o.ToNumber = "" },
			"call.record.both_parties_missing",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			obs := validObservation()
			tc.mutate(&obs)

			err := obs.Validate()
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("want valid, got %v", err)
				}

				return
			}

			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("want rejection %q, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestStreamIDShape(t *testing.T) {
	obs := validObservation()

	sid, err := obs.StreamID()
	if err != nil {
		t.Fatal(err)
	}

	if got := sid.String(); !strings.Contains(got, "telnyx-conversation:conv-1234") {
		t.Fatalf("stream id = %q, want it to carry kind:native", got)
	}

	if back, err := id.ParseStreamID(sid.String()); err != nil || back != sid {
		t.Fatalf("round-trip law broken: %v -> %v (%v)", sid, back, err)
	}
}

func TestRecordIdempotentReplay(t *testing.T) {
	obs := validObservation()
	sid, _ := obs.StreamID()

	events, err := Record(obs, sid)(CallState{}, 0)
	if err != nil || len(events) != 1 {
		t.Fatalf("first record: %d events, err %v", len(events), err)
	}

	state := CallState{}
	for _, evt := range events {
		state, err = Apply(state, evt)
		if err != nil {
			t.Fatal(err)
		}
	}

	if !state.Observed || state.ContentHash != obs.ContentHash() {
		t.Fatalf("fold state wrong: %+v", state)
	}

	replay, err := Record(obs, sid)(state, 1)
	if err != nil {
		t.Fatalf("idempotent replay errored: %v", err)
	}

	if len(replay) != 0 {
		t.Fatalf("idempotent replay emitted %d events, want 0", len(replay))
	}
}

func TestRecordChangedObservationCorrects(t *testing.T) {
	obs := validObservation()
	sid := mustSID(t, obs)

	events, err := Record(obs, sid)(CallState{}, 0)
	if err != nil || len(events) != 1 {
		t.Fatalf("first record: %d events, err %v", len(events), err)
	}

	state := foldAll(t, events)

	grown := obs
	grown.DurationSecs = 99
	grown.Transcript = []TranscriptTurn{{Role: "user", Text: "one more thing"}}

	corrections, err := Record(grown, sid)(state, 1)
	if err != nil {
		t.Fatalf("changed observation must correct, got %v", err)
	}

	if len(corrections) != 1 || corrections[0].Type() != EventCorrected {
		t.Fatalf("want one call.corrected event, got %v", corrections)
	}

	corrected := foldAll(t, corrections)
	if corrected.ContentHash != grown.ContentHash() {
		t.Fatalf("corrected state hash = %q, want grown %q", corrected.ContentHash, grown.ContentHash())
	}

	if corrected.Current.DurationSecs != 99 {
		t.Fatalf("corrected state must hold the new snapshot: %+v", corrected.Current)
	}
}

func TestCorrectionCarriesEnrichmentForward(t *testing.T) {
	obs := validObservation()
	sid := mustSID(t, obs)
	enr := Enrichment{CostMicros: 4200, CostCurrency: "USD"}

	state := foldAll(t, mustEvents(t, Record(obs, sid)(CallState{}, 0)))
	state = foldAll(t, mustEvents(t, Enrich(obs, enr, sid)(state, 1)))
	if state.CurrentEnrichment != enr {
		t.Fatalf("enriched state must hold enrichment: %+v", state.CurrentEnrichment)
	}

	grown := obs
	grown.DurationSecs = 120

	corrections, err := Record(grown, sid)(state, 2)
	if err != nil || len(corrections) != 1 {
		t.Fatalf("correction after enrichment: %d events, err %v", len(corrections), err)
	}

	var payload CallCorrected
	if err := json.Unmarshal(mustPayload(t, corrections[0]), &payload); err != nil {
		t.Fatal(err)
	}

	if payload.Enrichment != enr {
		t.Fatalf("correction must carry enrichment forward, got %+v", payload.Enrichment)
	}
}

func TestEnrichIdempotentAndOrdered(t *testing.T) {
	obs := validObservation()
	sid := mustSID(t, obs)
	enr := Enrichment{InsightSummary: "caller asked about eggs"}

	if _, err := Enrich(obs, enr, sid)(CallState{}, 0); err == nil {
		t.Fatal("enrich before observed must reject")
	}

	state := foldAll(t, mustEvents(t, Record(obs, sid)(CallState{}, 0)))

	events, err := Enrich(obs, enr, sid)(state, 1)
	if err != nil || len(events) != 1 || events[0].Type() != EventEnriched {
		t.Fatalf("enrich: %d events, err %v", len(events), err)
	}

	enriched := foldAll(t, events)
	if enriched.EnrichHash != enr.Hash() || enriched.CurrentEnrichment != enr {
		t.Fatalf("enriched state wrong: %+v", enriched)
	}

	replay, err := Enrich(obs, enr, sid)(enriched, 2)
	if err != nil {
		t.Fatal(err)
	}

	if len(replay) != 0 {
		t.Fatalf("enrichment replay must emit 0 events, got %d", len(replay))
	}
}

func TestApplyRejectsForeignEvents(t *testing.T) {
	sid := mustSID(t, validObservation())

	evt, err := event.New("call.deleted", sid, StreamType, 1, struct{}{})
	if err != nil {
		t.Fatal(err)
	}

	state, err := Apply(CallState{}, evt)
	if err == nil {
		t.Fatal("foreign event must corrupt-fail, not silently ignore")
	}

	if !strings.Contains(err.Error(), "call.fold.unknown_event") {
		t.Fatalf("corruption code missing from %q", err)
	}

	if state.Observed {
		t.Fatal("foreign event must not set Observed")
	}
}

func TestE164(t *testing.T) {
	if !E164("+17287289311").Valid() {
		t.Fatal("+17287289311 must be valid")
	}

	for _, bad := range []E164{"", "17287289311", "+17287", "+abc12345"} {
		if bad.Valid() {
			t.Fatalf("%q must be invalid", bad)
		}
	}

	h1, h2 := E164("+17287289311").Hash(), E164("+17287289311").Hash()
	if h1 != h2 || len(h1) != 16 {
		t.Fatalf("hash not deterministic/16-hex: %q vs %q", h1, h2)
	}

	if h1 == E164("+14084757593").Hash() {
		t.Fatal("different numbers must hash differently")
	}
}

func mustSID(t *testing.T, obs Observation) id.StreamID {
	t.Helper()

	sid, err := obs.StreamID()
	if err != nil {
		t.Fatal(err)
	}

	return sid
}

func mustEvents(t *testing.T, events []event.Event, err error) []event.Event {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}

	return events
}

func foldAll(t *testing.T, events []event.Event) CallState {
	t.Helper()

	state := CallState{}
	for _, evt := range events {
		var err error
		state, err = Apply(state, evt)
		if err != nil {
			t.Fatal(err)
		}
	}

	return state
}

func mustPayload(t *testing.T, evt event.Event) []byte {
	t.Helper()

	var payload any = evt.Payload()
	switch p := payload.(type) {
	case []byte:
		return p
	default:
		b, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}

		return b
	}
}
