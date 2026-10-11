package telnyx

import (
	"testing"
)

func TestMicrosFromCost(t *testing.T) {
	tests := []struct {
		raw   string
		micro int64
		ok    bool
	}{
		{raw: "0.002", micro: 2000, ok: true},
		{raw: "0.0031", micro: 3100, ok: true},
		{raw: "1.5", micro: 1_500_000, ok: true},
		{raw: "0", micro: 0, ok: true},
		{raw: " 0.002 ", micro: 2000, ok: true},
		{raw: "", ok: false},
		{raw: "abc", ok: false},
		{raw: "-0.002", ok: false},
		{raw: "2000000", ok: false},
	}

	for _, test := range tests {
		got, ok := MicrosFromCost(test.raw)
		if ok != test.ok || (ok && got != test.micro) {
			t.Errorf("MicrosFromCost(%q) = (%d, %t), want (%d, %t)", test.raw, got, ok, test.micro, test.ok)
		}
	}
}

func TestCostEnrichmentCarriesCurrency(t *testing.T) {
	micros, currency, ok := CostEnrichment(DetailRecord{Cost: "0.0123", Currency: "USD"})
	if !ok || micros != 12_300 || currency != "USD" {
		t.Fatalf("CostEnrichment = (%d, %q, %t), want (12300, USD, true)", micros, currency, ok)
	}

	if _, _, ok := CostEnrichment(DetailRecord{Cost: "n/a"}); ok {
		t.Error("unparseable cost must report not-ok")
	}
}

func TestInsightSummaryNewestCompletedWins(t *testing.T) {
	older := ft(1000)
	newer := ft(2000)

	insights := []ConversationInsight{
		{ID: "old", Status: "completed", CreatedAt: older, Results: []InsightResult{{Result: "old summary"}}},
		{ID: "pending", Status: "pending", CreatedAt: newer, Results: []InsightResult{{Result: "not ready"}}},
		{ID: "new", Status: "completed", CreatedAt: newer, Results: []InsightResult{
			{InsightID: "t1", Result: "Caller asked about invoices."},
			{InsightID: "t2", Result: "  sent  "},
			{InsightID: "t3", Result: ""},
		}},
		{ID: "failed", Status: "failed", CreatedAt: newer},
	}

	got := InsightSummary(insights)
	want := "Caller asked about invoices.\nsent"
	if got != want {
		t.Fatalf("InsightSummary = %q, want %q", got, want)
	}
}

func TestInsightSummaryEmptyWithoutCompleted(t *testing.T) {
	insights := []ConversationInsight{
		{ID: "p", Status: "in_progress", Results: []InsightResult{{Result: "generating"}}},
	}

	if got := InsightSummary(insights); got != "" {
		t.Fatalf("in_progress-only rows must summarize to empty, got %q", got)
	}

	if got := InsightSummary(nil); got != "" {
		t.Fatalf("nil rows must summarize to empty, got %q", got)
	}
}
