package telnyx

import (
	"slices"
	"strings"
	"time"

	"github.com/larsartmann/pbx-lens/internal/calls"
)

const DefaultExcerptMax = 200

type MapOptions struct {
	Direction  calls.Direction
	ExcerptMax int
}

func DefaultMapOptions() MapOptions {
	return MapOptions{Direction: calls.DirectionOutbound, ExcerptMax: DefaultExcerptMax}
}

func MapConversation(conv Conversation, msgs []Message, rec *Recording, opts MapOptions) (calls.Observation, error) {
	if opts.ExcerptMax <= 0 {
		opts.ExcerptMax = DefaultExcerptMax
	}

	// The messages API serves NEWEST-FIRST; the observation must be
	// order-independent, so sort chronologically here (regression:
	// TestExcerptOrderIndependent).
	turns := turnsByTime(msgs)

	obs := calls.Observation{
		NativeID:          calls.NativeID(conv.ID),
		Kind:              calls.SourceTelnyxConversation,
		Direction:         opts.Direction,
		StartedAt:         conv.CreatedAt.Truncate(time.Second).UTC(),
		DurationSecs:      durationSecs(conv, rec),
		FromNumber:        conv.MetadataString("telnyx_agent_target"),
		ToNumber:          conv.MetadataString("telnyx_end_user_target"),
		IsAI:              conv.MetadataString("assistant_id") != "",
		TranscriptExcerpt: excerpt(turns, opts.ExcerptMax),
		Transcript:        turns,
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

// turnsByTime maps wire messages to domain transcript turns in chronological
// order. Timestamps truncate to whole seconds BEFORE entering the turn: the
// live API is precision-unstable (nanoseconds in one response, microseconds
// minutes later), and sub-second noise must never reach ContentHash
// (AGENTS 2026-10-10).
func turnsByTime(msgs []Message) []calls.TranscriptTurn {
	sorted := slices.Clone(msgs)
	slices.SortStableFunc(sorted, func(a, b Message) int {
		return a.CreatedAt.Compare(b.CreatedAt.Time)
	})

	turns := make([]calls.TranscriptTurn, 0, len(sorted))
	for _, m := range sorted {
		if m.Role != "user" && m.Role != "assistant" {
			continue
		}

		text := strings.TrimSpace(m.Text)
		if text == "" {
			continue
		}

		turns = append(turns, calls.TranscriptTurn{
			Role: m.Role,
			Text: text,
			At:   m.CreatedAt.Truncate(time.Second).UTC(),
		})
	}

	return turns
}

// excerpt is the call opening for the list page: the first utterance of the
// human (what did they want?), falling back to the assistant greeting when
// the call never got a user turn. Input must already be chronological.
func excerpt(turns []calls.TranscriptTurn, maxRunes int) string {
	first := firstUserTurn(turns)
	if first == nil {
		first = firstAnyTurn(turns)
	}

	if first == nil {
		return ""
	}

	runes := []rune(first.Text)
	if len(runes) > maxRunes {
		return string(runes[:maxRunes])
	}

	return first.Text
}

func firstUserTurn(turns []calls.TranscriptTurn) *calls.TranscriptTurn {
	for i := range turns {
		if turns[i].Role == "user" {
			return &turns[i]
		}
	}

	return nil
}

func firstAnyTurn(turns []calls.TranscriptTurn) *calls.TranscriptTurn {
	if len(turns) == 0 {
		return nil
	}

	return &turns[0]
}
