package telnyx

import (
	"slices"
	"strings"
	"time"

	"github.com/larsartmann/pbx-lens/internal/calls"
)

// DefaultExcerptMax is the domain's excerpt ceiling; the alias keeps the
// mapper's public option surface speaking its own name.
const DefaultExcerptMax = calls.DefaultExcerptMax

type MapOptions struct {
	Direction  calls.Direction
	ExcerptMax int

	// ResolveDirection, when set, overrides Direction per conversation:
	// it receives the mapped party numbers (from = agent target, to = end
	// user target) and returns the direction the observation carries. Nil
	// keeps the caller-declared Direction for everything (today's
	// behavior).
	ResolveDirection func(from, to string) calls.Direction
}

func DefaultMapOptions() MapOptions {
	return MapOptions{Direction: calls.DirectionOutbound, ExcerptMax: DefaultExcerptMax}
}

// DIDDirectionResolver marks a conversation inbound when either party's
// number is a known AI DID: the sibling dialplan routes AI DIDs to the
// agent runtime, so the dialed number is the deployment-side DID (metadata
// key telnyx_agent_target, surfaced as the mapper's from), and DIDs do not
// originate calls, so an end-user-side match is not a real shape.
// Everything else keeps the declared direction.
func DIDDirectionResolver(
	aiDIDs map[string]bool,
	declared calls.Direction,
) func(from, to string) calls.Direction {
	return func(from, to string) calls.Direction {
		if aiDIDs[from] || aiDIDs[to] {
			return calls.DirectionInbound
		}

		return declared
	}
}

// ParseAIDIDs turns a comma-separated -ai-did flag value into the lookup
// set, trimming whitespace and dropping empties (same shape as cdr's
// ParseAIDestinations).
func ParseAIDIDs(flag string) map[string]bool {
	out := map[string]bool{}

	for part := range strings.SplitSeq(flag, ",") {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out[trimmed] = true
		}
	}

	return out
}

func MapConversation(conv Conversation, msgs []Message, rec *Recording, opts MapOptions) (calls.Observation, error) {
	if opts.ExcerptMax <= 0 {
		opts.ExcerptMax = DefaultExcerptMax
	}

	// The messages API serves NEWEST-FIRST; the observation must be
	// order-independent, so sort chronologically here (regression:
	// TestExcerptOrderIndependent).
	turns, events := turnsByTime(msgs)

	from := conv.MetadataString("telnyx_agent_target")
	to := conv.MetadataString("telnyx_end_user_target")

	direction := opts.Direction
	if opts.ResolveDirection != nil {
		direction = opts.ResolveDirection(from, to)
	}

	obs := calls.Observation{
		NativeID:          calls.NativeID(conv.ID),
		Kind:              calls.SourceTelnyxConversation,
		Direction:         direction,
		StartedAt:         conv.CreatedAt.Truncate(time.Second).UTC(),
		DurationSecs:      durationSecs(conv, rec),
		FromNumber:        from,
		ToNumber:          to,
		IsAI:              conv.MetadataString("assistant_id") != "",
		TranscriptExcerpt: calls.Excerpt(turns, opts.ExcerptMax),
		Transcript:        turns,
		SourceEvents:      events,
	}

	if rec != nil {
		obs.RecordingRef = rec.ID
	}

	return obs, obs.Validate()
}

func RecordingByCallControl(recs []Recording, callControlID string) (Recording, bool) {
	for _, r := range recs {
		if r.CallControlID == callControlID {
			return r, true
		}
	}

	return Recording{}, false
}

func durationSecs(conv Conversation, rec *Recording) int {
	if rec != nil && rec.DurationMillis > 0 {
		return rec.DurationMillis / millisPerSec
	}

	start := conv.CreatedAt.Truncate(time.Second)
	last := conv.LastMessageAt.Truncate(time.Second)

	if last.After(start) {
		return int(last.Sub(start).Seconds())
	}

	return 0
}

const millisPerSec = 1000

// turnsByTime maps wire messages to domain transcript turns and source events
// in chronological order. Timestamps truncate to whole seconds BEFORE entering
// the turn: the live API is precision-unstable (nanoseconds in one response,
// microseconds minutes later), and sub-second noise must never reach
// ContentHash (AGENTS 2026-10-10).
//
// Neither return value drops a message. Spoken turns become TranscriptTurn;
// everything else (idle markers like "[long silence]", tool calls, blank-text
// rows) becomes a SourceEvent. The earlier shape `continue`d on those rows,
// which is how a two-sided call could render as a silent one: the source had
// written its own admission that no user turn arrived, and the mapper threw
// the admission away (2026-10-23 forensic report).
func turnsByTime(msgs []Message) ([]calls.TranscriptTurn, []calls.SourceEvent) {
	sorted := slices.Clone(msgs)
	slices.SortStableFunc(sorted, func(a, b Message) int {
		return a.SpokeAt().Compare(b.SpokeAt())
	})

	turns := make([]calls.TranscriptTurn, 0, len(sorted))
	events := make([]calls.SourceEvent, 0, len(sorted))

	for _, m := range sorted {
		role := calls.TurnRole(m.Role)
		text := strings.TrimSpace(m.Text)
		at := m.SpokeAt().Truncate(time.Second).UTC()

		spoken := (role == calls.TurnRoleUser || role == calls.TurnRoleAssistant) && text != ""

		if spoken {
			turns = append(turns, calls.TranscriptTurn{Role: role, Text: text, At: at})

			continue
		}

		events = append(events, calls.SourceEvent{Role: m.Role, Text: text, At: at})
	}

	return turns, events
}
