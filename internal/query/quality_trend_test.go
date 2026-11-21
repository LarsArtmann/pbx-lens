package query

import (
	"strconv"
	"testing"
)

func TestQualityTrendMovesInPoints(t *testing.T) {
	t.Parallel()

	// 14 days: prior week 10 transcripts, 5 without a user turn (50%);
	// this week 10 transcripts, 2 without (20%) → −30 points.
	days := make([]DayVolume, 0, 2*daysPerWeek)
	for i := range 2 * daysPerWeek {
		day := DayVolume{Day: "2026-10-" + strconv.Itoa(i+1), Transcripts: 10}
		if i < daysPerWeek {
			day.NoUserTurn = 5
		} else {
			day.NoUserTurn = 2
		}

		days = append(days, day)
	}

	trend := qualityTrend(days)
	if !trend.HasBaseline || trend.Points != -30 {
		t.Errorf("qualityTrend = %+v, want HasBaseline with -30 points", trend)
	}

	// No transcripts in the prior week → no baseline, never a fabricated move.
	for i := range days[:daysPerWeek] {
		days[i].Transcripts = 0
		days[i].NoUserTurn = 0
	}

	if trend := qualityTrend(days); trend.HasBaseline {
		t.Errorf("qualityTrend = %+v, want no baseline without prior transcripts", trend)
	}

	// A prior week with a zero share has no meaningful move either (a
	// percentage from zero is drama, not signal).
	for i := range days[:daysPerWeek] {
		days[i].Transcripts = 10
		days[i].NoUserTurn = 0
	}

	if trend := qualityTrend(days); trend.HasBaseline {
		t.Errorf("qualityTrend = %+v, want no baseline from a zero prior share", trend)
	}

	// Short windows cannot compare weeks at all.
	if trend := qualityTrend(days[:daysPerWeek]); trend.HasBaseline {
		t.Errorf("qualityTrend = %+v, want no baseline from a short window", trend)
	}
}
