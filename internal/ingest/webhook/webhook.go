// Package webhook turns the telnyx-webhook receiver's inbound.jsonl into
// Observations. The receiver is LOG-ONLY for call.* events: every line is
// `{"received_at", "remote", "body": {"data": {"event_type", "occurred_at",
// "payload": {...}}}}`. One call spans several lifecycle events, so the
// parser AGGREGATES per call_control_id and emits one Observation per call
// per run — re-runs are no-ops by ContentHash.
package webhook

import (
	"bufio"
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/larsartmann/pbx-lens/internal/calls"
	"github.com/larsartmann/pbx-lens/internal/wire"
)

const (
	scannerInitialBuf = 64 * 1024
	scannerMaxLine    = 1024 * 1024
)

// Envelope is one JSONL line as the receiver writes it.
type Envelope struct {
	ReceivedAt wire.FlexTime `json:"received_at"`
	Remote     string        `json:"remote"`
	Body       struct {
		Data Event `json:"data"`
	} `json:"body"`
}

// Event is the Telnyx v2 webhook body.data shape (the fields the lens needs).
type Event struct {
	EventType  string        `json:"event_type"`
	OccurredAt wire.FlexTime `json:"occurred_at"`
	Payload    Payload       `json:"payload"`
}

// Payload carries the call-control identifiers present on call.* events.
type Payload struct {
	CallControlID string `json:"call_control_id"`
	CallSessionID string `json:"call_session_id"`
	Direction     string `json:"direction"` // incoming | outgoing
	From          string `json:"from"`
	To            string `json:"to"`
	HangupCause   string `json:"hangup_cause"`
}

// Result is one pass over the JSONL file.
type Result struct {
	Observations []calls.Observation
	// Skipped counts lines that are empty, undecodable, or carry no usable
	// call lifecycle event — a log-only source tolerates garbage lines
	// instead of failing the whole index.
	Skipped int
}

// call is the per-call_control_id aggregation state.
type call struct {
	startedAt time.Time
	endedAt   time.Time
	direction string
	from      string
	to        string
	cause     string
	sawCall   bool
}

// Parse reads the whole JSONL and folds it into one Observation per call.
// Events may arrive in any order (the receiver appends as Telnyx retries);
// the aggregation is order-independent so the ContentHash is stable.
func Parse(r io.Reader) (Result, error) {
	callsByID := map[string]*call{}
	res := Result{}

	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, scannerInitialBuf), scannerMaxLine)

	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}

		var env Envelope
		if err := json.Unmarshal([]byte(line), &env); err != nil {
			res.Skipped++

			continue
		}

		if !applyEvent(callsByID, env.Body.Data) {
			res.Skipped++
		}
	}

	if err := sc.Err(); err != nil {
		return Result{}, err
	}

	for callControlID, state := range callsByID {
		obs, err := observation(callControlID, state)
		if err != nil {
			res.Skipped++

			continue
		}

		res.Observations = append(res.Observations, obs)
	}

	return res, nil
}

// applyEvent folds one lifecycle event into the aggregation; it reports
// whether the event was usable.
func applyEvent(callsByID map[string]*call, evt Event) bool {
	switch evt.EventType {
	case "call.initiated", "call.answered", "call.hangup":
	default:
		return false
	}

	if evt.Payload.CallControlID == "" || evt.OccurredAt.IsZero() {
		return false
	}

	state, exists := callsByID[evt.Payload.CallControlID]
	if !exists {
		state = &call{}
		callsByID[evt.Payload.CallControlID] = state
	}

	at := evt.OccurredAt.UTC()

	if !state.sawCall || at.Before(state.startedAt) {
		state.startedAt = at
		state.sawCall = true
	}

	if at.After(state.endedAt) {
		state.endedAt = at
	}

	foldPayload(state, evt.Payload)

	if evt.Payload.HangupCause != "" {
		state.cause = evt.Payload.HangupCause
	}

	return true
}

// foldPayload keeps the FIRST non-empty value per field: retry deliveries of
// the same event must not change the aggregation.
func foldPayload(state *call, payload Payload) {
	if state.direction == "" {
		state.direction = payload.Direction
	}

	if state.from == "" {
		state.from = payload.From
	}

	if state.to == "" {
		state.to = payload.To
	}
}

func observation(callControlID string, state *call) (calls.Observation, error) {
	direction := calls.DirectionOutbound
	if state.direction == "incoming" {
		direction = calls.DirectionInbound
	}

	// Whole seconds before the domain: sub-second webhook noise must never
	// reach ContentHash (same doctrine as the Telnyx mapper).
	startedAt := state.startedAt.Truncate(time.Second)

	endedAt := state.endedAt.Truncate(time.Second)

	duration := 0
	if endedAt.After(startedAt) {
		duration = int(endedAt.Sub(startedAt).Seconds())
	}

	obs := calls.Observation{
		NativeID:     calls.NativeID(callControlID),
		Kind:         calls.SourceWebhookLog,
		Direction:    direction,
		StartedAt:    startedAt,
		DurationSecs: duration,
		FromNumber:   state.from,
		ToNumber:     state.to,
		HangupCause:  state.cause,
	}

	return obs, obs.Validate()
}
