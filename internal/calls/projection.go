package calls

import (
	"time"

	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

const Collection = "call_views"

type CallView struct {
	ID                string     `json:"id"`
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
}

func ViewFrom(o Observation) CallView {
	return CallView{
		Kind:              o.Kind,
		Direction:         o.Direction,
		StartedAt:         o.StartedAt,
		DurationSecs:      o.DurationSecs,
		FromNumber:        o.FromNumber,
		ToNumber:          o.ToNumber,
		HangupCause:       o.HangupCause,
		IsAI:              o.IsAI,
		TranscriptExcerpt: o.TranscriptExcerpt,
		RecordingRef:      o.RecordingRef,
	}
}

func Evolution() system.EvolutionSpec {
	return system.OnEvolution(
		system.Evolve[CallView](Collection),
		string(EventObserved),
		CallObserved{},
		func(e CallObserved, v *CallView) {
			*v = ViewFrom(e.Observation)
		},
	).Done()
}
