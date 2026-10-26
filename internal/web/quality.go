package web

import (
	"sort"

	"github.com/larsartmann/pbx-lens/internal/calls"
)

// hangupCauseLimit caps the cause table. A PBX produces a handful of causes;
// anything beyond the cap is folded into an "other" bar so the card never
// grows a scrollbar on a pathological source.
const hangupCauseLimit = 6

// shortCallMaxSecs bounds the short-call signal (P11): an ANSWERED call with
// fewer billed seconds than this hung up right after connect — the "did
// anything actually happen" tell. Ten seconds covers the connect-then-drop
// class while leaving normal brief calls ("yes, see you then") alone.
const shortCallMaxSecs = 10

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

	// Answer rate (P11): Answered over DurationReported. DurationReported
	// counts calls whose KIND reports a billsec (calls.ReportsDuration) —
	// agent-native rows are outside the denominator because their zero
	// duration means "not reported", not "unanswered". Answered counts
	// those whose billed seconds ran (DurationSecs > 0); ShortCalls counts
	// the answered whose billed seconds stayed under shortCallMaxSecs, so
	// its denominator is Answered, never DurationReported.
	DurationReported int
	Answered         int
	ShortCalls       int
	// UnansweredTop is the most common hangup cause among the unanswered
	// (zero-billsec, duration-reporting) calls — the "why did they fail"
	// cross the bare rate lacks. Zero-valued when nothing went unanswered.
	UnansweredTop HangupCauseCount
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
	rollup           *QualityRollup
	causes           map[string]int
	unansweredCauses map[string]int
}

func newQualityAccumulator(rollup *QualityRollup) qualityAccumulator {
	return qualityAccumulator{
		rollup:           rollup,
		causes:           make(map[string]int),
		unansweredCauses: make(map[string]int),
	}
}

func (a qualityAccumulator) view(view calls.CallView) {
	a.rollup.Calls++

	cause := view.HangupCause
	if cause == "" {
		cause = "unknown"
	}

	a.causes[cause]++

	if view.Kind.ReportsDuration() {
		a.rollup.DurationReported++

		if view.DurationSecs > 0 {
			a.rollup.Answered++

			if view.DurationSecs < shortCallMaxSecs {
				a.rollup.ShortCalls++
			}
		} else {
			a.unansweredCauses[cause]++
		}
	}

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

// finish resolves the accumulated causes into the ordered tables.
func (a qualityAccumulator) finish() {
	a.rollup.HangupCauses = topHangupCauses(a.causes, hangupCauseLimit)
	if top := topHangupCauses(a.unansweredCauses, 1); len(top) > 0 {
		a.rollup.UnansweredTop = top[0]
	}
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
