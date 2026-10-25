package app_test

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/projectionadapter/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
	"github.com/larsartmann/go-cqrs-lite/systemscenario/v4"
	"github.com/larsartmann/pbx-lens/internal/app"
	"github.com/larsartmann/pbx-lens/internal/calls"
)

// TestLookupViewServesBothKeyGenerations pins the read path's key-generation
// contract end to end: a row written by the real dispatch (the bare key),
// a row written the pre-v4.6.0 way (the marker key, seeded through
// ApplyRecord with a hand-built EventWithID — the only write path that lets
// the test choose the row key), an absent stream, and the both-exist case
// where the bare key must win. LookupView's fallback is what keeps every
// pre-2026-10-23 call reachable; docs/gotchas.md owns the incident.
func TestLookupViewServesBothKeyGenerations(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), scenarioAwaitTimeout)
	defer cancel()

	sc := systemscenario.System(
		t,
		ctx,
		app.DomainConfig(),
		systemscenario.Memory(),
		systemscenario.WithAwaitTimeout(scenarioAwaitTimeout),
	)

	store := sc.System().MetaEngine()
	reader := metaengine.NewReader[calls.CallView](store, calls.Collection)

	// Bare row: the real dispatch writes at RecordKey.
	live := sampleObservation()

	liveCmd, err := calls.NewRecordCommand(live)
	if err != nil {
		t.Fatal(err)
	}

	sc.When(liveCmd).
		ThenSuccess().
		ThenQueryFunc(viewReader(sc.System(), calls.RecordKey(liveCmd.StreamID())), func(got any) error {
			return checkProjectedHash(got, live)
		})

	// Legacy row: seeded through ApplyRecord with an EventWithID whose ID is
	// the pre-v4.6.0 marker form — no live write path produces this key.
	legacy := sampleObservation()
	legacy.NativeID = "conv-legacy-key"
	legacy.DurationSecs = 321

	legacySid, err := legacy.StreamID()
	if err != nil {
		t.Fatal(err)
	}

	err = store.ApplyRecord(ctx, record.Record{
		Type:       string(calls.EventObserved),
		StreamID:   record.NewStreamRef(calls.StreamType, legacySid.Get()),
		StreamType: calls.StreamType,
		Version:    1,
	}, projectionadapter.EventWithID[calls.CallObserved]{
		ID:         calls.LegacyRecordKey(legacySid),
		Payload:    calls.CallObserved{Observation: legacy},
		OccurredAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("seed legacy row: %v", err)
	}

	// Absent sid: never recorded anywhere.
	ghost := sampleObservation()
	ghost.NativeID = "conv-never-recorded"

	ghostSid, err := ghost.StreamID()
	if err != nil {
		t.Fatal(err)
	}

	// 1. Bare row found through the canonical key.
	view, found, err := calls.LookupView(ctx, reader, liveCmd.StreamID())
	if err != nil || !found {
		t.Fatalf("LookupView(bare row) = found %v err %v, want found", found, err)
	}

	if view.ContentHash != live.ContentHash() {
		t.Fatalf("bare row ContentHash = %q, want the recorded snapshot's", view.ContentHash)
	}

	// 2. Legacy row reachable ONLY through the fallback: the bare key misses,
	// the marker key hits, LookupView returns it.
	if _, found, err := reader.Get(ctx, calls.RecordKey(legacySid)); err != nil || found {
		t.Fatalf("legacy row visible at the bare key (found %v err %v): key generations bled", found, err)
	}

	view, found, err = calls.LookupView(ctx, reader, legacySid)
	if err != nil || !found {
		t.Fatalf("LookupView(legacy row) = found %v err %v, want found via fallback", found, err)
	}

	if view.DurationSecs != 321 {
		t.Fatalf("legacy row DurationSecs = %d, want the seeded 321", view.DurationSecs)
	}

	// 3. Absent stream: not-found is clean, no error.
	if _, found, err := calls.LookupView(ctx, reader, ghostSid); found || err != nil {
		t.Fatalf("LookupView(absent) = found %v err %v, want clean not-found", found, err)
	}

	// 4. Both generations present: the bare key wins. Seed a decoy at the
	// marker key of the LIVE stream with a tell-tale duration; LookupView
	// must keep answering with the bare row.
	decoy := live
	decoy.DurationSecs = 999

	err = store.ApplyRecord(ctx, record.Record{
		Type:       string(calls.EventObserved),
		StreamID:   record.NewStreamRef(calls.StreamType, liveCmd.StreamID().Get()),
		StreamType: calls.StreamType,
		Version:    1,
	}, projectionadapter.EventWithID[calls.CallObserved]{
		ID:         calls.LegacyRecordKey(liveCmd.StreamID()),
		Payload:    calls.CallObserved{Observation: decoy},
		OccurredAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("seed decoy row: %v", err)
	}

	view, found, err = calls.LookupView(ctx, reader, liveCmd.StreamID())
	if err != nil || !found {
		t.Fatalf("LookupView(both rows) = found %v err %v, want found", found, err)
	}

	if view.DurationSecs == 999 {
		t.Fatal("LookupView answered the legacy decoy; the bare key must win")
	}

	if view.ContentHash != live.ContentHash() {
		t.Fatalf("LookupView(both rows) ContentHash = %q, want the bare row's", view.ContentHash)
	}
}
