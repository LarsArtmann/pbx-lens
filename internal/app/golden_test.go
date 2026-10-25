package app_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/systemscenario/v4"
	"github.com/larsartmann/pbx-lens/internal/app"
	"github.com/larsartmann/pbx-lens/internal/calls"
	"github.com/larsartmann/pbx-lens/internal/retention"
)

// TestGoldenRecordScenarioPayloadTrail pins the journal shape of the
// headline record scenario as a payload-hash golden: one call.observed event
// whose payload hashes stably with the stream IDs masked to ordinal tokens.
// A decide or payload-encoding regression that changes what lands in the
// journal breaks this golden in CI, not in production. Refresh with
// UPDATE_SNAPS=true go test ./internal/app/ after an INTENDED payload change.
func TestGoldenRecordScenarioPayloadTrail(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), scenarioAwaitTimeout)
	defer cancel()

	sc := systemscenario.System(
		t,
		ctx,
		app.DomainConfig(),
		systemscenario.Memory(),
		systemscenario.WithAwaitTimeout(scenarioAwaitTimeout),
	)

	obs := sampleObservation()

	cmd, err := calls.NewRecordCommand(obs)
	if err != nil {
		t.Fatal(err)
	}

	sc.When(cmd).
		Then(calls.EventObserved).
		ThenGoldenWithPayloads(t, "record_scenario_payloads")
}

// TestGoldenRetireScenarioPayloadTrail pins the journal shape of the full
// lifecycle the purge job drives: record emits call.observed, retire emits
// call.retired carrying the policy reason. Together with the record golden
// this covers both event types the live decide can emit into a fresh
// journal, so a payload-encoding regression on either breaks CI. Refresh
// with UPDATE_SNAPS=true go test ./internal/app/ after an INTENDED payload
// change.
func TestGoldenRetireScenarioPayloadTrail(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), scenarioAwaitTimeout)
	defer cancel()

	sc := systemscenario.System(
		t,
		ctx,
		app.DomainConfig(),
		systemscenario.Memory(),
		systemscenario.WithAwaitTimeout(scenarioAwaitTimeout),
	)

	obs := sampleObservation()

	recordCmd, err := calls.NewRecordCommand(obs)
	if err != nil {
		t.Fatal(err)
	}

	retireCmd, err := calls.NewRetireCommand(obs.Kind, obs.NativeID, retention.ReasonRetention)
	if err != nil {
		t.Fatal(err)
	}

	sc.When(recordCmd).
		Command(retireCmd).
		Then(calls.EventObserved, calls.EventRetired).
		ThenGoldenWithPayloads(t, "retire_scenario_payloads")
}
