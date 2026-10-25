package web

import (
	"context"
	"embed"
	"encoding/json/v2"
	"io/fs"
	"maps"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/larsartmann/go-cqrs-lite/system/v4"
	errorfamily "github.com/larsartmann/go-error-family"
	servertiming "github.com/larsartmann/httputil/server_timing"
	"github.com/larsartmann/pbx-lens/internal/calls"
	"github.com/larsartmann/pbx-lens/internal/search"
)

//go:embed static
var staticFS embed.FS

// staleAfter is the freshness threshold (T12): when the newest indexed call
// is older than this, the list page shows the staleness banner.
const staleAfter = 6 * time.Hour

// Options bundles the web server's data sources beyond the read model.
type Options struct {
	Recordings RecordingOptions

	// Search answers FTS content queries (search.Index). Nil disables
	// /search with an explanatory 503.
	Search Searcher

	// Log receives mid-response failures the handler cannot turn into an
	// HTTP error (the body has already started). Nil logs nothing; serve
	// passes the process logger (slog-compatible shape: msg + kv args).
	Log func(msg string, args ...any)

	// Ops serves the poller's per-source ingest counters for /api/ops
	// (R15). Nil leaves the poller section out; the endpoint still answers
	// when Cache is wired.
	Ops func() any

	// Cache serves the ETag middleware's outcome counters as the "cache"
	// section of /api/ops (R26): generated tags, 304 revalidations, hit
	// ratio, buffer overflows. Nil leaves the section out; the endpoint
	// answers when at least one of Ops and Cache is wired.
	Cache *CacheCounters

	// RetentionMaxAge is the age window the ops page reports on (05-40
	// M10): the serve poller's ingest gate, shown alongside a live count of
	// indexed calls currently past it. Zero renders "off — keeping
	// everything"; retention must be opted into, never silent.
	RetentionMaxAge time.Duration

	// Now overrides the wall clock the dashboard buckets by (T22: the
	// today-count and day-bucket position are wall-clock-relative, so the
	// golden test pins them with a frozen clock; nil uses time.Now).
	Now func() time.Time
}

// Searcher is the content-query surface the web UI needs: the call view
// and the per-turn transcript view.
type Searcher interface {
	Query(ctx context.Context, query string, limit int) ([]search.Hit, error)
	QueryTurns(ctx context.Context, query string, limit int) ([]search.TurnHit, error)
}

// Register mounts every pbx-lens route on mux. The handler surface is
// strictly read-only over the read model.
func Register(mux *http.ServeMux, sys *system.System, opts Options) {
	timeout := resolveUpstreamTimeout(opts.Recordings.UpstreamTimeout)

	web := &server{
		sys:               sys,
		recordings:        opts.Recordings,
		search:            opts.Search,
		log:               opts.Log,
		ops:               opts.Ops,
		cache:             opts.Cache,
		retention:         opts.RetentionMaxAge,
		recTimeout:        timeout,
		upstream:          newUpstreamClient(timeout),
		now:               opts.Now,
		recordingThrottle: newRecordingThrottle(),
	}

	if web.now == nil {
		web.now = time.Now
	}

	mux.HandleFunc("GET /{$}", web.list)
	mux.HandleFunc("GET /dashboard", web.dashboard)
	mux.HandleFunc("GET /dashboard/{$}", redirectTrailingSlash("/dashboard"))
	mux.HandleFunc("GET /ops", web.opsPage)
	mux.HandleFunc("GET /ops/{$}", redirectTrailingSlash("/ops"))
	mux.HandleFunc("GET /frag/dashboard", web.dashboardFragment)
	mux.HandleFunc("GET /frag/staleness", web.stalenessFragment)
	mux.HandleFunc("GET /search", web.searchPage)
	mux.HandleFunc("GET /search/{$}", redirectTrailingSlash("/search"))
	mux.HandleFunc("GET /api/freshness", web.freshness)
	mux.HandleFunc("GET /api/ops", web.opsHandler)
	mux.HandleFunc("GET /api/export.jsonl", web.exportJSONL)
	mux.HandleFunc("GET /api/export.csv", web.exportCSV)
	mux.HandleFunc("GET /calls/{kind}/{id}", web.detail)
	mux.HandleFunc("GET /calls/{kind}/{id}/recording", web.recording)
	mux.HandleFunc("GET /segments/{number}", web.segmentPage)

	for route, asset := range staticRoutes {
		mux.HandleFunc(route, asset.serve)
	}

	mux.HandleFunc("GET /robots.txt", serveRobots)
}

// serveRobots answers the one honest robots.txt (M15/F46): everything is
// disallowed. The pages are auth-gated operator data on a private host;
// crawlers have no business here, and the file is cheaper than robots
// meta alone (which the pages ALSO carry via BaseProps SEO.NoIndex). It
// is intentionally public — robots.txt must be fetchable to be honored.
func serveRobots(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	_, _ = w.Write([]byte("User-agent: *\nDisallow: /\n"))
}

// redirectTrailingSlash answers /name/ with a permanent redirect to the
// canonical /name: the mux only auto-redirects the other direction (a subtree
// root missing its slash), so without this a trailing-slash bookmark 404s
// instead of reaching the canonical document's ETag chain.
func redirectTrailingSlash(canonical string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, canonical, http.StatusMovedPermanently)
	}
}

type server struct {
	sys               *system.System
	recordings        RecordingOptions
	search            Searcher
	log               func(msg string, args ...any)
	ops               func() any
	cache             *CacheCounters
	retention         time.Duration
	recTimeout        time.Duration
	upstream          *http.Client
	now               func() time.Time
	recordingThrottle recordingThrottle
}

// logf reports a failure that surfaces after the response body started —
// at that point an http.Error would append plaintext to a half-written
// body, so the only honest move is to log and let the truncated response
// stand.
func (s *server) logf(msg string, args ...any) {
	if s.log != nil {
		s.log(msg, args...)
	}
}

// listStaleness is the list page plus its staleness-banner inputs — the
// shared read behind the list page and its polled region.
type listStaleness struct {
	filters  ListCallsQuery
	page     CallListPage
	staleFor time.Duration
	stale    bool
}

// queryListStaleness is the shared body of the list page and its polled
// staleness region: parse the filters, read the page under the "query"
// Server-Timing span, and derive the banner inputs. Bad filters and index
// failures already wrote their error response; callers that get ok only
// render.
func (s *server) queryListStaleness(w http.ResponseWriter, r *http.Request) (listStaleness, bool) {
	filters, err := parseFilters(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return listStaleness{}, false
	}

	stop := servertiming.MeasureServerTiming(r.Context(), "query")
	page, err := ListCalls(r.Context(), s.sys, filters)

	stop()

	if err != nil {
		http.Error(w, "reading the call index failed: "+err.Error(), http.StatusInternalServerError)

		return listStaleness{}, false
	}

	staleFor, stale := staleness(filters, page)

	return listStaleness{filters: filters, page: page, staleFor: staleFor, stale: stale}, true
}

func (s *server) list(w http.ResponseWriter, r *http.Request) {
	view, ok := s.queryListStaleness(w, r)
	if !ok {
		return
	}

	render(w, r, listRoute(nonce(r), view.page, view.filters, view.staleFor, view.stale))
}

// parseFilters turns query parameters into the typed query; garbage values
// reject with a 400-class error instead of silently ignoring.
func parseFilters(r *http.Request) (ListCallsQuery, error) {
	query := ListCallsQuery{}

	if v := r.URL.Query().Get("direction"); v != "" {
		dir := calls.Direction(v)
		if !dir.Valid() {
			return ListCallsQuery{}, errorfamily.NewRejection(
				"lens.web.bad_direction",
				"direction must be inbound or outbound, got "+strconv.Quote(v),
			)
		}

		query.Direction = dir
	}

	if err := parseAIFilter(r.URL.Query().Get("ai"), &query); err != nil {
		return ListCallsQuery{}, err
	}

	if number := strings.TrimSpace(r.URL.Query().Get("number")); number != "" {
		if len(number) > maxNumberFilterLen {
			return ListCallsQuery{}, errorfamily.NewRejection(
				"lens.web.bad_number",
				"number is longer than "+strconv.Itoa(maxNumberFilterLen)+" characters; paste one number at a time",
			)
		}

		query.Number = number
	}

	for _, field := range []struct {
		name   string
		target *time.Time
	}{
		{name: "from", target: &query.From},
		{name: "to", target: &query.To},
		{name: "before", target: &query.Before},
		{name: "after", target: &query.After},
	} {
		v := r.URL.Query().Get(field.name)
		if v == "" {
			continue
		}

		parsed, dateOnly, err := parseDateParam(v)
		if err != nil {
			return ListCallsQuery{}, errorfamily.NewRejection(
				"lens.web.bad_date",
				field.name+" must be YYYY-MM-DD or RFC3339, got "+strconv.Quote(v),
			)
		}

		// A bare date on the upper bound means the whole day, not its
		// midnight — otherwise ?to= excludes the day it names.
		if field.name == "to" && dateOnly {
			parsed = parsed.AddDate(0, 0, 1).Add(-time.Second)
		}

		*field.target = parsed
	}

	return query, nil
}

func parseAIFilter(v string, query *ListCallsQuery) error {
	switch v {
	case "":
		return nil
	case "yes":
		yes := true
		query.AI = &yes
	case "no":
		no := false
		query.AI = &no
	default:
		return errorfamily.NewRejection(
			"lens.web.bad_ai",
			"ai must be yes or no, got "+strconv.Quote(v),
		)
	}

	return nil
}

// parseDateParam accepts a bare YYYY-MM-DD in the dashboard zone (flagged
// dateOnly, so callers can widen the bound to the whole day) or an RFC3339
// timestamp in whatever zone it carries. Bare dates mean the operator's
// own day (R2): ?from=2026-10-12 is the Berlin day, not UTC midnight.
func parseDateParam(v string) (time.Time, bool, error) {
	if t, err := time.ParseInLocation("2006-01-02", v, dashboardLocation); err == nil {
		return t, true, nil
	}

	t, err := time.Parse(time.RFC3339, v)

	return t, false, err
}

// staleness reports the age of the newest indexed call and whether it
// crosses the threshold. It rides ONLY the unfiltered first page, where
// "newest" is what the page itself already shows — no second scan, and a
// typed-filtered view (a deliberate slice) never carries a global-freshness
// warning. Walk cursors are pagination, not filtering: a walked page is
// still the unfiltered view, so its banner survives (plan T9). An empty
// index is never stale (the empty state explains itself).
func staleness(filters ListCallsQuery, page CallListPage) (time.Duration, bool) {
	if hasTypedFilters(filters) || len(page.Rows) == 0 {
		return 0, false
	}

	age := time.Since(page.Rows[0].When)

	return age, age > staleAfter
}

// hasTypedFilters reports whether the query narrows the SLICE (as opposed
// to walking it): direction/ai/from/to change what the list means;
// before/after only change where in the same list you stand.
func hasTypedFilters(filters ListCallsQuery) bool {
	return filters.Direction != "" || filters.AI != nil || filters.Number != "" ||
		!filters.From.IsZero() || !filters.To.IsZero()
}

// queryDashboardStats is the shared body of the dashboard page and its
// polled region: resolve the range picker, read the stats under the
// "stats" Server-Timing span. Bad ranges and index failures already wrote
// their error response; callers that get ok only render. The fragment URL
// rides along so the polled region re-requests the same range and window.
func (s *server) queryDashboardStats(w http.ResponseWriter, r *http.Request) (DashboardStats, string, bool) {
	filters, fragURL, window, ok := parseRange(w, r)
	if !ok {
		return DashboardStats{}, "", false
	}

	stop := servertiming.MeasureServerTiming(r.Context(), "stats")
	stats, err := DashboardStatsQuery(r.Context(), s.sys, s.now(), filters, window)

	stop()

	if err != nil {
		http.Error(w, "reading the call index failed: "+err.Error(), http.StatusInternalServerError)

		return DashboardStats{}, "", false
	}

	return stats, fragURL, true
}

// dashboard serves the quality-dashboard page (T25). The from/to picker
// narrows the underlying scan; the cards region polls for updates (R9).
func (s *server) dashboard(w http.ResponseWriter, r *http.Request) {
	stats, fragURL, ok := s.queryDashboardStats(w, r)
	if !ok {
		return
	}

	render(w, r, dashboardRoute(nonce(r), stats, s.now(), fragURL))
}

// dashboardFragment is the polled region payload (R9): the same live cards
// the page renders, without the shell — the region replaces itself.
func (s *server) dashboardFragment(w http.ResponseWriter, r *http.Request) {
	stats, fragURL, ok := s.queryDashboardStats(w, r)
	if !ok {
		return
	}

	render(w, r, dashboardLive(nonce(r), stats, s.now(), fragURL))
}

// parseRange reads the dashboard's date-range picker (from/to only — list
// filters like direction/ai are not dashboard concepts and stay ignored)
// and derives the polled-region URL carrying the same range, plus the chart
// window preset (?window=7|30), which rides the fragment URL so the polled
// region re-renders the same chart span the page chose. Garbage dates or
// windows reject with a 400, house style.
func parseRange(w http.ResponseWriter, r *http.Request) (ListCallsQuery, string, int, bool) {
	var filters ListCallsQuery

	window, ok := parseWindowParam(w, r)
	if !ok {
		return ListCallsQuery{}, "", 0, false
	}

	if !parseRangeDates(w, r, &filters) {
		return ListCallsQuery{}, "", 0, false
	}

	fragURL := "/frag/dashboard"

	query := url.Values{}

	if window != dashboardWindowDays {
		query.Set("window", strconv.Itoa(window))
	}

	if !filters.From.IsZero() {
		query.Set("from", filters.From.Format(dayFormat))
	}

	if !filters.To.IsZero() {
		// To was widened to end-of-day; the date format drops the time,
		// so the picker echo shows the day the operator picked.
		query.Set("to", filters.To.Format(dayFormat))
	}

	if len(query) > 0 {
		fragURL += "?" + query.Encode()
	}

	return filters, fragURL, window, true
}

// parseWindowParam reads the chart-window preset (?window=7|30); anything
// else rejects with a 400 and reports false.
func parseWindowParam(w http.ResponseWriter, r *http.Request) (int, bool) {
	window := dashboardWindowDays

	if v := r.URL.Query().Get("window"); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil || (parsed != 7 && parsed != 30) {
			http.Error(w, "window must be 7 or 30, got "+strconv.Quote(v), http.StatusBadRequest)

			return 0, false
		}

		window = parsed
	}

	return window, true
}

// parseRangeDates fills from/to from the picker params; garbage rejects
// with a 400 and reports false.
func parseRangeDates(w http.ResponseWriter, r *http.Request, filters *ListCallsQuery) bool {
	for _, field := range []struct {
		name   string
		target *time.Time
	}{
		{name: "from", target: &filters.From},
		{name: "to", target: &filters.To},
	} {
		v := r.URL.Query().Get(field.name)
		if v == "" {
			continue
		}

		parsed, dateOnly, err := parseDateParam(v)
		if err != nil || !dateOnly {
			http.Error(w, field.name+" must be YYYY-MM-DD, got "+strconv.Quote(v), http.StatusBadRequest)

			return false
		}

		// A bare upper bound covers the whole day it names (same semantics
		// as the list filters).
		if field.name == "to" {
			parsed = parsed.AddDate(0, 0, 1).Add(-time.Second)
		}

		*field.target = parsed
	}

	return true
}

// stalenessFragment re-checks the freshness banner (R9): the polled region
// re-renders the banner, or nothing once the index is fresh again.
func (s *server) stalenessFragment(w http.ResponseWriter, r *http.Request) {
	view, ok := s.queryListStaleness(w, r)
	if !ok {
		return
	}

	render(w, r, staleLive(view.filters, view.staleFor, view.stale))
}

func (s *server) detail(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))

	stop := servertiming.MeasureServerTiming(r.Context(), "query")
	page, found, err := CallDetailPage(r.Context(), s.sys, r.PathValue("kind"), r.PathValue("id"))

	stop()

	if err != nil {
		http.Error(w, "reading the call index failed: "+err.Error(), http.StatusInternalServerError)

		return
	}

	if !found {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)

		render(w, r, notFoundPage(nonce(r), "No indexed call with that id."))

		return
	}

	// The recordings directory is the server's to know: resolve the call's
	// on-disk WAV here so the template can offer the player (the CDR row's
	// RecordingRef is empty by design — FreeSWITCH recordings are local
	// files named after the call uuid, not an indexed reference). Only the
	// uuid-keyed sources are probed: a Telnyx id must not trigger a
	// directory walk for an audio file that naming scheme cannot hold.
	if localRecordingKind(page.Kind) {
		if path, ok := s.localRecordingPath(page.CallView); ok {
			page.LocalRecording = filepath.Base(path)
		}
	}

	// Search deep link (?q=, plan T21): when the query matches transcript
	// turns, one redirect lands the browser on the first match's anchor.
	// The URL fragment never reaches the server, so the at param breaks
	// the would-be redirect loop — its presence means "already jumped".
	if query != "" && !r.URL.Query().Has("at") {
		if first, ok := firstMatchedTurn(page.Transcript, query); ok {
			//nolint:gosec // same-origin by construction, pinned by TestSearchJumpTargetSameOrigin
			http.Redirect(w, r, searchJumpTarget(page.CallView, query, first), http.StatusFound)

			return
		}
	}

	render(w, r, detailRoute(nonce(r), page, query))
}

// staticAsset is one embedded build artifact served verbatim: a missing
// artifact (a build that skipped its codegen step) rejects with the
// asset's own hint, and every response revalidates on each load
// (no-cache) instead of caching stale.
type staticAsset struct {
	path        string
	contentType string
	missingMsg  string
}

func (a staticAsset) serve(w http.ResponseWriter, _ *http.Request) {
	src, err := fs.ReadFile(staticFS, a.path)
	if err != nil {
		http.Error(w, a.missingMsg, http.StatusNotFound)

		return
	}

	w.Header().Set("Content-Type", a.contentType)
	w.Header().Set("Cache-Control", "no-cache")

	_, _ = w.Write(src)
}

// staticRoutes maps each embedded build artifact to its route: the
// Tailwind-built stylesheet (/app.css, produced by the build pipeline —
// source css files are never served), the self-hosted htmx runtime (R9: a
// LAN-bound ops tool must not reach a public CDN for its JavaScript; pinned
// at htmx.org 2.0.10, the exact version templ-components
// layout.DefaultPageProps pins upstream), and the lens-mark favicon
// (layout.Base pins /favicon.svg upstream, else every page load logs a 404
// in the browser console).
var staticRoutes = map[string]staticAsset{ //nolint:gochecknoglobals // declarative route table, read-only
	"GET /app.css": {
		path:        "static/styles.css",
		contentType: "text/css; charset=utf-8",
		missingMsg:  "styles not built (run the tailwind build step)",
	},
	"GET /static/htmx.min.js": {
		path:        "static/htmx.min.js",
		contentType: "text/javascript; charset=utf-8",
		missingMsg:  "htmx runtime missing from the build",
	},
	"GET /static/recent-views.js": {
		path:        "static/recent-views.js",
		contentType: "text/javascript; charset=utf-8",
		missingMsg:  "recent-views script missing from the build",
	},
	"GET /favicon.svg": {
		path:        "static/favicon.svg",
		contentType: "image/svg+xml",
		missingMsg:  "favicon missing from the build",
	},
}

// ops serves the operator counters (R15 poller per-source ingest health,
// R26 etag cache outcomes): offered, recorded, corrected, and rejected per
// source from the latest pass, plus the cache hit ratio and overflow count,
// so ingest and cache health are visible without shelling into the host.
// The route sits behind the same auth middleware as every other route.
func (s *server) opsHandler(w http.ResponseWriter, r *http.Request) {
	var poller any

	if s.ops != nil {
		poller = s.ops()
	}

	var cache *CacheStats

	if s.cache != nil {
		snapshot := s.cache.Snapshot()
		cache = &snapshot
	}

	if poller == nil && cache == nil {
		http.Error(
			w,
			"no ops counters are wired: /api/ops needs a running poller or the etag cache middleware behind it",
			http.StatusServiceUnavailable,
		)

		return
	}

	payload, err := json.Marshal(composeOpsPayload(poller, cache, s.coverageWire(r.Context())))
	if err != nil {
		http.Error(w, "encoding ops counters failed: "+err.Error(), http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	_, _ = w.Write(append(payload, '\n'))
}

// coverageWire is the machine view of one source's index coverage (plan
// P17): what the source delivered and how fresh its newest row is. Stale
// is the gap flag against the coverageStaleAfter SLO ceiling.
type coverageWire struct {
	Kind             string    `json:"kind"`
	Views            int       `json:"views"`
	Oldest           time.Time `json:"oldest"`
	Newest           time.Time `json:"newest"`
	NewestAgeSeconds float64   `json:"newest_age_seconds"`
	Stale            bool      `json:"stale"`
}

// coverageWire computes the JSON coverage section; nil when the walk fails
// (the key is simply absent — consumers treat absence as unknown, never as
// an empty index).
func (s *server) coverageWire(ctx context.Context) []coverageWire {
	rows, err := IngestCoverageQuery(ctx, s.sys, s.now())
	if err != nil {
		s.logf("ingest coverage walk failed: %v", err)

		return nil
	}

	wire := make([]coverageWire, 0, len(rows))

	for _, row := range rows {
		wire = append(wire, coverageWire{
			Kind:             string(row.Kind),
			Views:            row.Views,
			Oldest:           row.Oldest,
			Newest:           row.Newest,
			NewestAgeSeconds: row.NewestAge(s.now()).Seconds(),
			Stale:            row.Stale(s.now()),
		})
	}

	return wire
}

// composeOpsPayload merges the independent counter sources into one
// top-level payload: the poller section keeps the keys it always had
// (sources, last_pass) and the cache section joins as a sibling, so R15
// consumers parse unchanged. Coverage (plan P17) rides as its own key for
// the same reason. A non-map poller payload (custom embeds) gets wrapped
// under "poller" rather than merged.
func composeOpsPayload(poller any, cache *CacheStats, coverage []coverageWire) any {
	if cache == nil && len(coverage) == 0 {
		return poller
	}

	pollerMap, merged := poller.(map[string]any)
	if !merged {
		pollerMap = make(map[string]any, 3)

		if poller != nil {
			pollerMap["poller"] = poller
		}
	} else {
		copied := make(map[string]any, len(pollerMap)+2)
		maps.Copy(copied, pollerMap)
		pollerMap = copied
	}

	if cache != nil {
		pollerMap["cache"] = *cache
	}

	if len(coverage) > 0 {
		pollerMap["coverage"] = coverage
	}

	return pollerMap
}
