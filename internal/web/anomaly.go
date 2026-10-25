package web

import (
	"fmt"
	"math"
	"time"
)

// The anomaly baseline (plan P12): a quiet "how does TODAY compare" note on
// the density heatmap. The expected volume for a weekday comes from the
// window's own history — the accumulated weekday×hour matrix the heatmap
// already carries — with a Poisson-shaped band (±2σ, counts are counts).
// Read-only over existing buckets; no new source, no new query.

// VolumeNoteKind discriminates what the baseline note says. The zero value
// renders nothing, so a caller that never computed a note stays silent.
type VolumeNoteKind uint8

const (
	// VolumeNoHistory: fewer than two prior occurrences of today's weekday —
	// a "usual" band from a single observation would be a guess.
	VolumeNoHistory VolumeNoteKind = iota
	// VolumeUsual: today's count sits inside the band.
	VolumeUsual
	// VolumeQuiet: today's count sits below the band.
	VolumeQuiet
	// VolumeBusy: today's count sits above the band.
	VolumeBusy
)

const (
	// volumeMinWeekdayHistory is the smallest number of complete prior
	// occurrences of today's weekday the band is willing to stand on — two
	// weeks of history, matching the quality trend's minimum.
	volumeMinWeekdayHistory = 2

	// volumeBandSigmas widens the band: ±2σ covers ~95% of a Poisson count.
	volumeBandSigmas = 2
)

// VolumeNote is today's verdict against the weekday baseline. Low/High are
// the rounded band edges the note quotes ("typically 4-16"); Expected is the
// unrounded mean for tests and future UI.
type VolumeNote struct {
	Kind      VolumeNoteKind
	Weekday   string // today's weekday, spelled out ("Tuesday")
	Today     int
	Expected  float64
	Low, High int
}

// volumeNote derives today's volume verdict from the dashboard's own
// buckets: the mean per weekday over the window's COMPLETE days only (a
// partial today must not dilute its own baseline), and today's count against
// the ±2σ band around that mean. Fewer than two prior same-weekdays renders
// the honest no-history state instead of a band built on one observation.
func volumeNote(stats DashboardStats, now time.Time) VolumeNote {
	local := now.In(dashboardLocation)
	todayKey := local.Format(dayFormat)
	weekday := local.Weekday()

	var (
		occurrences  int
		weekdayTotal int
	)

	for _, day := range stats.ByDay {
		if day.Day == todayKey {
			continue
		}

		parsed, err := time.ParseInLocation(dayFormat, day.Day, dashboardLocation)
		if err != nil {
			continue
		}

		if parsed.Weekday() == weekday {
			occurrences++
			weekdayTotal += day.Calls
		}
	}

	note := VolumeNote{Weekday: weekday.String(), Today: stats.Today}

	if occurrences < volumeMinWeekdayHistory {
		return note
	}

	expected := float64(weekdayTotal) / float64(occurrences)
	spread := volumeBandSigmas * math.Sqrt(expected)

	note.Kind = VolumeUsual
	note.Expected = expected
	note.Low = max(0, int(math.Round(expected-spread)))
	note.High = int(math.Round(expected + spread))

	switch {
	case float64(stats.Today) < expected-spread:
		note.Kind = VolumeQuiet
	case float64(stats.Today) > expected+spread:
		note.Kind = VolumeBusy
	}

	return note
}

// String renders the note's sentence for tests and logs; the templ renders
// its own markup around the same fields.
func (n VolumeNote) String() string {
	switch n.Kind {
	case VolumeNoHistory:
		return "not enough history for a usual-range note"
	case VolumeUsual:
		return fmt.Sprintf("today: %d call(s), usual for a %s (typically %d-%d)", n.Today, n.Weekday, n.Low, n.High)
	case VolumeQuiet:
		return fmt.Sprintf(
			"today: %d call(s), unusually quiet for a %s (typically %d-%d)",
			n.Today,
			n.Weekday,
			n.Low,
			n.High,
		)
	case VolumeBusy:
		return fmt.Sprintf(
			"today: %d call(s), unusually busy for a %s (typically %d-%d)",
			n.Today,
			n.Weekday,
			n.Low,
			n.High,
		)
	default:
		return fmt.Sprintf("unknown volume note kind %d", n.Kind)
	}
}
