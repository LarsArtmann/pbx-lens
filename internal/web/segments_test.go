package web

import (
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/pbx-lens/internal/calls"
)

func TestDashboardSegmentsSplitOursAndTheirs(t *testing.T) {
	t.Parallel()

	started := testCallStarted()

	outbound := webObservation("conv-out", started)
	outbound.FromNumber = "+4930123456"
	outbound.ToNumber = "+17287289311"

	inbound := webObservation("conv-in", started.Add(time.Hour))
	inbound.Direction = calls.DirectionInbound
	inbound.FromNumber = "+17287289311"
	inbound.ToNumber = "+4930123456"
	inbound.IsAI = true

	stats, err := DashboardStatsQuery(
		t.Context(),
		newTestSystem(t, outbound, inbound),
		started.Add(2*time.Hour),
		ListCallsQuery{},
		dashboardWindowDays,
	)
	if err != nil {
		t.Fatalf("DashboardStatsQuery: %v", err)
	}

	if len(stats.TopNumbers) != 1 || stats.TopNumbers[0].Number != "+4930123456" {
		t.Fatalf("our numbers = %+v, want only +4930123456", stats.TopNumbers)
	}

	if got := stats.TopNumbers[0]; got.Calls != 2 || got.AICalls != 1 {
		t.Errorf("our number = %+v, want 2 calls / 1 AI", got)
	}

	if len(stats.TopPeers) != 1 || stats.TopPeers[0].Number != "+17287289311" {
		t.Fatalf("counterparties = %+v, want only +17287289311", stats.TopPeers)
	}

	if got := stats.TopPeers[0]; got.Calls != 2 || got.AICalls != 1 {
		t.Errorf("counterparty = %+v, want 2 calls / 1 AI", got)
	}
}

func TestDashboardSegmentsRankByVolumeAndCap(t *testing.T) {
	t.Parallel()

	started := testCallStarted()
	seeded := make([]calls.Observation, 0, 14)

	// Number i gets (i+1) calls; seven numbers means the cap must bite.
	for i := range 7 {
		for repeat := range i + 1 {
			obs := webObservation(fmt.Sprintf("conv-%d-%d", i, repeat), started.Add(time.Duration(repeat)*time.Minute))
			obs.FromNumber = "+4930123456"
			obs.ToNumber = fmt.Sprintf("+1555000%04d", i)
			seeded = append(seeded, obs)
		}
	}

	stats, err := DashboardStatsQuery(
		t.Context(),
		newTestSystem(t, seeded...),
		started.Add(time.Hour),
		ListCallsQuery{},
		dashboardWindowDays,
	)
	if err != nil {
		t.Fatalf("DashboardStatsQuery: %v", err)
	}

	if len(stats.TopPeers) != segmentLimit {
		t.Fatalf("segments = %d rows, want the cap of %d", len(stats.TopPeers), segmentLimit)
	}

	for i, segment := range stats.TopPeers {
		wantCalls := 7 - i

		if segment.Calls != wantCalls {
			t.Fatalf("row %d = %+v, want %d calls (volume order)", i, segment, wantCalls)
		}
	}
}

func TestDashboardSegmentsCarryDrillThroughAndSpend(t *testing.T) {
	t.Parallel()

	started := testCallStarted()
	priced := webObservation("conv-priced", started)
	priced.FromNumber = "+4930999"
	priced.ToNumber = "+1555123456"

	unpriced := webObservation("conv-unpriced", started.Add(time.Minute))
	unpriced.FromNumber = "+4930999"
	unpriced.ToNumber = "+1555654321"

	sys := newTestSystem(t, priced, unpriced)
	enrichCall(t, sys, "conv-priced", calls.Enrichment{CostMicros: 250_000, CostCurrency: "USD"})

	stats, err := DashboardStatsQuery(t.Context(), sys, started.Add(time.Hour), ListCallsQuery{}, dashboardWindowDays)
	if err != nil {
		t.Fatalf("DashboardStatsQuery: %v", err)
	}

	if len(stats.TopNumbers) != 1 {
		t.Fatalf("our numbers = %+v", stats.TopNumbers)
	}

	segment := stats.TopNumbers[0]
	if !segment.HasCost() || segment.CostKnown != 1 || segment.CostMicros != 250_000 {
		t.Errorf("segment spend = %+v, want one priced call at 250000 micros", segment)
	}

	if !segment.HasRate() || segment.PricedSecs != 42 {
		t.Errorf("segment rate inputs = %+v, want the priced call's own 42 seconds", segment)
	}

	if want := 250_000.0 / 1_000_000 / (42.0 / 60); segment.CostPerMinute() != want {
		t.Errorf("segment CostPerMinute = %v, want %v", segment.CostPerMinute(), want)
	}

	if want := "/?number=%2B4930999"; segment.Href() != want {
		t.Errorf("drill-through href = %q, want %q", segment.Href(), want)
	}

	_, body := getBody(t, muxForSystem(sys), "/dashboard")
	if !strings.Contains(body, "Who calls what") || !strings.Contains(body, "+4930999") {
		t.Error("dashboard missing the segment table")
	}

	if !strings.Contains(body, "/min") {
		t.Error("segment table missing the per-minute rate column")
	}
}

func TestSegmentCostPerMinuteUsesPricedSecondsOnly(t *testing.T) {
	t.Parallel()

	// Two unpriced minutes must not dilute one minute priced at 0.25: the
	// rate names its denominator, exactly like the window-level rate.
	segment := Segment{
		Number:       "+49",
		Calls:        2,
		DurationSecs: 180,
		CostMicros:   250_000,
		CostKnown:    1,
		PricedSecs:   60,
	}

	if !segment.HasRate() {
		t.Fatal("HasRate = false for a segment with priced seconds")
	}

	if got, want := segment.CostPerMinute(), 0.25; math.Abs(got-want) > 1e-9 {
		t.Errorf("CostPerMinute = %v, want %v", got, want)
	}

	unpriced := Segment{Number: "+49", Calls: 1, DurationSecs: 60}
	if unpriced.HasRate() || unpriced.CostPerMinute() != 0 {
		t.Errorf("unpriced segment = %+v, want no rate, never a confident zero", unpriced)
	}
}

func TestListFiltersByEitherSideNumber(t *testing.T) {
	t.Parallel()

	started := testCallStarted()

	outbound := webObservation("conv-out", started)
	outbound.FromNumber = "+4930123456"
	outbound.ToNumber = "+17287289311"

	inbound := webObservation("conv-in", started.Add(time.Hour))
	inbound.Direction = calls.DirectionInbound
	inbound.FromNumber = "+17287289311"
	inbound.ToNumber = "+4930123456"

	unrelated := webObservation("conv-other", started.Add(2*time.Hour))
	unrelated.FromNumber = "+4930999999"
	unrelated.ToNumber = "+1555000111"

	mux := newTestServer(t, outbound, inbound, unrelated)

	_, body := getBody(t, mux, "/?number=%2B4930123456")
	if !strings.Contains(body, "conv-out") || !strings.Contains(body, "conv-in") {
		t.Error("both directions of the same number-led call must match")
	}

	if strings.Contains(body, "conv-other") {
		t.Error("an unrelated number must not match")
	}

	// The walk and the poll region must carry the filter, not silently drop it.
	if !strings.Contains(body, "number=%2B4930123456") {
		t.Error("filtered page must preserve the number in its links/poll URL")
	}
}

func TestListRejectsOverlongNumberFilter(t *testing.T) {
	t.Parallel()

	tooLong := strings.Repeat("9", maxNumberFilterLen+1)

	status, body := getBody(t, newTestServer(t), "/?number="+tooLong)
	if status != 400 {
		t.Fatalf("overlong number filter status = %d, want 400", status)
	}

	if !strings.Contains(body, "number is longer than") {
		t.Errorf("rejection should explain the limit, got: %s", body)
	}
}
