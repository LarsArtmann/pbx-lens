package app_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/systemscenario/v4"
	"github.com/larsartmann/pbx-lens/internal/app"
	"github.com/larsartmann/pbx-lens/internal/calls"
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
