package cli

import (
	"context"
	"time"

	"github.com/larsartmann/pbx-lens/internal/calls"
	"github.com/larsartmann/pbx-lens/internal/ingest/telnyx"
)

// DefaultRecordingsLookback is how many recent recordings one pass pulls to
// correlate against conversations (correlation is client-side by
// call_control_id; the recordings list filter's supported keys are
// undocumented, so no server-side filter is trusted).
const DefaultRecordingsLookback = 300

// TelnyxOptions shapes one Telnyx fetch pass.
type TelnyxOptions struct {
	// Since is the last_message_at lower bound; zero means unfiltered.
	Since time.Time
	// Direction is caller-declared: Telnyx conversation metadata exposes no
	// direction field (docs read 2026-10-10).
	Direction calls.Direction
	// Limit caps conversations fetched; 0 means all.
	Limit int
	// RecordingsLookback caps the recordings list pulled for correlation;
	// 0 means DefaultRecordingsLookback.
	RecordingsLookback int
}

// TelnyxSeed carries what the enrichment pass needs about one fetched
// conversation: its stream identity (the projection record key, built via
// the mapped observation's StreamID — never hand-assembled) and the
// call_control_id metadata the Detail Records correlation matches through.
type TelnyxSeed struct {
	ConversationID string
	CallControlID  string
	StreamKey      string
}

// FetchTelnyxObservations pulls conversations, their messages, and recent
// recordings, then maps each conversation to one observation. A conversation
// that cannot be fetched or mapped is reported in rejections (prefixed with
// its id) instead of failing the whole pass; fetch errors of the two list
// endpoints abort. Successfully mapped conversations also yield an
// enrichment seed for the Detail Records / insights pass.
func FetchTelnyxObservations(
	ctx context.Context,
	client *telnyx.Client,
	opts TelnyxOptions,
) ([]calls.Observation, []TelnyxSeed, []string, error) {
	convs, err := client.ListConversations(ctx, telnyx.ListOptions{Since: opts.Since, Limit: opts.Limit})
	if err != nil {
		return nil, nil, nil, err
	}

	lookback := opts.RecordingsLookback
	if lookback == 0 {
		lookback = DefaultRecordingsLookback
	}

	recs, err := client.ListRecordings(ctx, lookback)
	if err != nil {
		return nil, nil, nil, err
	}

	var (
		observations []calls.Observation
		seeds        []TelnyxSeed
		rejected     []string
	)

	for _, conv := range convs {
		msgs, msgErr := client.Messages(ctx, conv.ID)
		if msgErr != nil {
			rejected = append(rejected, conv.ID+": "+msgErr.Error())

			continue
		}

		var rec *telnyx.Recording
		if r, ok := telnyx.RecordingByCallControl(recs, conv.MetadataString("call_control_id")); ok {
			rec = &r
		}

		obs, mapErr := telnyx.MapConversation(conv, msgs, rec, telnyx.MapOptions{Direction: opts.Direction})
		if mapErr != nil {
			rejected = append(rejected, conv.ID+": "+mapErr.Error())

			continue
		}

		seed := TelnyxSeed{ConversationID: conv.ID, CallControlID: conv.MetadataString("call_control_id")}
		if sid, sidErr := obs.StreamID(); sidErr == nil {
			seed.StreamKey = sid.String()
		}

		observations = append(observations, obs)
		seeds = append(seeds, seed)
	}

	return observations, seeds, rejected, nil
}
