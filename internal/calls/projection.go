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
	NativeID          NativeID   `json:"native_id"`
	Kind              SourceKind `json:"kind"`
	Direction         Direction  `json:"direction"`
	StartedAt         time.Time  `json:"started_at"`
	DurationSecs      int        `json:"duration_secs"`
	FromNumber        string     `json:"from_number"`
	ToNumber          string     `json:"to_number"`
	HangupCause       string     `json:"hangup_cause"`
	IsAI              bool       `json:"is_ai"`
	TranscriptExcerpt string     `json:"transcript_excerpt"`
	RecordingRef      string     `json:"recording_ref"`
	ContentHash       string     `json:"content_hash"`
}

func ViewFrom(o Observation) CallView {
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
		RecordingRef:      o.RecordingRef,
		ContentHash:       o.ContentHash(),
	}
}

func Projection() ([]system.ProjectionDeclaration, *projectionadapter.TypeDecoder) {
	query := metaengine.Query[callLookup, CallView](Collection,
		metaengine.Volume(100_000),
		metaengine.OnRecordTyped(
			string(EventObserved),
			projectionadapter.EventWithID[CallObserved]{},
			func(_ record.Record, e projectionadapter.EventWithID[CallObserved]) (string, CallView) {
				return e.ID, ViewFrom(e.Payload.Observation)
			},
		),
	)

	decoder := projectionadapter.NewTypeDecoder(
		projectionadapter.Register(EventObserved, CallObserved{}),
	)

	return []system.ProjectionDeclaration{system.RawQuery(query)}, decoder
}
