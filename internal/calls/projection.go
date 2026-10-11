package calls

import (
	"time"

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
	RecordingRef      string           `json:"recording_ref"`
	CostMicros        int64            `json:"cost_micros,omitempty"`
	CostCurrency      string           `json:"cost_currency,omitempty"`
	InsightSummary    string           `json:"insight_summary,omitempty"`
	MOS               float64          `json:"mos,omitempty"`
	ContentHash       string           `json:"content_hash"`
}

func ViewFrom(obs Observation, enr Enrichment) CallView {
	return CallView{
		NativeID:          obs.NativeID,
		Kind:              obs.Kind,
		Direction:         obs.Direction,
		StartedAt:         obs.StartedAt.UTC(),
		DurationSecs:      obs.DurationSecs,
		FromNumber:        obs.FromNumber,
		ToNumber:          obs.ToNumber,
		HangupCause:       obs.HangupCause,
		IsAI:              obs.IsAI,
		TranscriptExcerpt: obs.TranscriptExcerpt,
		Transcript:        obs.Transcript,
		RecordingRef:      obs.RecordingRef,
		CostMicros:        enr.CostMicros,
		CostCurrency:      enr.CostCurrency,
		InsightSummary:    enr.InsightSummary,
		MOS:               obs.MOS,
		ContentHash:       obs.ContentHash(),
	}
}

// Observation rebuilds the stored observation snapshot from the view.
// ViewFrom is a lossless projection of the observation (enrichment aside),
// so the rebuild's ContentHash equals the view's stored ContentHash — the
// invariant the enrich pass relies on when it needs the current observation
// to address a call.enrich command.
func (v CallView) Observation() Observation {
	return Observation{
		NativeID:          v.NativeID,
		Kind:              v.Kind,
		Direction:         v.Direction,
		StartedAt:         v.StartedAt,
		DurationSecs:      v.DurationSecs,
		FromNumber:        v.FromNumber,
		ToNumber:          v.ToNumber,
		HangupCause:       v.HangupCause,
		IsAI:              v.IsAI,
		TranscriptExcerpt: v.TranscriptExcerpt,
		Transcript:        v.Transcript,
		RecordingRef:      v.RecordingRef,
		MOS:               v.MOS,
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
		metaengine.OnRecordTyped(
			string(EventRetired),
			CallRetired{},
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
