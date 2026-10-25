package app_test

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
	"github.com/larsartmann/go-cqrs-lite/systemscenario/v4"
	"github.com/larsartmann/pbx-lens/internal/app"
	"github.com/larsartmann/pbx-lens/internal/calls"
	"github.com/larsartmann/pbx-lens/internal/retention"
)

// TestMemoryAndSqliteDeploymentsStayEquivalent is the deployment-swap proof
// (plan P19.1): two systems over the same DomainConfig, one on the memory
// engine and one on production sqlite, fed the identical lifecycle — record,
// correction, enrichment, retire. The journals must be observationally
// equivalent (same per-stream type/version trails) and the read models must
// agree, because app.New's dsn knob is the ONLY deployment difference and
// nothing else may leak into the record or its projection.
func TestMemoryAndSqliteDeploymentsStayEquivalent(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	boot := func(deploy system.DeploymentConfig) *systemscenario.Scenario {
		t.Helper()

		return systemscenario.System(
			t,
			ctx,
			app.DomainConfig(),
			deploy,
			systemscenario.WithAwaitTimeout(scenarioAwaitTimeout),
		)
	}

	memory := boot(systemscenario.Memory())
	sqlite := boot(app.DeploymentConfig(filepath.Join(t.TempDir(), "equivalence.db")))

	// driveLifecycle feeds both deployments the identical four-op lifecycle
	// and returns the (deterministic) stream id, for the cross-engine view
	// comparison below.
	driveLifecycle := func(sc *systemscenario.Scenario) id.StreamID {
		t.Helper()

		obs := sampleObservation()

		recordCmd, err := calls.NewRecordCommand(obs)
		if err != nil {
			t.Fatal(err)
		}

		sc.When(recordCmd).
			ThenSuccess().
			ThenQueryFunc(viewReader(sc.System(), calls.RecordKey(recordCmd.StreamID())), func(got any) error {
				return checkProjectedHash(got, obs)
			})

		obs.DurationSecs += 30

		correctCmd, err := calls.NewRecordCommand(obs)
		if err != nil {
			t.Fatal(err)
		}

		sc.When(correctCmd).
			ThenSuccess().
			ThenQueryFunc(viewReader(sc.System(), calls.RecordKey(correctCmd.StreamID())), func(got any) error {
				return checkProjectedHash(got, obs)
			})

		enr := calls.Enrichment{CostMicros: 4200, CostCurrency: "USD", InsightSummary: "swap-proof enrichment"}

		enrichCmd, err := calls.NewEnrichCommand(obs, enr)
		if err != nil {
			t.Fatal(err)
		}

		sc.When(enrichCmd).
			ThenSuccess().
			ThenQueryFunc(viewReader(sc.System(), calls.RecordKey(enrichCmd.StreamID())), func(got any) error {
				view, ok := got.(calls.CallView)
				if !ok {
					return errViewNotACallView
				}

				if view.InsightSummary != enr.InsightSummary {
					return errProjectedEnrichmentStale
				}

				return nil
			})

		retireCmd, err := calls.NewRetireCommand(obs.Kind, obs.NativeID, retention.ReasonRetention)
		if err != nil {
			t.Fatal(err)
		}

		sc.When(retireCmd).
			ThenSuccess().
			ThenQueryFails(viewReader(sc.System(), calls.RecordKey(retireCmd.StreamID())), errViewNotFound)

		return recordCmd.StreamID()
	}

	sid := driveLifecycle(memory)

	if sqliteSid := driveLifecycle(sqlite); sqliteSid.Get() != sid.Get() {
		t.Fatalf(
			"stream ids diverged: %s vs %s — the id minting must be deterministic for the swap proof",
			sid.Get(),
			sqliteSid.Get(),
		)
	}

	systemscenario.AssertJournalEquivalence(t, ctx, memory.System(), sqlite.System())

	// Read-model parity on the retired stream: absent from BOTH engines (the
	// live-stream parity is pinned per-act above; this is the tombstone's
	// cross-engine half).
	lookup := func(sc *systemscenario.Scenario) (calls.CallView, bool) {
		t.Helper()

		view, found, err := calls.LookupView(
			ctx,
			metaengine.NewReader[calls.CallView](sc.System().MetaEngine(), calls.Collection),
			sid,
		)
		if err != nil {
			t.Fatal(err)
		}

		return view, found
	}

	memoryView, memoryFound := lookup(memory)
	sqliteView, sqliteFound := lookup(sqlite)

	if memoryFound || sqliteFound {
		t.Fatalf("retired stream still projected (memory %v, sqlite %v)", memoryFound, sqliteFound)
	}

	if !reflect.DeepEqual(memoryView, sqliteView) {
		t.Fatalf("zero views diverged across engines\nmemory: %#v\nsqlite: %#v", memoryView, sqliteView)
	}
}
