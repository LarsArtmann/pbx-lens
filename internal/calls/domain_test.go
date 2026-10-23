package calls

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/decider/v4"
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

func TestContentHashMOSOnlyWhenReported(t *testing.T) {
	obs := validObservation()

	// An unreported MOS contributes zero bytes: the pre-image is exactly the
	// named fields below, with no mos entry. That is the contract which lets a
	// source that never carries a score stop at version-level identity instead
	// of pretending to be a scored call. (The absolute digest is versioned —
	// see ObservationSchemaVersion — so this pins the FIELD SET, not the bytes.)
	h := sha256.New()
	writeHashedField(h, "schema", strconv.Itoa(ObservationSchemaVersion))
	writeHashedField(h, "native_id", string(obs.NativeID))
	writeHashedField(h, "kind", string(obs.Kind))
	writeHashedField(h, "direction", string(obs.Direction))
	writeHashedField(h, "started_at_nanos", strconv.FormatInt(obs.StartedAt.UTC().UnixNano(), 10))
	writeHashedField(h, "duration_secs", strconv.Itoa(obs.DurationSecs))
	writeHashedField(h, "from_number", obs.FromNumber)
	writeHashedField(h, "to_number", obs.ToNumber)
	writeHashedField(h, "hangup_cause", obs.HangupCause)
	writeHashedField(h, "is_ai", strconv.FormatBool(obs.IsAI))
	writeHashedField(h, "transcript_excerpt", obs.TranscriptExcerpt)
	writeHashedField(h, "transcript", transcriptHash(obs.Transcript))
	writeHashedField(h, "recording_ref", obs.RecordingRef)

	if got := obs.ContentHash(); got != hex.EncodeToString(h.Sum(nil)) {
		t.Error("an unreported MOS must contribute no bytes to the pre-image")
	}

	scored := obs

	scored.MOS = 4.2
	if scored.ContentHash() == obs.ContentHash() {
		t.Error("a reported MOS must change the content hash")
	}

	rescored := obs

	rescored.MOS = 3.1
	if rescored.ContentHash() == scored.ContentHash() {
		t.Error("different MOS values must hash differently")
	}
}

func TestContentHashSchemaVersionMovesTheHash(t *testing.T) {
	obs := validObservation()

	if obs.contentHash(1) == obs.contentHash(2) {
		t.Fatal("a schema bump must produce a different hash — otherwise an additive field never reaches history")
	}

	if obs.ContentHash() != obs.contentHash(ObservationSchemaVersion) {
		t.Fatal("ContentHash must be the pre-image of the shipped schema version")
	}
}

func TestContentHashSeparatesFieldBoundaries(t *testing.T) {
	// Under the retired newline-joined encoding these two observations could
	// collide: the same concatenated bytes, different field values. The
	// length-prefixed pre-image must keep them apart.
	left := validObservation()
	left.TranscriptExcerpt = "ab"
	left.FromNumber = "+1728728931"

	right := validObservation()
	right.TranscriptExcerpt = "a"
	right.FromNumber = "+17287289311"

	if left.ContentHash() == right.ContentHash() {
		t.Fatal("shifting a byte across a field boundary must not keep the identity")
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

	state := foldAll(t, decide(t, Record(obs, sid), CallState{}, 0))

	state = foldAll(t, decide(t, Enrich(obs, enr, sid), state, 1))
	if state.CurrentEnrichment != enr {
		t.Fatalf("enriched state must hold enrichment: %+v", state.CurrentEnrichment)
	}

	grown := obs
	grown.DurationSecs = 120

	corrections := decide(t, Record(grown, sid), state, 2)
	if len(corrections) != 1 {
		t.Fatalf("correction after enrichment: %d events", len(corrections))
	}

	payload, err := event.DecodePayloadAuto[CallCorrected](corrections[0])
	if err != nil {
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

	state := foldAll(t, decide(t, Record(obs, sid), CallState{}, 0))

	events := decide(t, Enrich(obs, enr, sid), state, 1)
	if len(events) != 1 || events[0].Type() != EventEnriched {
		t.Fatalf("enrich: %d events", len(events))
	}

	enriched := foldAll(t, events)
	if enriched.EnrichHash != enr.Hash() || enriched.CurrentEnrichment != enr {
		t.Fatalf("enriched state wrong: %+v", enriched)
	}

	replay := decide(t, Enrich(obs, enr, sid), enriched, 2)
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

func decide(
	t *testing.T,
	fn decider.DecideFunc[CallState],
	s CallState,
	v event.Version,
) []event.Event {
	t.Helper()

	events, err := fn(s, v)
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
