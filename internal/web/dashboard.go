package web

import (
	"context"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// dashboardWindowDays is the chart window: the volume charts cover the last
// N days ending today (UTC), zero days included — a quiet stretch must show
// as a gap, not vanish.
const dashboardWindowDays = 30

// utcDay is the UTC day boundary used to bucket and truncate timestamps.
const utcDay = 24 * time.Hour

// dayFormat is the dashboard's day key: the exact "2006-01-02" string is
// load-bearing — it keys ByDay buckets, matches the URL filters todayHref
// builds, and renders directly in the chart tooltip.
const dayFormat = "2006-01-02"

// DayVolume is one day's call count in the dashboard window. MOSKnown is
// the subset of that day's calls whose source reported a quality score;
// MOSSum feeds AvgMOS.
type DayVolume struct {
	Day      string // YYYY-MM-DD, UTC
	Calls    int
	MOSKnown int
	MOSSum   float64
}

// AvgMOS is the day's mean MOS, or 0 when no call that day reported one;
// the renderer treats 0-known as "no data", never as a 0.0 measurement.
func (d DayVolume) AvgMOS() float64 {
	if d.MOSKnown == 0 {
		return 0
	}

	return d.MOSSum / float64(d.MOSKnown)
}

// DashboardStats is the quality-dashboard view model (T25): totals, AI
// share, average duration, and the per-day volume series. Aggregation is
// in-process over the same bounded full scan the export uses — personal-PBX
// volumes are far below the cap, and the index is re-derivable.
type DashboardStats struct {
	Total       int
	Today       int
	AICalls     int
	HumanCalls  int
	AvgDuration time.Duration
	MOSKnown    int     // calls whose source reported a MOS score
	MOSSum      float64 // sum of reported scores; AvgMOS divides by MOSKnown
	WindowDays  int
	ByDay       []DayVolume
}

// AvgMOS is the mean MOS across every scored call in the index, or 0 when
// nothing has reported a score yet.
func (s DashboardStats) AvgMOS() float64 {
	if s.MOSKnown == 0 {
		return 0
	}

	return s.MOSSum / float64(s.MOSKnown)
}

// AIRatio is the AI share of all calls in [0,1]. Zero calls → zero value
// (the page renders an em dash, not a fake percentage).
func (s DashboardStats) AIRatio() float64 {
	if s.Total == 0 {
		return 0
	}

	return float64(s.AICalls) / float64(s.Total)
}

// DashboardStatsQuery computes the dashboard aggregates over the whole
// read model. The day window always ends "today" in the dashboard zone
// (dashboardLocation), so an idle index shows its volume drifting out of
// the chart instead of freezing.
func DashboardStatsQuery(ctx context.Context, sys *system.System, now time.Time) (DashboardStats, error) {
	stats := DashboardStats{WindowDays: dashboardWindowDays}

	today := dayKey(now)
	windowStart := startOfDay(now).AddDate(0, 0, -(dashboardWindowDays - 1))
	dayIndex := make(map[string]int, dashboardWindowDays)

	for offset := range dashboardWindowDays {
		day := startOfDay(now).AddDate(0, 0, offset).Format(dayFormat)
		dayIndex[day] = len(stats.ByDay)
		stats.ByDay = append(stats.ByDay, DayVolume{Day: day})
	}

	var totalDuration time.Duration

	err := StreamExportCalls(ctx, sys, ListCallsQuery{}, func(view calls.CallView) error {
		stats.Total++
		totalDuration += time.Duration(view.DurationSecs) * time.Second

		if view.IsAI {
			stats.AICalls++
		} else {
			stats.HumanCalls++
		}

		day := dayKey(view.StartedAt)
		if day == today {
			stats.Today++
		}

		if idx, ok := dayIndex[day]; ok {
			stats.ByDay[idx].Calls++
		}

		if view.MOS > 0 {
			stats.MOSKnown++
			stats.MOSSum += view.MOS

			if idx, ok := dayIndex[day]; ok {
				stats.ByDay[idx].MOSKnown++
				stats.ByDay[idx].MOSSum += view.MOS
			}
		}

		return nil
	})
	if err != nil {
		return DashboardStats{}, err
	}

	if stats.Total > 0 {
		stats.AvgDuration = totalDuration / time.Duration(stats.Total)
	}

	return stats, nil
}

// todayHref deep-links the "Calls today" stat card into the filtered list.
func (s DashboardStats) todayHref(now time.Time) string {
	today := now.UTC().Format(dayFormat)

	return fmt.Sprintf("/?from=%s&to=%s", today, today)
}
