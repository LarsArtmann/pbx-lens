package web

import (
	"net/url"
	"sort"
	"time"

	"github.com/larsartmann/pbx-lens/internal/calls"
)

// segmentLimit caps each breakdown table. The operator scans the top few
// numbers; the complete per-number set belongs to an export, not a card.
const segmentLimit = 5

// Segment is one grouped slice of the window: a number with the aggregates an
// operator actually asks about (how many calls, how many of them AI, how long
// they ran, what they cost). Every aggregate carries its own denominator, so
// a segment with no priced calls shows no spend rather than a confident zero.
type Segment struct {
	Number       string
	Calls        int
	AICalls      int
	DurationSecs int
	CostMicros   int64
	CostKnown    int
	// PricedSecs is the billed time of the priced calls only — the honest
	// denominator for the per-minute rate, so unpriced time never dilutes it.
	PricedSecs int
}

// AIRatio is the segment's AI share in [0,1]; zero calls → zero value.
func (s Segment) AIRatio() float64 {
	if s.Calls == 0 {
		return 0
	}

	return float64(s.AICalls) / float64(s.Calls)
}

// AvgDuration is the segment's mean call length.
func (s Segment) AvgDuration() time.Duration {
	if s.Calls == 0 {
		return 0
	}

	return time.Duration(s.DurationSecs/s.Calls) * time.Second
}

// HasCost reports whether any call in the segment carried a price.
func (s Segment) HasCost() bool { return s.CostKnown > 0 }

// HasRate reports whether the segment can state a per-minute rate at all:
// priced calls with at least one billed second. Without them the renderer
// shows an em dash, never a confident zero.
func (s Segment) HasRate() bool { return s.PricedSecs > 0 }

// CostPerMinute is the segment's mean spend per billed minute across its
// priced calls only — the same denominator discipline as the window rate
// (DashboardStats.CostPerMinute), so a partially priced segment states the
// rate its priced calls actually earned.
func (s Segment) CostPerMinute() float64 {
	if s.PricedSecs <= 0 {
		return 0
	}

	units := float64(s.CostMicros) / 1_000_000
	return units / (float64(s.PricedSecs) / secondsPerMinute)
}

// Href is the drill-through into the filtered call list: every call that has
// this number on either side, in the same order the list always uses.
func (s Segment) Href() string {
	return "/?number=" + url.QueryEscape(s.Number)
}

// callParties splits one call into the PBX-side number (the one this PBX is
// reachable on or presented as) and the outside party. An inbound call's
// destination is our number; an outbound call's caller-ID is. A source row
// with no parties (an agent transcript can legitimately carry neither yet)
// yields empty strings, and the caller skips that side.
func callParties(view calls.CallView) (string, string) {
	if view.Direction == calls.DirectionInbound {
		return view.ToNumber, view.FromNumber
	}

	return view.FromNumber, view.ToNumber
}

// accumulateSegment folds one call into a number's segment.
func accumulateSegment(segments map[string]*Segment, number string, view calls.CallView) {
	segment, ok := segments[number]
	if !ok {
		segment = &Segment{Number: number}
		segments[number] = segment
	}

	segment.Calls++
	segment.DurationSecs += view.DurationSecs

	if view.IsAI {
		segment.AICalls++
	}

	if view.CostMicros > 0 {
		segment.CostMicros += view.CostMicros
		segment.CostKnown++
		segment.PricedSecs += view.DurationSecs
	}
}

// topSegments orders the breakdown by volume, then by billed time, then by
// number — a total order, so the table never reshuffles between two polls of
// the same data.
func topSegments(segments map[string]*Segment, limit int) []Segment {
	ordered := make([]Segment, 0, len(segments))
	for _, segment := range segments {
		ordered = append(ordered, *segment)
	}

	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].Calls != ordered[j].Calls {
			return ordered[i].Calls > ordered[j].Calls
		}

		if ordered[i].DurationSecs != ordered[j].DurationSecs {
			return ordered[i].DurationSecs > ordered[j].DurationSecs
		}

		return ordered[i].Number < ordered[j].Number
	})

	if len(ordered) > limit {
		return ordered[:limit]
	}

	return ordered
}
