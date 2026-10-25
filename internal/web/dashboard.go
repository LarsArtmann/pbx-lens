package web

import (
	"context"
	"fmt"
	"time"
	_ "time/tzdata"

	"github.com/larsartmann/go-cqrs-lite/system/v4"
	"github.com/larsartmann/pbx-lens/internal/calls"
)

// dashboardWindowDays is the chart window: the volume charts cover the last
// N days ending today, zero days included — a quiet stretch must show as a
// gap, not vanish.
const dashboardWindowDays = 30

// dayFormat is the dashboard's day key: the exact "2006-01-02" string is
// load-bearing — it keys ByDay buckets, matches the URL filters todayHref
// builds, and renders directly in the chart tooltip.
const dayFormat = "2006-01-02"

const (
	weekDays     = 7   // one WeekDelta half-window
	percentScale = 100 // ratio to percent

	secondsPerMinute = 60 // cost rate denominator
)

// dashboardLocation buckets dashboard days (R2, owner decision 2026-10-12):
// the operator's wall clock, Europe/Berlin — a German PBX's "Calls today"
// must agree with its own day, not drift an hour behind after 22:00 UTC.
// The embedded tzdata database keeps the zone resolvable in the hermetic
// nix binary with no system zoneinfo dependency.
//
//nolint:gochecknoglobals // injectable so tests can pin the zone (save/restore)
var dashboardLocation = mustZone("Europe/Berlin")

// daysPerWeek keeps the Mon-first weekday arithmetic self-describing.
const daysPerWeek = 7

func mustZone(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(fmt.Sprintf("dashboard zone %q unavailable: %v", name, err))
	}

	return loc
}

// dayKey buckets a timestamp into its dashboard-zone day (YYYY-MM-DD).
func dayKey(t time.Time) string {
	return t.In(dashboardLocation).Format(dayFormat)
}

// startOfDay returns midnight of t's dashboard-zone day. DST-safe: time.Date
// normalizes through zone transitions instead of misadding fixed hours.
func startOfDay(t time.Time) time.Time {
	local := t.In(dashboardLocation)
	year, month, day := local.Date()

	return time.Date(year, month, day, 0, 0, 0, 0, dashboardLocation)
}

// DayVolume is one day's call count in the dashboard window. MOSKnown is
// the subset of that day's calls whose source reported a quality score;
// MOSSum feeds AvgMOS. CostKnown is the priced subset and CostMicros its
// spend, in the same currency the window reports. Transcripts/NoUserTurn
// carry the per-day quality denominators so the no-user-turn share can be
// compared week over week.
type DayVolume struct {
	Day         string // YYYY-MM-DD, dashboard zone
	Calls       int
	MOSKnown    int
	MOSSum      float64
	CostKnown   int
	CostMicros  int64
	Transcripts int
	NoUserTurn  int
}

// AvgMOS is the day's mean MOS, or 0 when no call that day reported one;
// the renderer treats 0-known as "no data", never as a 0.0 measurement.
func (d DayVolume) AvgMOS() float64 {
	if d.MOSKnown == 0 {
		return 0
	}

	return d.MOSSum / float64(d.MOSKnown)
}

// DurationBucket is one histogram slot of call durations.
type DurationBucket struct {
	Label string
	Calls int
}

// durationBuckets is the histogram shape (R8/M20.2): inclusive upper
// bounds in seconds; every duration lands in exactly one bucket.
//
//nolint:gochecknoglobals // declarative histogram table, read-only
var durationBuckets = []struct {
	maxSecs int
	label   string
}{
	{29, "< 30s"},
	{60, "30-60s"},
	{300, "1-5m"},
	{900, "5-15m"},
	{3600, "15-60m"},
	{1 << 62, "> 1h"},
}

// WeekDelta compares the last 7 days' call volume with the previous 7.
// HasBaseline is false until a full previous week exists in the window.
type WeekDelta struct {
	Pct         float64
	HasBaseline bool
}

// DashboardStats is the quality-dashboard view model (T25): totals, AI
// share, average duration, and the per-day volume series. Aggregation
// streams over the filtered scan through the representative-row walk (one
// call counts once, however many sources described it) — memory stays at
// one keyset page, and the index is re-derivable.
type DashboardStats struct {
	Total       int
	Today       int
	AICalls     int
	HumanCalls  int
	AvgDuration time.Duration
	MOSKnown    int     // calls whose source reported a MOS score
	MOSSum      float64 // sum of reported scores; AvgMOS divides by MOSKnown
	WindowDays  int
	// Spend (north-star slice 1): enrichment-derived, so every figure carries
	// its own denominator. A call with no reported cost contributes nothing —
	// no zero is ever averaged into a rate that claims to cover the window.
	CostMicros   int64
	CostKnown    int
	CostSeconds  int
	CostCurrency string // the single currency the priced calls share; "" = mixed/unknown
	CostWeek     WeekDelta
	// HourDay buckets call starts by weekday (Mon-first) x hour of day in
	// the dashboard zone — the "when do calls happen" heatmap (plan T19).
	HourDay [7][24]int
	// Segments (north-star slice 2): the window grouped by the PBX-side
	// number (TopNumbers) and by the outside party (TopPeers), top N by volume.
	TopNumbers []Segment
	TopPeers   []Segment
	// Quality (north-star slice 3): hangup causes plus the per-call truth
	// signals, each over its own denominator.
	Quality QualityRollup
	// QualityTrend moves the no-user-turn share week over week in percentage
	// points — the health arrow for AI conversations. False HasBaseline
	// renders nothing, never a fabricated move.
	QualityTrend QualityTrend
	ByDay        []DayVolume
	Durations    []DurationBucket
	Week         WeekDelta

	// rangeFrom/rangeTo echo the picker's active range back into the form
	// (zero when unset).
	rangeFrom time.Time
	rangeTo   time.Time
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

// HasCostData reports whether any call in the window carried a price. False
// means the cost card renders its "no data" state, never a 0 spend.
func (s DashboardStats) HasCostData() bool { return s.CostKnown > 0 }

// CostPerMinute is the mean spend per billed minute across the priced calls,
// in currency units, or 0 when none reported a cost (or none billed a
// second). Dividing by the priced calls' own seconds keeps a partially
// priced window honest.
func (s DashboardStats) CostPerMinute() float64 {
	if s.CostSeconds <= 0 {
		return 0
	}

	units := float64(s.CostMicros) / calls.MicrosPerUnit

	return units / (float64(s.CostSeconds) / secondsPerMinute)
}

// DailyCost is the per-day spend series for the window, in currency units
// (micros are an API detail; the chart axis is money).
func (s DashboardStats) DailyCost() []float64 {
	values := make([]float64, 0, len(s.ByDay))
	for _, day := range s.ByDay {
		values = append(values, float64(day.CostMicros)/calls.MicrosPerUnit)
	}

	return values
}

// DashboardStatsQuery computes the dashboard aggregates over the read
// model, narrowed by the given filters (the date-range picker feeds
// from/to) over the given chart window in days (the 7/30-day presets;
// clamp to a sane minimum so garbage can't produce an empty chart). The
// day window always ends "today" in the dashboard zone, so an idle index
// shows its volume drifting out of the chart instead of freezing.
func DashboardStatsQuery(
	ctx context.Context,
	sys *system.System,
	now time.Time,
	filters ListCallsQuery,
	windowDays int,
) (DashboardStats, error) {
	if windowDays <= 0 {
		windowDays = dashboardWindowDays
	}

	stats := DashboardStats{
		WindowDays: windowDays,
		Durations:  make([]DurationBucket, len(durationBuckets)),
		rangeFrom:  filters.From,
		rangeTo:    filters.To,
	}

	today := dayKey(now)
	windowStart := startOfDay(now).AddDate(0, 0, -(windowDays - 1))
	dayIndex := make(map[string]int, windowDays)

	for offset := range windowDays {
		day := windowStart.AddDate(0, 0, offset).Format(dayFormat)
		dayIndex[day] = len(stats.ByDay)
		stats.ByDay = append(stats.ByDay, DayVolume{Day: day})
	}

	for i, bucket := range durationBuckets {
		stats.Durations[i] = DurationBucket{Label: bucket.label}
	}

	var totalDuration time.Duration

	acc := statsAccumulator{
		stats:          &stats,
		dayIndex:       dayIndex,
		today:          today,
		costCurrencies: make(map[string]int, 1),
		numberSegments: make(map[string]*Segment),
		peerSegments:   make(map[string]*Segment),
		quality:        newQualityAccumulator(&stats.Quality),
	}

	err := streamRepresentativeCalls(ctx, sys, filters, func(view calls.CallView) error {
		acc.view(view)
		totalDuration += time.Duration(view.DurationSecs) * time.Second

		return nil
	})
	if err != nil {
		return DashboardStats{}, err
	}

	if stats.Total > 0 {
		stats.AvgDuration = totalDuration / time.Duration(stats.Total)
	}

	stats.Week = weekDelta(stats.ByDay)
	stats.CostWeek = costWeekDelta(stats.ByDay)
	stats.QualityTrend = qualityTrend(stats.ByDay)
	stats.CostCurrency = unanimousCurrency(acc.costCurrencies)
	stats.TopNumbers = topSegments(acc.numberSegments, segmentLimit)
	stats.TopPeers = topSegments(acc.peerSegments, segmentLimit)
	acc.quality.finish()

	return stats, nil
}

// QualityTrend is the week-over-week movement of the no-user-turn share
// (the AI-conversation health signal): last 7 days versus the prior 7, in
// percentage POINTS. Points, not percent-of-percent, because a share that
// moves 2%→4% is +2 points of conversations that never got a user word —
// a "+100%" label would dramatize a small baseline into a crisis.
// HasBaseline is false until the prior week has transcripts AND a nonzero
// share to compare against; a share that was zero has no meaningful move.
type QualityTrend struct {
	Points      float64 // percentage-point change; negative = better
	HasBaseline bool
}

// qualityTrend derives the no-user-turn share movement from the per-day
// quality buckets, mirroring weekDelta's two-half-window shape.
func qualityTrend(days []DayVolume) QualityTrend {
	if len(days) < 2*weekDays {
		return QualityTrend{}
	}

	sum := func(half []DayVolume) (transcripts, noUser int) {
		for _, day := range half {
			transcripts += day.Transcripts
			noUser += day.NoUserTurn
		}

		return transcripts, noUser
	}

	thisTranscripts, thisNoUser := sum(days[len(days)-weekDays:])
	lastTranscripts, lastNoUser := sum(days[len(days)-2*weekDays : len(days)-weekDays])

	if thisTranscripts == 0 || lastTranscripts == 0 {
		return QualityTrend{}
	}

	thisShare := float64(thisNoUser) / float64(thisTranscripts)
	lastShare := float64(lastNoUser) / float64(lastTranscripts)

	if lastShare == 0 {
		return QualityTrend{HasBaseline: false}
	}

	return QualityTrend{
		Points:      (thisShare - lastShare) * percentScale,
		HasBaseline: true,
	}
}

// costWeekDelta compares the newest 7 days' spend with the previous 7, in the
// same shape the volume delta uses. A currency that only exists in the newer
// week has no baseline to compare against, exactly like a quiet first week.
func costWeekDelta(days []DayVolume) WeekDelta {
	if len(days) < 2*weekDays {
		return WeekDelta{}
	}

	var thisWeek, lastWeek int64

	for _, day := range days[len(days)-weekDays:] {
		thisWeek += day.CostMicros
	}

	for _, day := range days[len(days)-2*weekDays : len(days)-weekDays] {
		lastWeek += day.CostMicros
	}

	if lastWeek == 0 {
		return WeekDelta{HasBaseline: false}
	}

	return WeekDelta{
		Pct:         float64(thisWeek-lastWeek) / float64(lastWeek) * percentScale,
		HasBaseline: true,
	}
}

// unanimousCurrency reports the currency every priced call used, or "" when
// they disagree (a total summed across currencies would be a lie).
func unanimousCurrency(seen map[string]int) string {
	if len(seen) != 1 {
		return ""
	}

	for currency := range seen {
		return currency
	}

	return ""
}

// weekDelta compares the newest 7 day buckets with the previous 7. The
// window is always 30 days, so both weeks exist whenever the window does.
func weekDelta(days []DayVolume) WeekDelta {
	if len(days) < 2*weekDays {
		return WeekDelta{}
	}

	thisWeek, lastWeek := 0, 0

	for _, day := range days[len(days)-weekDays:] {
		thisWeek += day.Calls
	}

	for _, day := range days[len(days)-2*weekDays : len(days)-weekDays] {
		lastWeek += day.Calls
	}

	if lastWeek == 0 {
		return WeekDelta{HasBaseline: false}
	}

	return WeekDelta{
		Pct:         float64(thisWeek-lastWeek) / float64(lastWeek) * percentScale,
		HasBaseline: true,
	}
}

// todayHref deep-links the "Calls today" stat card into the filtered list.
func (s DashboardStats) todayHref(now time.Time) string {
	today := dayKey(now)

	return fmt.Sprintf("/?from=%s&to=%s", today, today)
}

// statsAccumulator folds one call view into the dashboard aggregates. The
// day buckets index by YYYY-MM-DD in the dashboard zone; a call outside the
// chart window still counts toward totals (the charts show the window, the
// headline cards show everything).
type statsAccumulator struct {
	stats    *DashboardStats
	dayIndex map[string]int
	today    string
	// costCurrencies counts the priced calls per currency, so the window can
	// report one currency only when every priced call agreed on it.
	costCurrencies map[string]int
	// Segment tallies by PBX-side number and by outside party; keys are the
	// observed numbers, values the running aggregate.
	numberSegments map[string]*Segment
	peerSegments   map[string]*Segment
	quality        qualityAccumulator
}

func (a statsAccumulator) view(view calls.CallView) {
	stats := a.stats

	stats.Total++

	if view.IsAI {
		stats.AICalls++
	} else {
		stats.HumanCalls++
	}

	day := dayKey(view.StartedAt)
	if day == a.today {
		stats.Today++
	}

	// Heatmap bucket: weekday x hour in the dashboard zone (plan T19). The
	// zone conversion is the DST story — Go's zone logic maps each instant
	// to the local wall clock that was in effect.
	local := view.StartedAt.In(dashboardLocation)
	weekday := (int(local.Weekday()) + daysPerWeek - int(time.Monday)) % daysPerWeek
	stats.HourDay[weekday][local.Hour()]++

	if idx, ok := a.dayIndex[day]; ok {
		stats.ByDay[idx].Calls++
	}

	a.viewQualityDay(view, day)

	for i, bucket := range durationBuckets {
		if view.DurationSecs <= bucket.maxSecs {
			stats.Durations[i].Calls++

			break
		}
	}

	a.viewMOS(view, day)
	a.viewCost(view, day)
	a.quality.view(view)
	a.viewSegments(view)
}

// viewMOS folds one call's quality score into the totals and its day bucket.
// Unscored calls (MOS == 0) contribute nothing anywhere.
func (a statsAccumulator) viewMOS(view calls.CallView, day string) {
	if view.MOS <= 0 {
		return
	}

	stats := a.stats
	stats.MOSKnown++
	stats.MOSSum += view.MOS

	if idx, ok := a.dayIndex[day]; ok {
		stats.ByDay[idx].MOSKnown++
		stats.ByDay[idx].MOSSum += view.MOS
	}
}

// viewQualityDay folds one call's transcript truth into its day bucket, so
// the no-user-turn share can trend week over week. Calls without a
// transcript contribute to neither side of the rate, exactly like the
// window-wide QualityRollup.
func (a statsAccumulator) viewQualityDay(view calls.CallView, day string) {
	if len(view.Transcript) == 0 {
		return
	}

	if idx, ok := a.dayIndex[day]; ok {
		a.stats.ByDay[idx].Transcripts++

		if !hasUserTurn(view.Transcript) {
			a.stats.ByDay[idx].NoUserTurn++
		}
	}
}

// viewCost folds one call's price into the totals and its day bucket. Only a
// reported price counts: CostMicros == 0 with a currency is "priced at
// nothing" as often as it is "not priced"; the enrichment contract writes a
// positive micros value for a priced call, so > 0 is the only honest "known"
// test.
func (a statsAccumulator) viewCost(view calls.CallView, day string) {
	if view.CostMicros <= 0 {
		return
	}

	stats := a.stats
	stats.CostKnown++
	stats.CostMicros += view.CostMicros
	stats.CostSeconds += view.DurationSecs
	a.costCurrencies[view.CostCurrency]++

	if idx, ok := a.dayIndex[day]; ok {
		stats.ByDay[idx].CostKnown++
		stats.ByDay[idx].CostMicros += view.CostMicros
	}
}

// viewSegments folds one call into the number/peer segment tables; a source
// row with no parties contributes to neither.
func (a statsAccumulator) viewSegments(view calls.CallView) {
	if ours, peer := callParties(view); ours != "" || peer != "" {
		if ours != "" {
			accumulateSegment(a.numberSegments, ours, view)
		}

		if peer != "" {
			accumulateSegment(a.peerSegments, peer, view)
		}
	}
}
