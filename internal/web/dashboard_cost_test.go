package web

import (
	"math"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/system/v4"
	"github.com/larsartmann/pbx-lens/internal/calls"
	"github.com/larsartmann/pbx-lens/internal/cli"
)

// enrichCall dispatches the enrichment command the poller's Detail-Records
// pass would dispatch, then drains so the read model carries the price.
func enrichCall(
	t *testing.T,
	sys *system.System,
	nativeID calls.NativeID,
	enr calls.Enrichment,
) {
	t.Helper()

	ctx := t.Context()

	view, found, err := CallDetail(ctx, sys, string(calls.SourceTelnyxConversation), string(nativeID))
	if err != nil || !found {
		t.Fatalf("read call %q for enrichment: found=%v err=%v", nativeID, found, err)
	}

	cmd, err := calls.NewEnrichCommand(view.Observation(), enr)
	if err != nil {
		t.Fatalf("build enrich command: %v", err)
	}

	if err := sys.CommandDispatcher().Dispatch(ctx, cmd); err != nil {
		t.Fatalf("dispatch enrich: %v", err)
	}

	if err := cli.DrainProjections(ctx, sys); err != nil {
		t.Fatalf("drain: %v", err)
	}
}

func TestDashboardSpendCountsOnlyPricedCalls(t *testing.T) {
	t.Parallel()

	started := testCallStarted()
	first := webObservation("conv-cost-a", started)
	second := webObservation("conv-cost-b", started.Add(time.Hour))
	unpriced := webObservation("conv-no-cost", started.Add(2*time.Hour))

	sys := newTestSystem(t, first, second, unpriced)
	enrichCall(t, sys, "conv-cost-a", calls.Enrichment{CostMicros: 500_000, CostCurrency: "USD"})
	enrichCall(t, sys, "conv-cost-b", calls.Enrichment{CostMicros: 120_000, CostCurrency: "USD"})

	stats, err := DashboardStatsQuery(t.Context(), sys, started.Add(3*time.Hour), ListCallsQuery{}, dashboardWindowDays)
	if err != nil {
		t.Fatalf("DashboardStatsQuery: %v", err)
	}

	if stats.Total != 3 || stats.CostKnown != 2 {
		t.Fatalf("total/priced = %d/%d, want 3/2 (the unpriced call is not a zero)", stats.Total, stats.CostKnown)
	}

	if stats.CostMicros != 620_000 {
		t.Errorf("spend = %d micros, want 620000", stats.CostMicros)
	}

	if stats.CostCurrency != "USD" {
		t.Errorf("currency = %q, want USD", stats.CostCurrency)
	}

	// 0.62 USD over 84 billed seconds = 0.4428… USD/min.
	if got := stats.CostPerMinute(); math.Abs(got-0.442857) > 0.0001 {
		t.Errorf("cost per minute = %v, want ~0.4429", got)
	}

	var daily float64
	for _, value := range stats.DailyCost() {
		daily += value
	}

	if math.Abs(daily-0.62) > 1e-9 {
		t.Errorf("daily spend series sums to %v, want 0.62", daily)
	}
}

func TestDashboardSpendRefusesMixedCurrencies(t *testing.T) {
	t.Parallel()

	started := testCallStarted()
	usd := webObservation("conv-usd", started)
	eur := webObservation("conv-eur", started.Add(time.Hour))

	sys := newTestSystem(t, usd, eur)
	enrichCall(t, sys, "conv-usd", calls.Enrichment{CostMicros: 300_000, CostCurrency: "USD"})
	enrichCall(t, sys, "conv-eur", calls.Enrichment{CostMicros: 200_000, CostCurrency: "EUR"})

	stats, err := DashboardStatsQuery(t.Context(), sys, started.Add(2*time.Hour), ListCallsQuery{}, dashboardWindowDays)
	if err != nil {
		t.Fatalf("DashboardStatsQuery: %v", err)
	}

	if stats.CostCurrency != "" {
		t.Errorf("currency = %q, want the empty (mixed) verdict", stats.CostCurrency)
	}

	_, body := getBody(t, muxForSystem(sys), "/dashboard")
	if !strings.Contains(body, "mixed/min") {
		t.Error("a mixed-currency window must say so instead of naming one currency")
	}
}

func TestDashboardRendersSpendCard(t *testing.T) {
	t.Parallel()

	started := testCallStarted()
	priced := webObservation("conv-cost", started)
	unpriced := webObservation("conv-free", started.Add(time.Hour))

	sys := newTestSystem(t, priced, unpriced)
	enrichCall(t, sys, "conv-cost", calls.Enrichment{CostMicros: 500_000, CostCurrency: "USD"})

	_, body := getBody(t, muxForSystem(sys), "/dashboard")

	for _, want := range []string{"Spend", "0.5 USD", "0.714 USD/min", "1 of 2 calls priced"} {
		if !strings.Contains(body, want) {
			t.Errorf("spend card missing %q", want)
		}
	}

	if strings.Contains(body, "No cost data yet") {
		t.Error("a priced window must not render the empty state")
	}
}

// muxForSystem serves the registered routes over an already-seeded index.
func muxForSystem(sys *system.System) *http.ServeMux {
	mux := http.NewServeMux()

	Register(mux, sys, Options{})

	return mux
}
