package calls

import (
	"github.com/larsartmann/go-cqrs-lite/metaengine/projectionadapter/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

const Collection = "call_views"

// viewVolumeCap sizes the call_views collection: one row per indexed call,
// with headroom for years of a personal PBX.
const viewVolumeCap = 100_000

type callLookup struct{}

type CallView struct {
	// The read model IS a composition, not a mirror (PC-1, 2026-11-20):
	// Observation and Enrichment embed whole, so adding a domain field can
	// never desynchronize the view — the JSON keys flatten identically
	// (field tags are byte-identical to the pre-embed view's), decode is
	// key-based so pre-embed rows read fine, and promoted field access
	// (view.FromNumber, view.CostMicros) keeps every call site shape.
	Observation
	Enrichment

	ContentHash string `json:"content_hash"`
}

// ViewFrom is the one canonical observation→view projection (the fold's
// defensive UTC normalization lives here and nowhere else). The observation
// is embedded losslessly, so the stored view's ContentHash equals the
// observation's — the invariant the enrich pass relies on when it needs the
// current observation to address a call.enrich command.
func ViewFrom(obs Observation, enr Enrichment) CallView {
	obs.StartedAt = obs.StartedAt.UTC()

	return CallView{
		Observation: obs,
		Enrichment:  enr,
		ContentHash: obs.ContentHash(),
	}
}

func Projection() ([]system.ProjectionDeclaration, *projectionadapter.TypeDecoder) {
	// All three folds are 2-arg create folds (full recompute): every event
	// payload carries the complete observation (and, for corrected/enriched,
	// the enrichment), so no fold needs a previous view. Pairing a create
	// fold with an update fold would route first events through the update
	// path with a zero prev and store a zero view (AGENTS gotcha 2026-10-10).
	query := metaengine.Query[callLookup, CallView](Collection,
		metaengine.Volume(viewVolumeCap),
		metaengine.OnRecordTyped(
			string(EventObserved),
			projectionadapter.EventWithID[CallObserved]{},
			func(_ record.Record, e projectionadapter.EventWithID[CallObserved]) (string, CallView) {
				return e.ID, ViewFrom(e.Payload.Observation, Enrichment{})
			},
		),
		metaengine.OnRecordTyped(
			string(EventCorrected),
			projectionadapter.EventWithID[CallCorrected]{},
			func(_ record.Record, e projectionadapter.EventWithID[CallCorrected]) (string, CallView) {
				return e.ID, ViewFrom(e.Payload.Observation, e.Payload.Enrichment)
			},
		),
		metaengine.OnRecordTyped(
			string(EventEnriched),
			projectionadapter.EventWithID[CallEnriched]{},
			func(_ record.Record, e projectionadapter.EventWithID[CallEnriched]) (string, CallView) {
				return e.ID, ViewFrom(e.Payload.Observation, e.Payload.Enrichment)
			},
		),
		// The retirement tombstone removes the view row; the stream's earlier
		// events stay in the store, but the create folds only run on replay of
		// those older events, after which the tombstone (last in the stream)
		// removes the row again. Replays converge to the same purged state.
		// The sample is the EventWithID envelope (not the bare payload)
		// because metaengine infers the remove key from the fold sample's
		// string field — the envelope's ID is that record key; the payload
		// itself carries only the RetentionReason, no string.
		metaengine.OnRecordTyped(
			string(EventRetired),
			projectionadapter.EventWithID[CallRetired]{},
			metaengine.Remove[CallView](),
		),
	)

	decoder := projectionadapter.NewTypeDecoder(
		projectionadapter.Register(EventObserved, CallObserved{}),
		projectionadapter.Register(EventCorrected, CallCorrected{}),
		projectionadapter.Register(EventEnriched, CallEnriched{}),
		projectionadapter.Register(EventRetired, CallRetired{}),
	)

	return []system.ProjectionDeclaration{system.RawQuery(query)}, decoder
}
