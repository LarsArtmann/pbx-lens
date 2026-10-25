package web

import (
	"context"
	"encoding/json/v2"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	servertiming "github.com/larsartmann/httputil/server_timing"
	"github.com/larsartmann/pbx-lens/internal/retention"
	"github.com/larsartmann/templ-components/utils/format"
)

// opsSnapshot is the typed read of the ops counters the /ops page renders.
// The JSON endpoint (/api/ops) keeps serving the wire map unchanged; the
// page decodes the SAME composition, so the human view and the machine
// view can never disagree. Poller is nil when no poller payload exists;
// Cache is nil when the cache counters are unwired; both nil means the
// page renders its honest "nothing wired" state (the JSON twin answers
// 503 with the same guidance).
type opsSnapshot struct {
	Poller *opsPoller
	Cache  *CacheStats
	// Retention is the age-posture card: always present — "off" is a fact
	// the operator should see, not an absence to guess at.
	Retention retentionSnapshot
	// Coverage is the per-source index coverage (plan P17): what each source
	// actually delivered and how fresh its newest row is. Nil when the walk
	// failed — the card then says so instead of showing a fake empty table.
	// CoverageNow is the clock the ages were computed against (frozen in
	// tests, so the card stays deterministic).
	Coverage    []CoverageRow
	CoverageNow time.Time
	Wired       bool
}

// retentionSnapshot is the /ops retention card's view: the configured
// window, whether it is active, and how many indexed calls currently
// violate it (the would-retire count a purge would act on).
type retentionSnapshot struct {
	MaxAge time.Duration
	Active bool
	// OverAge is the live would-retire count; CountKnown is false when the
	// scan failed (the card then says so instead of showing a fake zero).
	OverAge    int
	CountKnown bool
}

// opsPoller is the decoded poller section: per-source counters plus the
// pass-health verdict. AgeKnown/StaleKnown are false when the payload
// predates the verdict keys — the page then renders "unknown" instead of
// inventing a healthy answer.
type opsPoller struct {
	Sources    []opsSourceRow
	LastPass   time.Time
	Age        time.Duration
	AgeKnown   bool
	Stale      bool
	StaleKnown bool
}

// opsSourceRow is one source's latest-pass counters, named for rendering.
type opsSourceRow struct {
	Name      string
	Offered   int
	Recorded  int
	Corrected int
	Rejected  []string
}

// opsWirePoller mirrors the /api/ops poller wire keys. Age and Stale are
// pointers so an ABSENT verdict (older producer, custom embed) stays
// distinguishable from a false one.
type opsWirePoller struct {
	Sources            map[string]opsWireSource `json:"sources"`
	LastPass           time.Time                `json:"last_pass"`
	LastPassAgeSeconds *float64                 `json:"last_pass_age_seconds"`
	Stale              *bool                    `json:"stale"`
}

// opsWireSource mirrors one source's wire counters.
type opsWireSource struct {
	Offered   int      `json:"offered"`
	Recorded  int      `json:"recorded"`
	Corrected int      `json:"corrected"`
	Rejected  []string `json:"rejected"`
}

// collectOps gathers and decodes every counter section for the page.
func (s *server) collectOps(ctx context.Context) opsSnapshot {
	snap := opsSnapshot{
		Wired:     s.ops != nil || s.cache != nil,
		Retention: s.collectRetention(ctx),
	}

	if s.ops != nil {
		snap.Poller = decodeOpsPoller(s.ops())
	}

	if s.cache != nil {
		cache := s.cache.Snapshot()
		snap.Cache = &cache
	}

	coverage, err := IngestCoverageQuery(ctx, s.sys, s.now())
	if err != nil {
		s.logf("ingest coverage walk failed: %v", err)
	} else {
		snap.Coverage = coverage
		snap.CoverageNow = s.now()
	}

	return snap
}

// collectRetention counts the indexed calls past the age window (the
// would-retire set a purge would act on). The scan is the same bounded
// read the purge's own -dry-run uses — read-only over the disposable
// index, never over a source.
func (s *server) collectRetention(ctx context.Context) retentionSnapshot {
	snap := retentionSnapshot{MaxAge: s.retention, Active: s.retention > 0}
	if !snap.Active {
		return snap
	}

	stop := servertiming.MeasureServerTiming(ctx, "retention")
	defer stop()

	policy := retention.Policy{MaxAge: s.retention}

	views, err := retention.OldViews(ctx, s.sys, policy, s.now())
	if err != nil {
		s.logf("retention over-age count failed: %v", err)

		return snap
	}

	snap.OverAge = len(views)
	snap.CountKnown = true

	return snap
}

// decodeOpsPoller round-trips the poller payload through its wire shape.
// The wire format is the contract (opsHandler serves exactly these keys),
// so decoding it — not reaching into the cli package's concrete types —
// keeps web decoupled from cli while the page renders the real thing.
// An undecodable payload yields a nil poller with Wired already true; the
// page then renders only the sections it understands.
func decodeOpsPoller(poller any) *opsPoller {
	raw, err := json.Marshal(poller)
	if err != nil {
		return nil
	}

	var wire opsWirePoller
	if err := json.Unmarshal(raw, &wire); err != nil {
		return nil
	}

	decoded := &opsPoller{LastPass: wire.LastPass}

	names := make([]string, 0, len(wire.Sources))
	for name := range wire.Sources {
		names = append(names, name)
	}

	slices.Sort(names)

	for _, name := range names {
		src := wire.Sources[name]
		decoded.Sources = append(decoded.Sources, opsSourceRow{
			Name:      name,
			Offered:   src.Offered,
			Recorded:  src.Recorded,
			Corrected: src.Corrected,
			Rejected:  src.Rejected,
		})
	}

	if wire.LastPassAgeSeconds != nil {
		decoded.Age = time.Duration(*wire.LastPassAgeSeconds * float64(time.Second))
		decoded.AgeKnown = true
	}

	if wire.Stale != nil {
		decoded.Stale = *wire.Stale
		decoded.StaleKnown = true
	}

	return decoded
}

// opsPage serves the human /ops page: the same counters /api/ops serves,
// rendered for a glance instead of a scrape.
func (s *server) opsPage(w http.ResponseWriter, r *http.Request) {
	render(w, r, opsRoute(nonce(r), s.collectOps(r.Context())))
}

// totalRejected sums a source row's rejection reasons for the table cell.
func (row opsSourceRow) totalRejected() int {
	return len(row.Rejected)
}

// ageText renders the pass age honestly: the sentinel (-1s = never ran)
// and unknown verdicts say so instead of rounding to "0s".
func (p opsPoller) ageText() string {
	switch {
	case !p.AgeKnown:
		return "unknown"
	case p.Age < 0:
		return "never"
	default:
		return format.CompactDuration(p.Age) + " ago"
	}
}

// verdictText is the one-glance pass-health answer.
func (p opsPoller) verdictText() string {
	switch {
	case !p.StaleKnown:
		return "unknown"
	case p.Stale:
		return "stale"
	default:
		return "healthy"
	}
}

// rejectedList caps the reasons rendered under the sources table so one
// noisy source cannot flood the page; the count in the table stays exact.
const rejectedListMax = 5

func (p opsPoller) rejectedText() string {
	reasons := make([]string, 0)

	for _, row := range p.Sources {
		reasons = append(reasons, row.Rejected...)
	}

	if len(reasons) == 0 {
		return ""
	}

	shown := reasons
	if len(reasons) > rejectedListMax {
		shown = reasons[:rejectedListMax]
	}

	out := "Rejected: " + joinReasons(shown)
	if rest := len(reasons) - len(shown); rest > 0 {
		out += " · " + strconv.Itoa(rest) + " more"
	}

	return out
}

func joinReasons(reasons []string) string {
	return strings.Join(reasons, " · ")
}
