package calls

import (
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/projectionadapter/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

const Collection = "call_views"

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
	ContentHash       string           `json:"content_hash"`
}

func ViewFrom(o Observation, e Enrichment) CallView {
	return CallView{
		NativeID:          o.NativeID,
		Kind:              o.Kind,
		Direction:         o.Direction,
		StartedAt:         o.StartedAt.UTC(),
		DurationSecs:      o.DurationSecs,
		FromNumber:        o.FromNumber,
		ToNumber:          o.ToNumber,
		HangupCause:       o.HangupCause,
		IsAI:              o.IsAI,
		TranscriptExcerpt: o.TranscriptExcerpt,
		Transcript:        o.Transcript,
		RecordingRef:      o.RecordingRef,
		CostMicros:        e.CostMicros,
		CostCurrency:      e.CostCurrency,
		InsightSummary:    e.InsightSummary,
		ContentHash:       o.ContentHash(),
	}
}

func Projection() ([]system.ProjectionDeclaration, *projectionadapter.TypeDecoder) {
	// All three folds are 2-arg create folds (full recompute): every event
	// payload carries the complete observation (and, for corrected/enriched,
	// the enrichment), so no fold needs a previous view. Pairing a create
	// fold with an update fold would route first events through the update
	// path with a zero prev and store a zero view (AGENTS gotcha 2026-10-10).
	query := metaengine.Query[callLookup, CallView](Collection,
		metaengine.Volume(100_000),
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
	)

	decoder := projectionadapter.NewTypeDecoder(
		projectionadapter.Register(EventObserved, CallObserved{}),
		projectionadapter.Register(EventCorrected, CallCorrected{}),
		projectionadapter.Register(EventEnriched, CallEnriched{}),
	)

	return []system.ProjectionDeclaration{system.RawQuery(query)}, decoder
}
