// Package telnyx enrichment mapping: Telnyx Detail Records and conversation
// insights are derived data — they refresh independently of the observation
// snapshot and enter the domain only through calls.Enrichment via call.enrich.
package telnyx

import (
	"math"
	"strconv"
	"strings"

	"github.com/larsartmann/pbx-lens/internal/calls"
)

// maxCostUnits bounds the parsed cost to a sane ceiling (a million currency
// units); anything above is a parse artifact or an account-level aggregate,
// never one call's cost.
const maxCostUnits = 1_000_000

// MicrosFromCost parses a DetailRecord cost literal (decimal string in
// currency units, e.g. "0.002") into domain cost micros. ok is false for
// empty, unparseable, negative, or absurd values — those mean "no cost
// reported", and CostMicros zero means exactly that in the domain.
func MicrosFromCost(raw string) (int64, bool) {
	f, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil || f < 0 || f > maxCostUnits {
		return 0, false
	}

	return int64(math.Round(f * calls.MicrosPerUnit)), true
}

// CostEnrichment maps one correlated detail record to the cost half of an
// enrichment. ok is false when the record carries no usable cost.
func CostEnrichment(rec DetailRecord) (int64, string, bool) {
	micros, ok := MicrosFromCost(rec.Cost)
	if !ok {
		return 0, "", false
	}

	return micros, rec.Currency, true
}

// InsightSummary extracts the enrichment summary from a conversation's
// insight rows: the newest COMPLETED row wins (pending/in_progress rows are
// still generating — the next pass re-fetches; failed rows never finalize),
// and its non-empty results join with newlines because one conversation can
// carry several insight templates.
func InsightSummary(insights []ConversationInsight) string {
	var newest *ConversationInsight

	for i := range insights {
		ins := &insights[i]
		if ins.Status != insightStatusCompleted {
			continue
		}

		if newest == nil || ins.CreatedAt.After(newest.CreatedAt.Time) {
			newest = ins
		}
	}

	if newest == nil {
		return ""
	}

	results := make([]string, 0, len(newest.Results))
	for _, r := range newest.Results {
		if text := strings.TrimSpace(r.Result); text != "" {
			results = append(results, text)
		}
	}

	return strings.Join(results, "\n")
}

const insightStatusCompleted = "completed"
