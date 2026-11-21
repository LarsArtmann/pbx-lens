package query

import (
	"time"

	"github.com/larsartmann/pbx-lens/internal/calls"
)

// TranscriptTailThreshold is the "the source stopped writing long before the
// line dropped" bar. The 2026-10-23 forensic call ended 29.2 s after its last
// recorded turn start (45.06 s recording, last turn at 15.83 s) while the far
// end was still speaking at 44.3 s, so a 30 s bar would have missed the very
// case this signal exists for; 20 s catches it without flagging a call that
// ends on a normal goodbye.
const TranscriptTailThreshold = 20 * time.Second

// TranscriptTail reports how long the call ran past the last turn's start.
// Turn times are truncated to whole seconds and sources without per-turn
// stamps leave At zero, so a transcript with no usable stamp yields false and
// the signal stays off rather than guessing.
func TranscriptTail(view calls.CallView) (time.Duration, bool) {
	var last time.Time

	for _, turn := range view.Transcript {
		if turn.At.After(last) {
			last = turn.At
		}
	}

	if last.IsZero() || view.DurationSecs <= 0 || view.StartedAt.IsZero() {
		return 0, false
	}

	tail := time.Duration(view.DurationSecs)*time.Second - last.Sub(view.StartedAt)
	if tail < 0 {
		return 0, false
	}

	return tail, true
}
