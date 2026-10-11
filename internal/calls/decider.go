package calls

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

type CallState struct {
	Observed          bool
	Retired           bool
	RetireReason      string
	ContentHash       string
	EnrichHash        string
	Current           Observation
	CurrentEnrichment Enrichment
}

func Apply(state CallState, evt event.Event) (CallState, error) {
	switch evt.Type() {
	case EventObserved:
		p, err := event.DecodePayloadAuto[CallObserved](evt)
		if err != nil {
			return state, err
		}

		state.Observed = true
		state.ContentHash = p.ContentHash()
		state.Current = p.Observation
		state.CurrentEnrichment = Enrichment{}

		return state, nil

	case EventCorrected:
		p, err := event.DecodePayloadAuto[CallCorrected](evt)
		if err != nil {
			return state, err
		}

		state.Observed = true
		state.ContentHash = p.ContentHash()
		state.Current = p.Observation
		state.CurrentEnrichment = p.Enrichment
		state.EnrichHash = p.Enrichment.Hash()

		return state, nil

	case EventEnriched:
		p, err := event.DecodePayloadAuto[CallEnriched](evt)
		if err != nil {
			return state, err
		}

		state.Observed = true
		state.ContentHash = p.ContentHash()
		state.Current = p.Observation
		state.CurrentEnrichment = p.Enrichment
		state.EnrichHash = p.Hash()

		return state, nil

	case EventRetired:
		p, err := event.DecodePayloadAuto[CallRetired](evt)
		if err != nil {
			return state, err
		}

		state.Retired = true
		state.RetireReason = p.Reason

		return state, nil

	default:
		return state, errorfamily.NewCorruption(
			"call.fold.unknown_event",
			fmt.Sprintf("call stream received unknown event type %q", evt.Type()),
		)
	}
}

func Record(obs Observation, streamID id.StreamID) decider.DecideFunc[CallState] {
	return func(state CallState, v event.Version) ([]event.Event, error) {
		if err := obs.Validate(); err != nil {
			return nil, err
		}

		if state.Observed {
			if state.ContentHash == obs.ContentHash() {
				return nil, nil
			}

			// Sources are read-only but not immutable snapshots: a Telnyx
			// conversation grows between polls. Supersede via a correction
			// event instead of erroring, so the index converges to the final
			// state and the event log keeps the audit trail.
			evt, err := event.New(
				EventCorrected,
				streamID,
				StreamType,
				v.Increment(),
				CallCorrected{Observation: obs, Enrichment: state.CurrentEnrichment},
				event.WithSchemaVersion(1),
			)
			if err != nil {
				return nil, err
			}

			return []event.Event{evt}, nil
		}

		evt, err := event.New(
			EventObserved,
			streamID,
			StreamType,
			v.Increment(),
			CallObserved{Observation: obs},
			event.WithSchemaVersion(1),
		)
		if err != nil {
			return nil, err
		}

		return []event.Event{evt}, nil
	}
}

// Enrich attaches derived data (cost, insight) to an already-observed call.
// Enriching before the observation exists rejects: enrichment without its
// call is unaddressable, and the next poll re-attempts in the right order.
func Enrich(obs Observation, enr Enrichment, streamID id.StreamID) decider.DecideFunc[CallState] {
	return func(state CallState, v event.Version) ([]event.Event, error) {
		if !state.Observed {
			return nil, errorfamily.NewRejection(
				"call.enrich.before_observed",
				fmt.Sprintf("stream %s has no observation to enrich yet", streamID),
			)
		}

		if state.EnrichHash == enr.Hash() {
			return nil, nil
		}

		evt, err := event.New(
			EventEnriched,
			streamID,
			StreamType,
			v.Increment(),
			CallEnriched{Observation: state.Current, Enrichment: enr},
			event.WithSchemaVersion(1),
		)
		if err != nil {
			return nil, err
		}

		return []event.Event{evt}, nil
	}
}

// Retire applies a retention decision: the call leaves every read surface
// while the stream keeps its append-only audit trail. Retiring an
// unobserved stream rejects (nothing exists to retire) and re-retiring is
// an idempotent no-op, so a purge interrupted midway can safely rerun.
func Retire(reason RetentionReason, streamID id.StreamID) decider.DecideFunc[CallState] {
	return func(state CallState, v event.Version) ([]event.Event, error) {
		if !state.Observed {
			return nil, errorfamily.NewRejection(
				"call.retire.before_observed",
				fmt.Sprintf("stream %s has no observation to retire", streamID),
			)
		}

		if state.Retired {
			return nil, nil
		}

		evt, err := event.New(
			EventRetired,
			streamID,
			StreamType,
			v.Increment(),
			CallRetired{Reason: string(reason)},
			event.WithSchemaVersion(1),
		)
		if err != nil {
			return nil, err
		}

		return []event.Event{evt}, nil
	}
}
