package cli

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
	"github.com/larsartmann/pbx-lens/internal/calls"
	"github.com/larsartmann/pbx-lens/internal/ingest/agent"
)

// FetchAgentObservations parses every transcript file in dir and maps each
// to an Observation, joining the freeswitch-cdr view by call uuid (the
// transcript filename) for parties, direction, and duration. Transcripts
// without a CDR match still ingest — agent transcripts are proof of a call
// on their own; the numbers fill in whenever the CDR lands. Read errors on
// individual transcripts are reported in rejections; directory-level errors
// abort.
func FetchAgentObservations(
	ctx context.Context,
	sys *system.System,
	dir string,
) ([]calls.Observation, []string, error) {
	parsed, err := agent.ParseDir(dir)
	if err != nil {
		return nil, nil, err
	}

	reader := metaengine.NewReader[calls.CallView](sys.MetaEngine(), calls.Collection)

	var (
		observations []calls.Observation
		rejected     []string
	)

	for _, transcript := range parsed.Transcripts {
		obs, mapErr := agent.MapTranscript(transcript, cdrViewFor(ctx, reader, transcript.UUID))
		if mapErr != nil {
			rejected = append(rejected, "agent-transcript:"+transcript.UUID+": "+mapErr.Error())

			continue
		}

		observations = append(observations, obs)
	}

	return observations, rejected, nil
}

// cdrViewFor looks up the joined freeswitch-cdr view by call uuid. The key
// derives from the observation's stream identity through calls.LookupView, so
// the projection's record-key format stays owned by one place, never re-typed
// at a call site.
func cdrViewFor(ctx context.Context, reader *metaengine.TypedReader[calls.CallView], callUUID string) *calls.CallView {
	obs := calls.Observation{NativeID: calls.NativeID(callUUID), Kind: calls.SourceFreeSwitchCDR}

	sid, err := obs.StreamID()
	if err != nil {
		return nil
	}

	view, found, err := calls.LookupView(ctx, reader, sid)
	if err != nil || !found {
		return nil
	}

	return &view
}
