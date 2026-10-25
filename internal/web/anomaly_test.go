package web

import (
	"testing"
	"time"
)

// volumeMatrix builds a DashboardStats whose ByDay covers the given number of days back
// from (but excluding) today, with each day's call count drawn per weekday
// from counts (index time.Sunday). The HourDay matrix is summed to match so
// the baseline's named source stays truthful.
func volumeMatrix(t *testing.T, now time.Time, days int, counts [7]int, todayCalls int) DashboardStats {
	t.Helper()

	stats := DashboardStats{WindowDays: days + 1, Today: todayCalls}

	for offset := days; offset >= 1; offset-- {
		day := now.AddDate(0, 0, -offset)
		local := day.In(dashboardLocation)
		calls := counts[int(local.Weekday())]

		stats.ByDay = append(stats.ByDay, DayVolume{Day: local.Format(dayFormat), Calls: calls})
		stats.HourDay[int(local.Weekday())%daysPerWeek][local.Hour()] += calls
	}

	return stats
}

// TestVolumeNoteUsualQuietBusy pins the band math on a synthetic four-week
// matrix: every prior Sunday carried exactly 10 calls, so the baseline is
// 10 with a ±2√10 band ([4, 16] after rounding) — 10 is usual, 2 is quiet,
// 17 is busy. The partial today never enters its own baseline.
func TestVolumeNoteUsualQuietBusy(t *testing.T) {
	now := time.Date(2026, 10, 25, 13, 0, 0, 0, dashboardLocation) // a Sunday
	counts := [7]int{10, 5, 5, 5, 5, 5, 5}                         // Sunday = 10, rest = 5 (time.Weekday: Sun=0)

	for _, tc := range []struct {
		name        string
		today       int
		wantKind    VolumeNoteKind
		wantLow     int
		wantHigh    int
		wantNonHist bool
	}{
		{"at the mean", 10, VolumeUsual, 4, 16, false},
		{"inside the band", 4, VolumeUsual, 4, 16, false},
		{"below the band", 2, VolumeQuiet, 4, 16, false},
		{"above the band", 17, VolumeBusy, 4, 16, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stats := volumeMatrix(t, now, 28, counts, tc.today)

			note := volumeNote(stats, now)

			if note.Kind != tc.wantKind {
				t.Fatalf("kind = %d (%s), want %d", note.Kind, note, tc.wantKind)
			}

			if note.Low != tc.wantLow || note.High != tc.wantHigh {
				t.Fatalf("band = [%d, %d], want [%d, %d]", note.Low, note.High, tc.wantLow, tc.wantHigh)
			}

			if note.Expected != 10 {
				t.Fatalf("expected = %v, want 10 (the today partial must not dilute the mean)", note.Expected)
			}

			if note.Weekday != "Sunday" {
				t.Fatalf("weekday = %q, want Sunday", note.Weekday)
			}
		})
	}
}

// TestVolumeNoteNotEnoughHistory: a single prior occurrence of today's
// weekday is a guess, not a baseline — the note stays silent (NoHistory),
// including on an all-zero window.
func TestVolumeNoteNotEnoughHistory(t *testing.T) {
	now := time.Date(2026, 10, 25, 13, 0, 0, 0, dashboardLocation) // a Sunday
	oneSunday := volumeMatrix(t, now, 7, [7]int{3, 0, 0, 0, 0, 0, 0}, 1)

	if note := volumeNote(oneSunday, now); note.Kind != VolumeNoHistory {
		t.Fatalf("one prior Sunday: kind = %d (%s), want NoHistory", note.Kind, note)
	}

	zeroWindow := volumeMatrix(t, now, 28, [7]int{}, 0)

	if note := volumeNote(zeroWindow, now); note.Kind != VolumeUsual || note.Low != 0 || note.High != 0 {
		t.Fatalf(
			"zero calls anywhere: kind = %d (%s) band [%d, %d], want Usual [0, 0]",
			note.Kind,
			note,
			note.Low,
			note.High,
		)
	}
}

// TestVolumeNoteZeroBaselineFlagsAnyCall: a weekday that never carried a
// call in four weeks has a [0, 0] band — the first call on such a day IS
// unusual, and the note says so instead of hiding behind an empty average.
func TestVolumeNoteZeroBaselineFlagsAnyCall(t *testing.T) {
	now := time.Date(2026, 10, 25, 13, 0, 0, 0, dashboardLocation) // a Sunday
	stats := volumeMatrix(t, now, 28, [7]int{0, 5, 5, 5, 5, 5, 5}, 1)

	note := volumeNote(stats, now)

	if note.Kind != VolumeBusy {
		t.Fatalf("kind = %d (%s), want Busy", note.Kind, note)
	}

	if note.Low != 0 || note.High != 0 {
		t.Fatalf("band = [%d, %d], want [0, 0]", note.Low, note.High)
	}
}

// TestVolumeNoteDashboardGoldenFreeze pins the frozen-clock state the page
// golden renders: seeds land on 2026-10-12 (a Monday), the golden's now is
// 2026-10-13 (a Tuesday). The zero-filled 30-day window holds four prior
// zero-Tuesdays, so the note is the honest Usual [0, 0] line — "0 is usual
// for a Tuesday" — and the golden pins exactly that sentence. If this flips,
// the golden needs a re-update BY DESIGN, not by midnight accident.
func TestVolumeNoteDashboardGoldenFreeze(t *testing.T) {
	now := time.Date(2026, 10, 13, 10, 0, 0, 0, time.UTC)

	var stats DashboardStats

	for offset := 29; offset >= 0; offset-- {
		day := now.AddDate(0, 0, -offset)
		local := day.In(dashboardLocation)
		calls := 0

		if local.Format(dayFormat) == "2026-10-12" {
			calls = 2 // the two seeded calls
		}

		stats.ByDay = append(stats.ByDay, DayVolume{Day: local.Format(dayFormat), Calls: calls})
	}

	note := volumeNote(stats, now)

	if note.Kind != VolumeUsual || note.Low != 0 || note.High != 0 || note.Today != 0 {
		t.Fatalf("golden-clock note = %s band [%d, %d], want Usual [0, 0] today 0", note, note.Low, note.High)
	}

	if got, want := note.String(), "today: 0 call(s), usual for a Tuesday (typically 0-0)"; got != want {
		t.Fatalf("note = %q, want %q", got, want)
	}
}

// TestVolumeNoteWeekdayOrdering: a Monday "now" reads its baseline from
// prior Mondays, never from the whole window — a mixed-weekday average
// would flatten the weekly rhythm the heatmap exists to show.
func TestVolumeNoteWeekdayOrdering(t *testing.T) {
	now := time.Date(2026, 10, 26, 9, 0, 0, 0, dashboardLocation) // a Monday
	counts := [7]int{4, 40, 4, 4, 4, 4, 4}                        // Monday = 40, rest = 4 (Sun=0, Mon=1)
	stats := volumeMatrix(t, now, 28, counts, 41)

	note := volumeNote(stats, now)

	if note.Kind != VolumeUsual {
		t.Fatalf("kind = %d (%s), want Usual (41 is inside a Monday baseline of 40)", note.Kind, note)
	}

	if note.Expected != 40 {
		t.Fatalf("expected = %v, want 40 (Mondays only)", note.Expected)
	}

	if note.Weekday != "Monday" {
		t.Fatalf("weekday = %q, want Monday", note.Weekday)
	}
}
