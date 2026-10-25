package calls

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	scenario "github.com/larsartmann/go-cqrs-lite/scenario/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// The scenario/v4 decider matrix: every decide path of the call aggregate,
// driven through the harness DSL instead of hand-rolled decide/fold
// bookkeeping. These specs supersede the hand-rolled decide cases that used
// to live in domain_test.go and commands_test.go; the pure-function asserts
// (validation, hashing, stream-ID shape) stay there.
//
// Domain decision recorded 2026-10-25: enriching a RETIRED call now rejects
// (call.enrich.after_retired). Before the guard, a late enrichment re-created
// the tombstoned view row and defeated the purge; the fold's "tombstone is
// last" invariant only holds when nothing appends after a retirement.

var (
	errEnrichBeforeObserved = errorfamily.NewRejection("call.enrich.before_observed", "target")
	errEnrichAfterRetired   = errorfamily.NewRejection("call.enrich.after_retired", "target")
	errRetireBeforeObserved = errorfamily.NewRejection("call.retire.before_observed", "target")
)

// reasonRetentionTest mirrors retention.ReasonRetention without importing the
// retention package (it imports calls; a test-level cycle would result).
const reasonRetentionTest = RetentionReason("retention:age")

// seedEvent mints a correctly-versioned history event for Given clauses.
func seedEvent(t *testing.T, typ event.Type, sid id.StreamID, version event.Version, payload any) event.Event {
	t.Helper()

	evt, err := event.New(typ, sid, StreamType, version, payload, event.WithSchemaVersion(1))
	if err != nil {
		t.Fatal(err)
	}

	return evt
}

// recordDecide adapts the version-first Record decide to the scenario DSL's
// command-first shape (the version is irrelevant to the events produced).
func recordDecide(sid id.StreamID) scenario.DecideFunc[RecordCallCmd, CallState] {
	return func(state CallState, cmd RecordCallCmd) ([]event.Event, error) {
		return Record(cmd.Observation, sid)(state, 0)
	}
}

func enrichDecide(sid id.StreamID, enr Enrichment) scenario.DecideFunc[EnrichCallCmd, CallState] {
	return func(state CallState, cmd EnrichCallCmd) ([]event.Event, error) {
		return Enrich(cmd.Observation, enr, sid)(state, 0)
	}
}

func retireDecide(sid id.StreamID, reason RetentionReason) scenario.DecideFunc[RetireCallCmd, CallState] {
	return func(state CallState, cmd RetireCallCmd) ([]event.Event, error) {
		return Retire(reason, sid)(state, 0)
	}
}

// mustRecord builds the validated record command for a scenario act.
func mustRecord(t *testing.T, obs Observation) RecordCallCmd {
	t.Helper()

	cmd, err := NewRecordCommand(obs)
	if err != nil {
		t.Fatal(err)
	}

	return cmd
}

// mustEnrich builds the validated enrich command for a scenario act.
func mustEnrich(t *testing.T, obs Observation, enr Enrichment) EnrichCallCmd {
	t.Helper()

	cmd, err := NewEnrichCommand(obs, enr)
	if err != nil {
		t.Fatal(err)
	}

	return cmd
}

// mustRetire builds the validated retire command for a scenario act.
func mustRetire(t *testing.T, obs Observation, reason RetentionReason) RetireCallCmd {
	t.Helper()

	cmd, err := NewRetireCommand(obs.Kind, obs.NativeID, reason)
	if err != nil {
		t.Fatal(err)
	}

	return cmd
}

// grownObservation returns obs with the larger snapshot the supersede matrix
// corrects to.
func grownObservation(obs Observation) Observation {
	grown := obs
	grown.DurationSecs = 99
	grown.Transcript = []TranscriptTurn{{Role: "user", Text: "one more thing"}}

	return grown
}

// TestScenarioRecordPaths: the first recording emits one call.observed; a
// byte-identical re-record replays to nothing; a grown snapshot supersedes
// via call.corrected WITH the enrichment riding along.
func TestScenarioRecordPaths(t *testing.T) {
	obs := validObservation()
	sid := mustSID(t, obs)
	enr := Enrichment{CostMicros: 4200, CostCurrency: "USD"}
	grown := grownObservation(obs)

	supersede := scenario.Given[RecordCallCmd](t, Apply, CallState{},
		seedEvent(t, EventObserved, sid, 1, CallObserved{Observation: obs}),
		seedEvent(t, EventEnriched, sid, 2, CallEnriched{Observation: obs, Enrichment: enr}),
	).When(mustRecord(t, grown), recordDecide(sid))

	supersede.Then(EventCorrected)
	supersede.ThenEvents(func(events []event.Event) {
		payload, err := event.DecodePayloadAuto[CallCorrected](events[0])
		if err != nil {
			t.Fatal(err)
		}

		if payload.Enrichment != enr {
			t.Errorf("correction must carry enrichment forward, got %+v", payload.Enrichment)
		}
	})

	replay := scenario.Given[RecordCallCmd](t, Apply, CallState{},
		seedEvent(t, EventObserved, sid, 1, CallObserved{Observation: obs}),
	).When(mustRecord(t, obs), recordDecide(sid))
	replay.Then()

	first := scenario.Given[RecordCallCmd](t, Apply, CallState{}).
		When(mustRecord(t, obs), recordDecide(sid))
	first.Then(EventObserved)
}

// TestScenarioEnrichPaths: enriching before any observation rejects; the
// first enrichment emits call.enriched; a replay of the same enrichment is a
// no-op; enriching a RETIRED call rejects (the tombstone must not resurrect).
func TestScenarioEnrichPaths(t *testing.T) {
	obs := validObservation()
	sid := mustSID(t, obs)
	enr := Enrichment{InsightSummary: "caller asked about eggs"}

	scenario.Given[EnrichCallCmd](t, Apply, CallState{}).
		When(mustEnrich(t, obs, enr), enrichDecide(sid, enr)).
		ThenError(errEnrichBeforeObserved)

	scenario.Given[EnrichCallCmd](t, Apply, CallState{},
		seedEvent(t, EventObserved, sid, 1, CallObserved{Observation: obs}),
	).When(mustEnrich(t, obs, enr), enrichDecide(sid, enr)).
		Then(EventEnriched)

	scenario.Given[EnrichCallCmd](t, Apply, CallState{},
		seedEvent(t, EventObserved, sid, 1, CallObserved{Observation: obs}),
		seedEvent(t, EventEnriched, sid, 2, CallEnriched{Observation: obs, Enrichment: enr}),
	).When(mustEnrich(t, obs, enr), enrichDecide(sid, enr)).
		Then()

	scenario.Given[EnrichCallCmd](t, Apply, CallState{},
		seedEvent(t, EventObserved, sid, 1, CallObserved{Observation: obs}),
		seedEvent(t, EventRetired, sid, 2, CallRetired{Reason: string(reasonRetentionTest)}),
	).When(mustEnrich(t, obs, enr), enrichDecide(sid, enr)).
		ThenError(errEnrichAfterRetired)
}

// TestScenarioRetirePaths: retiring an unobserved stream rejects; the first
// retirement emits call.retired; re-retiring is an idempotent no-op so an
// interrupted purge can safely rerun.
func TestScenarioRetirePaths(t *testing.T) {
	obs := validObservation()
	sid := mustSID(t, obs)
	retire := mustRetire(t, obs, reasonRetentionTest)

	scenario.Given[RetireCallCmd](t, Apply, CallState{}).
		When(retire, retireDecide(sid, reasonRetentionTest)).
		ThenError(errRetireBeforeObserved)

	scenario.Given[RetireCallCmd](t, Apply, CallState{},
		seedEvent(t, EventObserved, sid, 1, CallObserved{Observation: obs}),
	).When(retire, retireDecide(sid, reasonRetentionTest)).
		Then(EventRetired)

	scenario.Given[RetireCallCmd](t, Apply, CallState{},
		seedEvent(t, EventObserved, sid, 1, CallObserved{Observation: obs}),
		seedEvent(t, EventRetired, sid, 2, CallRetired{Reason: string(reasonRetentionTest)}),
	).When(retire, retireDecide(sid, reasonRetentionTest)).
		Then()
}
