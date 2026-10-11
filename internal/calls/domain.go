package calls

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

const (
	EventObserved  = event.Type("call.observed")
	EventCorrected = event.Type("call.corrected")
	EventEnriched  = event.Type("call.enriched")
	EventRetired   = event.Type("call.retired")

	CommandRecord = command.Type("call.record")
	CommandEnrich = command.Type("call.enrich")
	CommandRetire = command.Type("call.retire")
)

const StreamType = "Call"

// DefaultExcerptMax is the transcript-excerpt ceiling in runes that every
// source mapper applies: the excerpt renders in one list-table cell, so no
// source may ship an uncapped opening line into the read model. The bound
// lives in the domain because TranscriptExcerpt is a domain field — mappers
// must never disagree on its shape.
const DefaultExcerptMax = 200

// Excerpt selects the call opening for list-page display: the first user
// turn (what did they want?), falling back to the first turn of any role
// (the greeting), capped at maxRunes runes. The selection rule lives in the
// domain next to the ceiling — the twin implementations the source mappers
// carried before 2026-10-11 had begun to drift in their fallback wording,
// and the excerpt's meaning is a domain concern, not a mapper detail.
func Excerpt(turns []TranscriptTurn, maxRunes int) string {
	opening := ""

	for _, turn := range turns {
		if turn.Role == TurnRoleUser {
			opening = turn.Text

			break
		}
	}

	if opening == "" && len(turns) > 0 {
		opening = turns[0].Text
	}

	runes := []rune(opening)
	if len(runes) > maxRunes {
		return string(runes[:maxRunes])
	}

	return opening
}

type SourceKind string

const (
	SourceFreeSwitchCDR      SourceKind = "freeswitch-cdr"
	SourceTelnyxConversation SourceKind = "telnyx-conversation"
	SourceWebhookLog         SourceKind = "webhook-log"
	SourceAgentTranscript    SourceKind = "agent-transcript"
)

func (k SourceKind) Valid() bool {
	switch k {
	case SourceFreeSwitchCDR, SourceTelnyxConversation, SourceWebhookLog, SourceAgentTranscript:
		return true
	}

	return false
}

type Direction string

const (
	DirectionInbound  Direction = "inbound"
	DirectionOutbound Direction = "outbound"
)

func (d Direction) Valid() bool {
	return d == DirectionInbound || d == DirectionOutbound
}

type E164 string

func (n E164) Valid() bool {
	s := string(n)
	if len(s) < 8 || len(s) > 16 || !strings.HasPrefix(s, "+") {
		return false
	}

	_, err := strconv.Atoi(s[1:])

	return err == nil
}

func (n E164) Hash() ParticipantHash {
	sum := sha256.Sum256([]byte(n))

	return ParticipantHash(hex.EncodeToString(sum[:8]))
}

type ParticipantHash string

type NativeID string

// TurnRole is the transcript speaker role. The domain normalizes every
// source's vocabulary (Telnyx user/assistant, the FreeSWITCH agent's
// caller/agent) onto these two values; ContentHash renders the role via %s,
// so the string values are byte-identical to the plain strings used before
// the type existed and no historical hash changes.
type TurnRole string

const (
	TurnRoleUser      TurnRole = "user"
	TurnRoleAssistant TurnRole = "assistant"
)

// TranscriptTurn is one utterance of a call transcript, normalized to the
// user/assistant roles regardless of source vocabulary (Telnyx
// user/assistant, the FreeSWITCH agent's caller/agent). At is zero for
// sources that stamp no per-turn time (the agent JSONL writes a start ts
// only), and At is always truncated to whole seconds before it reaches the
// domain so the sub-second noise of live APIs never enters ContentHash.
type TranscriptTurn struct {
	Role TurnRole  `json:"role"`
	Text string    `json:"text"`
	At   time.Time `json:"at,omitzero"`
}

type Observation struct {
	NativeID          NativeID         `json:"native_id"`
	Kind              SourceKind       `json:"kind"`
	Direction         Direction        `json:"direction"`
	StartedAt         time.Time        `json:"started_at"`
	DurationSecs      int              `json:"duration_secs"`
	FromNumber        string           `json:"from_number"`
	ToNumber          string           `json:"to_number"`
	HangupCause       string           `json:"hangup_cause"`
	IsAI              bool             `json:"is_ai"`
	TranscriptExcerpt string           `json:"transcript_excerpt"`
	Transcript        []TranscriptTurn `json:"transcript,omitempty"`
	RecordingRef      string           `json:"recording_ref"`

	// MOS is the call's Mean Opinion Score when the source reports one
	// (FreeSWITCH CDR rtp_audio_in_mos). Zero means "not reported" and is
	// omitted from both JSON and ContentHash, so sources without MOS keep
	// their historical hashes.
	MOS float64 `json:"mos,omitempty"`
}

func (o Observation) Validate() error {
	if o.NativeID == "" {
		return errorfamily.NewRejection("call.record.empty_native_id", "observation needs the source's native id")
	}

	if !o.Kind.Valid() {
		return errorfamily.NewRejection(
			"call.record.unknown_source_kind",
			fmt.Sprintf("source kind %q is not one of the four index sources", o.Kind),
		)
	}

	if !o.Direction.Valid() {
		return errorfamily.NewRejection(
			"call.record.unknown_direction",
			fmt.Sprintf("direction %q must be inbound or outbound", o.Direction),
		)
	}

	if o.StartedAt.IsZero() {
		return errorfamily.NewRejection("call.record.zero_started_at", "observation needs a non-zero StartedAt")
	}

	if o.DurationSecs < 0 {
		return errorfamily.NewRejection("call.record.negative_duration", "duration cannot be negative")
	}

	if o.FromNumber == "" && o.ToNumber == "" && o.Kind != SourceAgentTranscript {
		return errorfamily.NewRejection(
			"call.record.both_parties_missing",
			"at least one participant number is required (agent transcripts may join one later)",
		)
	}

	return nil
}

func (o Observation) ContentHash() string {
	h := sha256.New()
	_, _ = fmt.Fprintf(h, "%s\n%s\n%s\n%d\n%d\n%s\n%s\n%s\n%t\n%s\n%s\n%s",
		o.NativeID, o.Kind, o.Direction, o.StartedAt.UTC().UnixNano(), o.DurationSecs,
		o.FromNumber, o.ToNumber, o.HangupCause, o.IsAI, o.TranscriptExcerpt,
		transcriptHash(o.Transcript), o.RecordingRef)

	// MOS joins the hash only when reported, so a source that never carries
	// it keeps the exact hash it had before the field existed.
	if o.MOS != 0 {
		_, _ = fmt.Fprintf(h, "\nmos=%g", o.MOS)
	}

	return hex.EncodeToString(h.Sum(nil))
}

func transcriptHash(turns []TranscriptTurn) string {
	if len(turns) == 0 {
		return ""
	}

	h := sha256.New()
	for _, turn := range turns {
		_, _ = fmt.Fprintf(h, "%s\x00%s\x00%d\x00", turn.Role, turn.Text, turn.At.UTC().Unix())
	}

	return hex.EncodeToString(h.Sum(nil))
}

func (o Observation) StreamID() (id.StreamID, error) {
	if o.NativeID == "" || !o.Kind.Valid() {
		return id.StreamID{}, errorfamily.NewRejection(
			"call.record.stream_id_unbuildable",
			"stream id needs a valid kind and native id",
		)
	}

	return id.ParseStreamID(string(o.Kind) + ":" + string(o.NativeID))
}

type CallObserved struct {
	Observation
}

// CallCorrected supersedes the stream's observation with a later snapshot
// from the same source (a Telnyx conversation that grew between polls). The
// enrichment rides along so the projection's full-recompute folds never lose
// cost/insight data a correction would otherwise clobber.
type CallCorrected struct {
	Observation

	Enrichment Enrichment
}

// CallEnriched carries derived data (cost, insight) that is NOT part of the
// observation identity: it refreshes independently of the source snapshot
// and never triggers corrections. The observation rides along for the same
// full-recompute reason as in CallCorrected.
type CallEnriched struct {
	Observation
	Enrichment
}

// CallRetired is the retention tombstone: it removes the call from the read
// model (every list, search, export, and dashboard surface) while the stream
// itself stays append-only in the event store. The payload carries no call
// data, only why: the audit answer lives in the stream id, and a tombstone
// must not re-introduce the PII it is retiring.
type CallRetired struct {
	Reason string `json:"reason"`
}

// Enrichment is derived data attached to a call after the fact: Telnyx
// Detail Records cost and AI conversation insights. Fields are zero when the
// source has not produced them yet; the hash covers the value fields only,
// never fetch bookkeeping, so re-polling an unchanged enrichment stays
// idempotent.
type Enrichment struct {
	CostMicros     int64  `json:"cost_micros,omitempty"`
	CostCurrency   string `json:"cost_currency,omitempty"`
	InsightSummary string `json:"insight_summary,omitempty"`
}

// MicrosPerUnit converts Telnyx cost micros to currency units. The domain
// owns the unit because CostMicros is a domain field: the ingest mapper and
// the web display must never disagree on it.
const MicrosPerUnit = 1_000_000

func (e Enrichment) Hash() string {
	if e == (Enrichment{}) {
		return ""
	}

	h := sha256.New()
	_, _ = fmt.Fprintf(h, "%d\n%s\n%s", e.CostMicros, e.CostCurrency, e.InsightSummary)

	return hex.EncodeToString(h.Sum(nil))
}

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
func Retire(reason string, streamID id.StreamID) decider.DecideFunc[CallState] {
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
			CallRetired{Reason: reason},
			event.WithSchemaVersion(1),
		)
		if err != nil {
			return nil, err
		}

		return []event.Event{evt}, nil
	}
}

type RetireCallCmd struct {
	*command.BasicCommand

	Kind     SourceKind
	NativeID NativeID
	Reason   string
}

// NewRetireCommand addresses the call by stream identity alone: a purge runs
// from read-model rows, and requiring a full observation would couple the
// retention job to the source's current snapshot shape.
func NewRetireCommand(kind SourceKind, nativeID NativeID, reason string) (RetireCallCmd, error) {
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
