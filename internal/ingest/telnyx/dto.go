package telnyx

import (
	"encoding/json"
	"strconv"

	"github.com/larsartmann/pbx-lens/internal/wire"
)

// This file holds the Telnyx wire shapes: the API's JSON decoded into Go
// structs, tolerant where the published spec is silent. The client lives in
// telnyx.go.

type Conversation struct {
	ID            string         `json:"id"`
	CreatedAt     wire.FlexTime  `json:"created_at"`
	LastMessageAt wire.FlexTime  `json:"last_message_at"`
	Name          string         `json:"name"`
	Metadata      map[string]any `json:"metadata"`
}

func (c Conversation) MetadataString(key string) string {
	s, _ := c.Metadata[key].(string)

	return s
}

type Message struct {
	Role      string        `json:"role"`
	Text      string        `json:"text"`
	CreatedAt wire.FlexTime `json:"created_at"`
	SentAt    wire.FlexTime `json:"sent_at"`
}

type DownloadURLs struct {
	MP3 string `json:"mp3"`
	WAV string `json:"wav"`
}

type Recording struct {
	ID                 string        `json:"id"`
	CallControlID      string        `json:"call_control_id"`
	Channels           string        `json:"channels"`
	CreatedAt          wire.FlexTime `json:"created_at"`
	DownloadURLs       DownloadURLs  `json:"download_urls"`
	DurationMillis     int           `json:"duration_millis"`
	RecordingStartedAt wire.FlexTime `json:"recording_started_at"`
	RecordingEndedAt   wire.FlexTime `json:"recording_ended_at"`
	From               string        `json:"from"`
	To                 string        `json:"to"`
	Status             string        `json:"status"`
}

// DetailRecord is one /detail_records row. The public OpenAPI spec publishes
// no call-control record schema (its DetailRecord union lists only
// messaging/conference/amd/verify/sim/media), so the decode is deliberately
// tolerant: both id spellings (id, uuid), both call-control id spellings
// (call_control_id, telnyx_call_control_id), and cost as either a decimal
// string ("0.002") or a JSON number. Live shape unverified (no API key at
// implementation time) — re-verify against the live API like the rest of the
// Telnyx cluster, quarterly per AGENTS.
type DetailRecord struct {
	ID            string
	CallControlID string
	Cost          string
	Currency      string
	StartedAt     wire.FlexTime
}

func (d *DetailRecord) UnmarshalJSON(data []byte) error {
	var raw struct {
		ID            *string          `json:"id"`
		UUID          *string          `json:"uuid"`
		CallControlID *string          `json:"call_control_id"`
		TelnyxCCID    *string          `json:"telnyx_call_control_id"`
		Cost          *json.RawMessage `json:"cost"`
		Currency      *string          `json:"currency"`
		StartedAt     wire.FlexTime    `json:"started_at"`
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	d.ID = firstString(raw.ID, raw.UUID)
	d.CallControlID = firstString(raw.CallControlID, raw.TelnyxCCID)
	d.Cost = costLiteral(raw.Cost)
	d.Currency = firstString(raw.Currency)
	d.StartedAt = raw.StartedAt

	return nil
}

func firstString(candidates ...*string) string {
	for _, c := range candidates {
		if c != nil && *c != "" {
			return *c
		}
	}

	return ""
}

// costLiteral accepts the documented decimal-string cost and tolerates a
// bare JSON number, normalizing both to the string form MicrosFromCost
// parses.
func costLiteral(raw *json.RawMessage) string {
	if raw == nil {
		return ""
	}

	var s string
	if err := json.Unmarshal(*raw, &s); err == nil {
		return s
	}

	var f float64
	if err := json.Unmarshal(*raw, &f); err == nil {
		return strconv.FormatFloat(f, 'f', -1, 64)
	}

	return ""
}

// ConversationInsight is one /ai/conversations/{id}/conversations-insights
// row (schema verified against the Telnyx OpenAPI spec 2026-10-11). Status
// is pending|in_progress|completed|failed; only completed rows carry final
// results.
type ConversationInsight struct {
	ID        string          `json:"id"`
	Status    string          `json:"status"`
	CreatedAt wire.FlexTime   `json:"created_at"`
	Results   []InsightResult `json:"conversation_insights"`
}

type InsightResult struct {
	InsightID string `json:"insight_id"`
	Result    string `json:"result"`
}

// DetailRecordByCallControl finds the detail record correlated to one call
// control id, mirroring the recordings correlation pattern: correlation is
// client-side because the endpoint's field-level filters beyond the
// documented time stamps are unverified.
func DetailRecordByCallControl(recs []DetailRecord, callControlID string) (DetailRecord, bool) {
	for _, r := range recs {
		if r.CallControlID == callControlID {
			return r, true
		}
	}

	return DetailRecord{}, false
}
