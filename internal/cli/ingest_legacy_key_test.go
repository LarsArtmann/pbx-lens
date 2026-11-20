package cli

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/projectionadapter/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
	"github.com/larsartmann/pbx-lens/internal/app"
	"github.com/larsartmann/pbx-lens/internal/calls"
	"github.com/larsartmann/pbx-lens/internal/ingest/cdr"
)

// TestIngestClassifiesCorrectionOnLegacyKeyedRow pins the correction
// classifier's read path: a stream whose stored view row still carries the
// pre-2026-10-23 marker key (no bare-key row exists) must still be recognized
// as recorded, so a changed snapshot classifies as Corrected — not Recorded.
// Before the LookupView routing, the bare-key read missed and every legacy
// row's correction misclassified. The count is operator display, never
// correctness (Ingest's own doc comment), but the display must not lie about
// history either.
func TestIngestClassifiesCorrectionOnLegacyKeyedRow(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	sys, err := app.New(ctx, "")
	if err != nil {
		t.Fatalf("boot memory system: %v", err)
	}

	res, err := cdr.Parse(strings.NewReader(strings.Join([]string{
		`"John Doe","17287289311","2000","public","2026-10-09 19:51:25",`,
		`"2026-10-09 19:51:26","2026-10-09 19:51:51","26","25",`,
		`"NORMAL_CLEARING","conv-legacy-correct","b0a9-2222","","4.4"`,
		"\n",
	}, "")))
	if err != nil || len(res.Observations) != 1 {
		t.Fatalf("parse fixture row: err %v observations %d", err, len(res.Observations))
	}

	obs := res.Observations[0]

	sid, err := obs.StreamID()
	if err != nil {
		t.Fatal(err)
	}

	if err := sys.MetaEngine().ApplyRecord(ctx, record.Record{
		Type:       string(calls.EventObserved),
		StreamID:   record.NewStreamRef(calls.StreamType, sid.Get()),
		StreamType: calls.StreamType,
		Version:    1,
	}, projectionadapter.EventWithID[calls.CallObserved]{
		ID:         calls.LegacyRecordKey(sid),
		Payload:    calls.CallObserved{Observation: obs},
		OccurredAt: time.Now(),
	}); err != nil {
		t.Fatalf("seed legacy-keyed row: %v", err)
	}

	changed := obs
	changed.MOS = 1.1

	stats := Ingest(ctx, sys, []calls.Observation{changed})

	if stats.Total != 1 || stats.Recorded != 0 || stats.Corrected != 1 || len(stats.Rejected) != 0 {
		t.Fatalf("Ingest on a legacy-keyed row = %+v, want exactly one Corrected", stats)
	}
}
