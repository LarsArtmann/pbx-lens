package web

import (
	"sort"

	"github.com/larsartmann/pbx-lens/internal/calls"
)

// hangupCauseLimit caps the cause table. A PBX produces a handful of causes;
// anything beyond the cap is folded into an "other" bar so the card never
// grows a scrollbar on a pathological source.
const hangupCauseLimit = 6

// HangupCauseCount is one hangup cause and how often the window ended with it.
type HangupCauseCount struct {
	Cause string
	Calls int
}

// QualityRollup turns the per-call truth signals the detail page shows one
// call at a time into standing rates for the whole window. Each rate carries
// its OWN denominator: "no user turn" is only meaningful over calls that had
// a transcript at all, and the recording tail only over calls whose turns
// carry timestamps. A rate over the wrong denominator is the lie this type
// exists to prevent.
type QualityRollup struct {
	HangupCauses []HangupCauseCount
	// Calls is the denominator for the hangup-cause distribution.
	Calls         int
	NoUserTurn    int
	Transcripts   int
	SelfDial      int
	PartiesKnown  int
	RecordingTail int
	TailEligible  int
}

// Rate renders one count against its denominator as a [0,1] ratio; a zero
// denominator yields the second result false, so the renderer shows an em
// dash instead of a fabricated 0%.
func Rate(count, denominator int) (float64, bool) {
	if denominator <= 0 {
		return 0, false
	}

	return float64(count) / float64(denominator), true
}

// qualityAccumulator folds one representative call into the rollup.
type qualityAccumulator struct {
	rollup  *QualityRollup
	causes  map[string]int
}

func newQualityAccumulator(rollup *QualityRollup) qualityAccumulator {
	return qualityAccumulator{rollup: rollup, causes: make(map[string]int)}
}

func (a qualityAccumulator) view(view calls.CallView) {
	a.rollup.Calls++

	cause := view.HangupCause
	if cause == "" {
		cause = "unknown"
	}

	a.causes[cause]++

	if len(view.Transcript) > 0 {
		a.rollup.Transcripts++

		if !hasUserTurn(view.Transcript) {
			a.rollup.NoUserTurn++
		}
	}

	if view.FromNumber != "" && view.ToNumber != "" {
		a.rollup.PartiesKnown++

		if view.FromNumber == view.ToNumber {
			a.rollup.SelfDial++
		}
	}

	if tail, ok := transcriptTail(view); ok {
		a.rollup.TailEligible++

		if tail >= transcriptTailThreshold {
			a.rollup.RecordingTail++
		}
	}
}

// finish resolves the accumulated causes into the ordered table.
func (a qualityAccumulator) finish() {
	a.rollup.HangupCauses = topHangupCauses(a.causes, hangupCauseLimit)
}

func hasUserTurn(turns []calls.TranscriptTurn) bool {
	for _, turn := range turns {
		if turn.Role == calls.TurnRoleUser {
			return true
		}
	}

	return false
}

// topHangupCauses orders causes by frequency (then alphabetically, so the
// table is stable between polls) and folds the tail into one "other" entry.
func topHangupCauses(causes map[string]int, limit int) []HangupCauseCount {
	ordered := make([]HangupCauseCount, 0, len(causes))
	for cause, count := range causes {
		ordered = append(ordered, HangupCauseCount{Cause: cause, Calls: count})
	}

	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].Calls != ordered[j].Calls {
			return ordered[i].Calls > ordered[j].Calls
		}

		return ordered[i].Cause < ordered[j].Cause
	})

	if len(ordered) <= limit {
		return ordered
	}

	other := HangupCauseCount{Cause: "other"}
	for _, entry := range ordered[limit:] {
		other.Calls += entry.Calls
	}

	return append(ordered[:limit], other)
}
