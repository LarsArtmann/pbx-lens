package app_test

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
	"github.com/larsartmann/go-cqrs-lite/systemscenario/v4"
	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/larsartmann/pbx-lens/internal/app"
	"github.com/larsartmann/pbx-lens/internal/calls"
	"github.com/larsartmann/pbx-lens/internal/retention"
	"pgregory.net/rapid"
)

// The depth trio: optimistic concurrency, engine latency chaos, and a
// property oracle. All three boot the production wiring (ADR-0123 holds:
// system.New is the only composition root); the OCC winner/loser writes use
// the store's Save gate directly — the same gate every command dispatch
// relies on — because a deterministic loser needs two writers aiming at one
// version slot, which a single-threaded dispatch cannot produce.

// TestOptimisticConcurrencyRejectsStaleWriter proves the losing side of an
// optimistic-concurrency race: a writer that loaded before the current head
// is rejected with a Conflict-family error naming event.ErrVersionConflict,
// its event is NOT persisted, and the journal keeps exactly the two
// legitimate events.
func TestOptimisticConcurrencyRejectsStaleWriter(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
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

	ref := streamRefOf(cmd)

	sc.Given(sc.Event(calls.EventObserved, ref, calls.CallObserved{Observation: obs}))

	winner := sc.Event(calls.EventRetired, ref, calls.CallRetired{Reason: "retention:age"})

	store := sc.System().EventStore()

	// The winning writer: loaded at head v1, saves at expected v1. Lands.
	if err := store.Save(ctx, ref, []event.Event{winner}, 1); err != nil {
		t.Fatalf("winning save rejected: %v", err)
	}

	// The losing writer: SAME expected version (it loaded before the winner
	// landed). Must be rejected as a version conflict in the Conflict family.
	loser := sc.Event(calls.EventRetired, ref, calls.CallRetired{Reason: "retention:age-loser"})

	err = store.Save(ctx, ref, []event.Event{loser}, 1)
	if err == nil {
		t.Fatal("stale save accepted: optimistic concurrency gate did not fire")
	}

	if !errors.Is(err, event.ErrVersionConflict) {
		t.Fatalf("stale save error = %v, want errors.Is event.ErrVersionConflict", err)
	}

	if got := errorfamily.Classify(err); got != errorfamily.Conflict {
		t.Fatalf("stale save error family = %v, want Conflict", got)
	}

	// Terminal scenario assertion: re-recording the unchanged snapshot is a
	// no-op (zero act events), then the journal-shape check at t level — the
	// journal holds exactly the two legitimate events; the loser's never
	// landed.
	sc.When(cmd).
		ThenSuccess().
		ThenNoEvents()

	events := journalOf(t, sc)
	if len(events) != 2 {
		t.Fatalf("journal holds %d events, want 2 (seed + winner)", len(events))
	}

	if events[1].Type() != calls.EventRetired || events[1].Version() != 2 {
		t.Fatalf("second event = %s v%d, want the winner's call.retired v2", events[1].Type(), events[1].Version())
	}
}

// primaryEngineName mirrors app's internal engine key; the deployment map is
// keyed by it, so the chaos driver swaps exactly the primary engine.
const primaryEngineName = "primary"

// TestDelayedSQLiteKeepsIdenticalSavesSerialized widens the race window with
// a 2ms-latency sqlite driver and fires ten identical record dispatches at
// one stream concurrently. Optimistic concurrency decides the winner;
// exactly one event may land, the view must exist and match, and every
// loser must leave the journal untouched.
func TestDelayedSQLiteKeepsIdenticalSavesSerialized(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	dsn := filepath.Join(t.TempDir(), "chaos.db")
	deploy := app.DeploymentConfig(dsn)
	deploy.Engines[primaryEngineName] = system.EngineConfig{
		Driver: systemscenario.DelayedDriver(t, "sqlite", 2*time.Millisecond),
		DSN:    dsn,
	}

	sc := systemscenario.System(
		t,
		ctx,
		app.DomainConfig(),
		deploy,
		systemscenario.WithAwaitTimeout(scenarioAwaitTimeout),
	)

	obs := sampleObservation()

	const racers = 10

	var (
		errs [racers]error
		wg   sync.WaitGroup
	)

	for i := range racers {
		wg.Go(func() {
			cmd, err := calls.NewRecordCommand(obs)
			if err != nil {
				errs[i] = err

				return
			}

			errs[i] = sc.System().CommandDispatcher().Dispatch(ctx, cmd)
		})
	}

	wg.Wait()

	won, rejected := 0, 0

	for _, err := range errs {
		switch {
		case err == nil:
			won++
		case errors.Is(err, event.ErrVersionConflict):
			rejected++
		default:
			t.Fatalf("unexpected dispatch error: %v", err)
		}
	}

	if won != 1 || rejected != racers-1 {
		t.Fatalf("race outcome: %d winners, %d rejected, want 1/%d", won, rejected, racers-1)
	}

	cmd, err := calls.NewRecordCommand(obs)
	if err != nil {
		t.Fatal(err)
	}

	sc.When(cmd).
		ThenSuccess().
		ThenNoEvents().
		ThenQueryFunc(viewReader(sc.System(), calls.RecordKey(cmd.StreamID())), func(got any) error {
			return checkProjectedHash(got, obs)
		})
}

// obsGenerator produces the random observations for the property oracle:
// three streams' worth of native IDs, both directions, and varying
// durations/causes — enough shape for decide/fold divergence to show.
func obsGenerator(rt *rapid.T) calls.Observation {
	nativeID := calls.NativeID(rapid.SampledFrom([]string{"conv-p1", "conv-p2", "conv-p3"}).Draw(rt, "nativeID"))
	direction := rapid.SampledFrom([]calls.Direction{calls.DirectionInbound, calls.DirectionOutbound}).
		Draw(rt, "direction")
	duration := rapid.IntRange(0, 3600).Draw(rt, "duration")
	cause := rapid.SampledFrom([]string{"normal_clearing", "no_answer"}).Draw(rt, "cause")

	return calls.Observation{
		NativeID:          nativeID,
		Kind:              calls.SourceTelnyxConversation,
		Direction:         direction,
		StartedAt:         time.Date(2026, 10, 9, 19, 51, 25, 0, time.UTC),
		DurationSecs:      duration,
		FromNumber:        "+17287289311",
		ToNumber:          "+14084757593",
		HangupCause:       cause,
		TranscriptExcerpt: "property oracle row",
	}
}

// TestPropertyProjectedRowEqualsPureFold drives random record sequences
// through the real wiring and, after each sequence, asserts the read-model
// row equals a pure in-process fold of the same journal, and that every
// stream's journal versions stay strictly ordered from 1.
func TestPropertyProjectedRowEqualsPureFold(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		sc := systemscenario.System(
			t,
			ctx,
			app.DomainConfig(),
			systemscenario.Memory(),
			systemscenario.WithAwaitTimeout(scenarioAwaitTimeout),
		)

		driveRandomRecords(rt, sc)
		assertFoldParity(rt, ctx, sc)
		assertJournalVersionOrdering(rt, sc)
	})
}

// TestPropertyLifecycleProjectedRowEqualsPureFold widens the oracle from
// record-only trails to full lifecycles: record, correction (a grown source
// snapshot), enrichment, correction-after-enrichment (the enrichment must
// ride along), a second enrichment, and an optional retire. Every op is
// forced to change state — strictly growing durations for snapshots, a
// distinct enrichment per op — because a no-op op would silently shrink the
// oracle's coverage; each act converges its own row before the next draw,
// and the whole journal faces the pure fold and the version-ordering check.
func TestPropertyLifecycleProjectedRowEqualsPureFold(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		sc := systemscenario.System(
			t,
			ctx,
			app.DomainConfig(),
			systemscenario.Memory(),
			systemscenario.WithAwaitTimeout(scenarioAwaitTimeout),
		)

		driveRandomLifecycles(rt, sc)
		assertFoldParity(rt, ctx, sc)
		assertJournalVersionOrdering(rt, sc)
	})
}

// driveRandomLifecycles runs a full lifecycle on each of three rapid-drawn
// streams, converging every act's own row before the next op: ThenQuery
// polls until the projection caught up, so the oracle never races the async
// fold.
func driveRandomLifecycles(rt *rapid.T, sc *systemscenario.Scenario) {
	for streamIdx := range 3 {
		obs := obsGenerator(rt)
		// A lifecycle owns a stream exclusively: obsGenerator's shared
		// three-ID pool could deal a retired stream into a later iteration,
		// whose record would then (correctly) reject — disjoint IDs keep the
		// oracle about lifecycle shape, not about stream collisions.
		obs.NativeID = calls.NativeID(fmt.Sprintf("conv-life-%d", streamIdx))

		recordCmd, err := calls.NewRecordCommand(obs)
		if err != nil {
			rt.Fatalf("NewRecordCommand: %v", err)
		}

		sc.When(recordCmd).
			ThenSuccess().
			ThenQueryFunc(viewReader(sc.System(), calls.RecordKey(recordCmd.StreamID())), func(got any) error {
				return checkProjectedHash(got, obs)
			})

		driveLifecycleTail(rt, sc, streamIdx, obs)
	}
}

// driveLifecycleTail runs the post-record lifecycle on one stream: a
// correction (grown snapshot), an enrichment, a correction-after-enrichment
// (the enrichment must ride along), an optional second enrichment, and an
// optional retire — every act converging its own row before the next op.
func driveLifecycleTail(rt *rapid.T, sc *systemscenario.Scenario, streamIdx int, obs calls.Observation) {
	grow := func(op string) {
		obs.DurationSecs += 1 + rapid.IntRange(0, 3600).Draw(rt, op+" duration")
		obs.TranscriptExcerpt = "property oracle row after " + op
	}

	check := func(cmd command.Command, want calls.Observation) {
		sc.When(cmd).
			ThenSuccess().
			ThenQueryFunc(viewReader(sc.System(), calls.RecordKey(cmd.StreamID())), func(got any) error {
				return checkProjectedHash(got, want)
			})
	}

	grow("first correction")

	correctCmd, err := calls.NewRecordCommand(obs)
	if err != nil {
		rt.Fatalf("NewRecordCommand (correct): %v", err)
	}

	check(correctCmd, obs)

	enr := calls.Enrichment{
		CostMicros:     int64(1 + rapid.IntRange(0, 10_000_000).Draw(rt, "cost micros")),
		CostCurrency:   rapid.SampledFrom([]string{"USD", "EUR"}).Draw(rt, "currency"),
		InsightSummary: fmt.Sprintf("stream %d asked about eggs", streamIdx),
	}

	enrichCmd, err := calls.NewEnrichCommand(obs, enr)
	if err != nil {
		rt.Fatalf("NewEnrichCommand: %v", err)
	}

	sc.When(enrichCmd).
		ThenSuccess().
		ThenQueryFunc(viewReader(sc.System(), calls.RecordKey(enrichCmd.StreamID())), enrichmentChecker(enr))

	grow("second correction")

	correctAfterEnrichCmd, err := calls.NewRecordCommand(obs)
	if err != nil {
		rt.Fatalf("NewRecordCommand (correct after enrich): %v", err)
	}

	sc.When(correctAfterEnrichCmd).
		ThenSuccess().
		ThenQueryFunc(
			viewReader(sc.System(), calls.RecordKey(correctAfterEnrichCmd.StreamID())),
			func(got any) error {
				if err := checkProjectedHash(got, obs); err != nil {
					return err
				}

				return enrichmentChecker(enr)(got)
			},
		)

	if rapid.Bool().Draw(rt, "re-enrich") {
		enr.InsightSummary += " (revised)"

		reEnrichCmd, err := calls.NewEnrichCommand(obs, enr)
		if err != nil {
			rt.Fatalf("NewEnrichCommand (revised): %v", err)
		}

		sc.When(reEnrichCmd).
			ThenSuccess().
			ThenQueryFunc(viewReader(sc.System(), calls.RecordKey(reEnrichCmd.StreamID())), enrichmentChecker(enr))
	}

	if rapid.Bool().Draw(rt, "retire") {
		retireCmd, err := calls.NewRetireCommand(obs.Kind, obs.NativeID, retention.ReasonRetention)
		if err != nil {
			rt.Fatalf("NewRetireCommand: %v", err)
		}

		sc.When(retireCmd).
			ThenSuccess().
			ThenQueryFails(viewReader(sc.System(), calls.RecordKey(retireCmd.StreamID())), errViewNotFound)
	}
}

// enrichmentChecker accepts a converged row only when its enrichment fields
// match the dispatched enrichment.
func enrichmentChecker(want calls.Enrichment) func(got any) error {
	return func(got any) error {
		view, ok := got.(calls.CallView)
		if !ok {
			return fmt.Errorf("%w: %T", errViewNotACallView, got)
		}

		if view.InsightSummary != want.InsightSummary || view.CostMicros != want.CostMicros {
			return fmt.Errorf(
				"%w: projected insight %q/%d, want %q/%d",
				errProjectedEnrichmentStale,
				view.InsightSummary,
				view.CostMicros,
				want.InsightSummary,
				want.CostMicros,
			)
		}

		return nil
	}
}

// driveRandomRecords records six rapid-generated snapshots, converging each
// act's own stream before the next draw: ThenQuery polls until the projected
// row carries the snapshot's hash, so the oracle never races the async fold.
func driveRandomRecords(rt *rapid.T, sc *systemscenario.Scenario) {
	const ops = 6

	for range ops {
		obs := obsGenerator(rt)

		cmd, err := calls.NewRecordCommand(obs)
		if err != nil {
			rt.Fatalf("NewRecordCommand: %v", err)
		}

		sc.When(cmd).
			ThenSuccess().
			ThenQueryFunc(viewReader(sc.System(), calls.RecordKey(cmd.StreamID())), func(got any) error {
				return checkProjectedHash(got, obs)
			})
	}
}

// errViewNotACallView marks a view query that returned something other than
// a CallView (impossible by construction; guards reader API drift).
var errViewNotACallView = errors.New("view query returned a non-CallView result")

// errProjectedHashStale marks a projected row still lagging its snapshot.
var errProjectedHashStale = errors.New("projected hash does not match the snapshot")

// errProjectedEnrichmentStale marks a projected row whose enrichment fields
// still lag the dispatched enrichment.
var errProjectedEnrichmentStale = errors.New("projected enrichment does not match the enrichment")

// checkProjectedHash accepts a converged row only when it carries the
// recorded snapshot's content hash.
func checkProjectedHash(got any, obs calls.Observation) error {
	view, ok := got.(calls.CallView)
	if !ok {
		return fmt.Errorf("%w: %T", errViewNotACallView, got)
	}

	if view.ContentHash != obs.ContentHash() {
		return fmt.Errorf("%w: projected %q, want %q", errProjectedHashStale, view.ContentHash, obs.ContentHash())
	}

	return nil
}

// pureFold replays the whole journal in-process through calls.Apply — the
// specification the projection's async fold must equal.
func pureFold(sc *systemscenario.Scenario) (map[string]foldEntry, error) {
	journal, err := journalEvents(sc)
	if err != nil {
		return nil, err
	}

	folded := map[string]foldEntry{}

	for _, evt := range journal {
		entry := folded[evt.StreamID().String()]
		entry.sid = evt.StreamID()

		next, err := calls.Apply(entry.state, evt)
		if err != nil {
			return nil, fmt.Errorf("fold %s: %w", evt.StreamID(), err)
		}

		entry.state = next
		folded[evt.StreamID().String()] = entry
	}

	return folded, nil
}

// foldEntry pairs a stream's folded state with its identity for lookup.
type foldEntry struct {
	state calls.CallState
	sid   id.StreamID
}

// assertFoldParity compares every projected row against the pure fold:
// present and equal while the stream is live, absent after a retire.
func assertFoldParity(rt *rapid.T, ctx context.Context, sc *systemscenario.Scenario) {
	folded, err := pureFold(sc)
	if err != nil {
		rt.Fatalf("pure fold: %v", err)
	}

	reader := metaengine.NewReader[calls.CallView](sc.System().MetaEngine(), calls.Collection)

	for refString, entry := range folded {
		key := calls.RecordKey(entry.sid)

		view, found, err := reader.Get(ctx, key)
		if err != nil {
			rt.Fatalf("lookup %s: %v", key, err)
		}

		if entry.state.Retired {
			if found {
				rt.Fatalf("stream %s is retired but still projected", refString)
			}

			continue
		}

		if !found {
			rt.Fatalf("stream %s has no projected row", refString)
		}

		want := calls.ViewFrom(entry.state.Current, entry.state.CurrentEnrichment)
		if !reflect.DeepEqual(view, want) {
			rt.Fatalf("stream %s projected row != pure fold\n got: %#v\nwant: %#v", refString, view, want)
		}
	}
}

// assertJournalVersionOrdering pins each stream's journal at strictly
// increasing versions starting at 1 — no gaps, no duplicates.
func assertJournalVersionOrdering(rt *rapid.T, sc *systemscenario.Scenario) {
	journal, err := journalEvents(sc)
	if err != nil {
		rt.Fatalf("read journal: %v", err)
	}

	seen := map[string]event.Version{}

	for _, evt := range journal {
		refString := evt.StreamID().String()

		if evt.Version() != seen[refString]+1 {
			rt.Fatalf(
				"stream %s journal version %d after %d: not strictly ordered",
				refString,
				evt.Version(),
				seen[refString],
			)
		}

		seen[refString] = evt.Version()
	}
}

// errJournalNotAJournal is returned when the event store lacks the
// Journal (ReadAll) capability the depth specs replay through.
var errJournalNotAJournal = errors.New("event store does not implement event.Journal (ReadAll)")

// journalEvents reads the scenario system's full journal — the same read
// the harness's own capture uses, reached through the documented escape
// hatch (Scenario.System) because the capture internals are unexported.
func journalEvents(sc *systemscenario.Scenario) ([]event.Event, error) {
	journal, ok := sc.System().EventStore().(event.Journal)
	if !ok {
		return nil, fmt.Errorf("%w", errJournalNotAJournal)
	}

	return journal.ReadAll(context.Background())
}

// journalOf is the fatal-ing wrapper the plain (non-property) tests use.
func journalOf(t *testing.T, sc *systemscenario.Scenario) []event.Event {
	t.Helper()

	events, err := journalEvents(sc)
	if err != nil {
		t.Fatal(err)
	}

	return events
}
