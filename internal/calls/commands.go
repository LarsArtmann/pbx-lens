package calls

import (
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

const (
	CommandRecord = command.Type("call.record")
	CommandEnrich = command.Type("call.enrich")
	CommandRetire = command.Type("call.retire")
)

type RecordCallCmd struct {
	*command.BasicCommand

	Observation Observation
}

func NewRecordCommand(obs Observation) (RecordCallCmd, error) {
	if err := obs.Validate(); err != nil {
		return RecordCallCmd{}, err
	}

	sid, err := obs.StreamID()
	if err != nil {
		return RecordCallCmd{}, err
	}

	bc, err := command.New(CommandRecord, sid)
	if err != nil {
		return RecordCallCmd{}, err
	}

	return RecordCallCmd{BasicCommand: bc, Observation: obs}, nil
}

type EnrichCallCmd struct {
	*command.BasicCommand

	Observation Observation
	Enrichment  Enrichment
}

func NewEnrichCommand(obs Observation, enr Enrichment) (EnrichCallCmd, error) {
	if err := obs.Validate(); err != nil {
		return EnrichCallCmd{}, err
	}

	sid, err := obs.StreamID()
	if err != nil {
		return EnrichCallCmd{}, err
	}

	bc, err := command.New(CommandEnrich, sid)
	if err != nil {
		return EnrichCallCmd{}, err
	}

	return EnrichCallCmd{BasicCommand: bc, Observation: obs, Enrichment: enr}, nil
}

type RetireCallCmd struct {
	*command.BasicCommand

	Kind     SourceKind
	NativeID NativeID
	Reason   RetentionReason
}

// RetentionReason is the audit reason a retirement carries: the purge job
// stamps retention:age, and every legitimate value is a named constant in
// internal/retention — a free-form string here would let a typo'd reason
// corrupt the event-log grouping silently.
type RetentionReason string

// NewRetireCommand addresses the call by stream identity alone: a purge runs
// from read-model rows, and requiring a full observation would couple the
// retention job to the source's current snapshot shape.
func NewRetireCommand(kind SourceKind, nativeID NativeID, reason RetentionReason) (RetireCallCmd, error) {
	if reason == "" {
		return RetireCallCmd{}, errorfamily.NewRejection(
			"call.retire.empty_reason",
			"a retirement needs a policy reason for the audit trail",
		)
	}

	obs := Observation{NativeID: nativeID, Kind: kind}

	sid, err := obs.StreamID()
	if err != nil {
		return RetireCallCmd{}, err
	}

	bc, err := command.New(CommandRetire, sid)
	if err != nil {
		return RetireCallCmd{}, err
	}

	return RetireCallCmd{BasicCommand: bc, Kind: kind, NativeID: nativeID, Reason: reason}, nil
}
