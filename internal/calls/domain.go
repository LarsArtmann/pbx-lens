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
	EventObserved = event.Type("call.observed")
	CommandRecord = command.Type("call.record")
)

const StreamType = "Call"

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

type Observation struct {
	NativeID          NativeID   `json:"native_id"`
	Kind              SourceKind `json:"kind"`
	Direction         Direction  `json:"direction"`
	StartedAt         time.Time  `json:"started_at"`
	DurationSecs      int        `json:"duration_secs"`
	FromNumber        string     `json:"from_number"`
	ToNumber          string     `json:"to_number"`
	HangupCause       string     `json:"hangup_cause"`
	IsAI              bool       `json:"is_ai"`
	TranscriptExcerpt string     `json:"transcript_excerpt"`
	RecordingRef      string     `json:"recording_ref"`
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

	if o.FromNumber == "" && o.ToNumber == "" {
		return errorfamily.NewRejection(
			"call.record.both_parties_missing",
			"at least one participant number is required",
		)
	}

	return nil
}

func (o Observation) ContentHash() string {
	h := sha256.New()
	fmt.Fprintf(h, "%s\n%s\n%s\n%d\n%d\n%s\n%s\n%s\n%t\n%s\n%s",
		o.NativeID, o.Kind, o.Direction, o.StartedAt.UTC().UnixNano(), o.DurationSecs,
		o.FromNumber, o.ToNumber, o.HangupCause, o.IsAI, o.TranscriptExcerpt, o.RecordingRef)

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

type CallState struct {
	Observed    bool
	ContentHash string
}

func Apply(s CallState, evt event.Event) (CallState, error) {
	switch evt.Type() {
	case EventObserved:
		p, err := event.DecodePayloadAuto[CallObserved](evt)
		if err != nil {
			return s, err
		}

		s.Observed = true
		s.ContentHash = p.ContentHash()

		return s, nil
	default:
		return s, errorfamily.NewCorruption(
			"call.fold.unknown_event",
			fmt.Sprintf("call stream received unknown event type %q", evt.Type()),
		)
	}
}

func Record(obs Observation, streamID id.StreamID) decider.DecideFunc[CallState] {
	return func(s CallState, v event.Version) ([]event.Event, error) {
		if err := obs.Validate(); err != nil {
			return nil, err
		}

		if s.Observed {
			if s.ContentHash == obs.ContentHash() {
				return nil, nil
			}

			return nil, errorfamily.NewConflict(
				"call.record.mutated",
				fmt.Sprintf("stream %s already holds a different observation", streamID),
			)
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
