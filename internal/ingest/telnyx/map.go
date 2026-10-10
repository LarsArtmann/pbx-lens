package telnyx

import (
	"strings"

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

	obs := calls.Observation{
		NativeID:          calls.NativeID(conv.ID),
		Kind:              calls.SourceTelnyxConversation,
		Direction:         opts.Direction,
		StartedAt:         conv.CreatedAt.UTC(),
		DurationSecs:      durationSecs(conv, rec),
		FromNumber:        conv.MetadataString("telnyx_agent_target"),
		ToNumber:          conv.MetadataString("telnyx_end_user_target"),
		IsAI:              conv.MetadataString("assistant_id") != "",
		TranscriptExcerpt: excerpt(msgs, opts.ExcerptMax),
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
		return rec.DurationMillis / 1000
	}

	if conv.LastMessageAt.After(conv.CreatedAt) {
		return int(conv.LastMessageAt.Sub(conv.CreatedAt).Seconds())
	}

	return 0
}

func excerpt(msgs []Message, maxRunes int) string {
	for _, m := range msgs {
		if m.Role != "user" && m.Role != "assistant" {
			continue
		}

		text := strings.TrimSpace(m.Text)
		if text == "" {
			continue
		}

		runes := []rune(text)
		if len(runes) > maxRunes {
			return string(runes[:maxRunes])
		}

		return text
	}

	return ""
}
