package app_test

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/systemscenario/v4"
	"github.com/larsartmann/pbx-lens/internal/app"
	"github.com/larsartmann/pbx-lens/internal/calls"
	"github.com/larsartmann/pbx-lens/internal/retention"
)

// The depth specs: the two behaviors the audit flagged as untested at the
// system level — re-ingesting an identical snapshot (the poller's steady
// state) and the retirement tombstone (the purge job's write). Both run on
// the memory preset: no durability is under test here, only decide/fold
// semantics through the real wiring.

func retireCmdFor(obs calls.Observation) (calls.RetireCallCmd, error) {
	return calls.NewRetireCommand(obs.Kind, obs.NativeID, retention.ReasonRetention)
}

// streamRefOf is the Given-seed address of a command's stream.
func streamRefOf(cmd calls.RecordCallCmd) id.StreamRef {
	return id.StreamRef{ID: cmd.StreamID(), Type: calls.StreamType}
}

// TestReIngestSameSnapshotIsANoOp pins the poller's steady state: a source
// row that re-polls byte-identical replays to ZERO new events. Without this
// spec a decide regression would silently bloat the journal by one
// call.observed per poll per call — invisible to every read surface.
func TestReIngestSameSnapshotIsANoOp(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	sc := systemscenario.System(t, ctx, app.DomainConfig(), systemscenario.Memory())

	obs := sampleObservation()
	cmd, err := calls.NewRecordCommand(obs)
	if err != nil {
		t.Fatal(err)
	}

	sc.Given(sc.Event(calls.EventObserved, streamRefOf(cmd), calls.CallObserved{Observation: obs}))

	sc.When(cmd).
		ThenSuccess().
		ThenNoEvents()
}

// TestRetireTombstonesTheViewRow pins the purge path end to end: recording
// projects a view, retiring stamps the tombstone event, and the projected
// row VANISHES once the fold catches up — while the stream's audit trail
// stays intact (the journal keeps both events; only the read surface loses
// the row).
func TestRetireTombstonesTheViewRow(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	sc := systemscenario.System(t, ctx, app.DomainConfig(), systemscenario.Memory())

	obs := sampleObservation()
	cmd, err := calls.NewRecordCommand(obs)
	if err != nil {
		t.Fatal(err)
	}

	key := calls.RecordKey(cmd.StreamID())

	sc.When(cmd).
		Then(calls.EventObserved).
		ThenQueryFunc(viewReader(sc.System(), key), func(got any) error {
			view, ok := got.(calls.CallView)
			if !ok || view.NativeID != obs.NativeID {
				t.Fatalf("projected view wrong before retire: %#v", got)
			}

			return nil
		})

	retire, err := retireCmdFor(obs)
	if err != nil {
		t.Fatal(err)
	}

	retiring := systemscenario.Adopt(t, ctx, sc.System())

	retiring.When(retire).
		Then(calls.EventRetired).
		ThenQueryEventuallyFails(viewReader(retiring.System(), key), errViewNotFound)
}
