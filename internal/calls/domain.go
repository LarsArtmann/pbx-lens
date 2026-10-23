package calls

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
)

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

// CountRole reports how many turns a role spoke. Zero is a finding, not an
// absence: an AI-agent transcript with no user turn means the recognizer never
// closed a turn for the far end (2026-10-23 forensic call), so the page and
// the tests ask this question in the same place.
func CountRole(turns []TranscriptTurn, role TurnRole) int {
	count := 0

	for _, turn := range turns {
		if turn.Role == role {
			count++
		}
	}

	return count
}

// SourceEvent is one message the source recorded that is NOT a spoken
// transcript turn: its idle markers (Telnyx writes "[long silence]" when no
// user turn arrived), tool calls, and blank-text rows. It exists because a
// transcript with no user turn must never render as if nobody spoke unless
// the source itself said so — the marker is the source's own admission, and
// dropping it made a two-sided call look one-sided (2026-10-23 forensic
// report: a live human greeting was recorded, mapped away, and reported as
// "the caller did not respond").
//
// SourceEvents are deliberately OUTSIDE ContentHash: Transcript is the hashed
// identity, so persisting markers here cannot fork a stored view (see
// ContentHash). Role keeps the source's raw vocabulary rather than TurnRole,
// because a marker is not a speaker.
type SourceEvent struct {
	Role string    `json:"role"`
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
	SourceEvents      []SourceEvent    `json:"source_events,omitempty"`
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
