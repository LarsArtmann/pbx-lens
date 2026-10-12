package web

import (
	"context"
	"embed"
	"encoding/json/v2"
	"io/fs"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/larsartmann/go-cqrs-lite/system/v4"
	errorfamily "github.com/larsartmann/go-error-family"
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
	// (R15). Nil disables the endpoint with an explanatory 503 (one-shot
	// ingests have no live counters to answer with).
	Ops func() any
}

// Searcher is the content-query surface the web UI needs.
type Searcher interface {
	Query(ctx context.Context, query string, limit int) ([]search.Hit, error)
}

// Register mounts every pbx-lens route on mux. The handler surface is
// strictly read-only over the read model.
func Register(mux *http.ServeMux, sys *system.System, opts Options) {
	timeout := resolveUpstreamTimeout(opts.Recordings.UpstreamTimeout)

	web := &server{
		sys:        sys,
		recordings: opts.Recordings,
		search:     opts.Search,
		log:        opts.Log,
		ops:        opts.Ops,
		recTimeout: timeout,
		upstream:   newUpstreamClient(timeout),
	}

	mux.HandleFunc("GET /{$}", web.list)
	mux.HandleFunc("GET /dashboard", web.dashboard)
	mux.HandleFunc("GET /frag/dashboard", web.dashboardFragment)
	mux.HandleFunc("GET /frag/staleness", web.stalenessFragment)
	mux.HandleFunc("GET /search", web.searchPage)
	mux.HandleFunc("GET /api/freshness", web.freshness)
	mux.HandleFunc("GET /api/ops", web.opsHandler)
	mux.HandleFunc("GET /api/export.jsonl", web.exportJSONL)
	mux.HandleFunc("GET /api/export.csv", web.exportCSV)
	mux.HandleFunc("GET /calls/{kind}/{id}", web.detail)
	mux.HandleFunc("GET /calls/{kind}/{id}/recording", web.recording)
	mux.HandleFunc("GET /app.css", serveStyles)
	mux.HandleFunc("GET /static/htmx.min.js", serveHtmx)
}

type server struct {
	sys        *system.System
	recordings RecordingOptions
	search     Searcher
	log        func(msg string, args ...any)
	ops        func() any
	recTimeout time.Duration
	upstream   *http.Client
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

func (s *server) list(w http.ResponseWriter, r *http.Request) {
	filters, err := parseFilters(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	page, err := ListCalls(r.Context(), s.sys, filters)
	if err != nil {
		http.Error(w, "reading the call index failed: "+err.Error(), http.StatusInternalServerError)

		return
	}

	staleFor, stale := staleness(filters, page)

	render(w, r, listRoute(page, filters, staleFor, stale))
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

	for _, field := range []struct {
		name   string
		target *time.Time
	}{
		{name: "from", target: &query.From},
		{name: "to", target: &query.To},
		{name: "before", target: &query.Before},
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
// filtered view (a deliberate slice) never carries a global-freshness
// warning. An empty index is never stale (the empty state explains itself).
func staleness(filters ListCallsQuery, page CallListPage) (time.Duration, bool) {
	if filters != (ListCallsQuery{}) || len(page.Rows) == 0 {
		return 0, false
	}

	age := time.Since(page.Rows[0].When)

	return age, age > staleAfter
}

// dashboard serves the quality-dashboard page (T25). The from/to picker
// narrows the underlying scan; the cards region polls for updates (R9).
func (s *server) dashboard(w http.ResponseWriter, r *http.Request) {
	filters, fragURL, ok := parseRange(w, r)
	if !ok {
		return
	}

	stats, err := DashboardStatsQuery(r.Context(), s.sys, time.Now(), filters)
	if err != nil {
		http.Error(w, "reading the call index failed: "+err.Error(), http.StatusInternalServerError)

		return
	}

	render(w, r, dashboardRoute(stats, time.Now(), fragURL))
}

// dashboardFragment is the polled region payload (R9): the same live cards
// the page renders, without the shell — the region replaces itself.
func (s *server) dashboardFragment(w http.ResponseWriter, r *http.Request) {
	filters, fragURL, ok := parseRange(w, r)
	if !ok {
		return
	}

	stats, err := DashboardStatsQuery(r.Context(), s.sys, time.Now(), filters)
	if err != nil {
		http.Error(w, "reading the call index failed: "+err.Error(), http.StatusInternalServerError)

		return
	}

	render(w, r, dashboardLive(stats, time.Now(), fragURL))
}

// parseRange reads the dashboard's date-range picker (from/to only — list
// filters like direction/ai are not dashboard concepts and stay ignored)
// and derives the polled-region URL carrying the same range. Garbage dates
// reject with a 400, house style.
func parseRange(w http.ResponseWriter, r *http.Request) (ListCallsQuery, string, bool) {
	var filters ListCallsQuery

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

			return ListCallsQuery{}, "", false
		}

		// A bare upper bound covers the whole day it names (same semantics
		// as the list filters).
		if field.name == "to" {
			parsed = parsed.AddDate(0, 0, 1).Add(-time.Second)
		}

		*field.target = parsed
	}

	fragURL := "/frag/dashboard"

	if !filters.From.IsZero() || !filters.To.IsZero() {
		query := url.Values{}

		if !filters.From.IsZero() {
			query.Set("from", filters.From.Format(dayFormat))
		}

		if !filters.To.IsZero() {
			// To was widened to end-of-day; the date format drops the time,
			// so the picker echo shows the day the operator picked.
			query.Set("to", filters.To.Format(dayFormat))
		}

		fragURL += "?" + query.Encode()
	}

	return filters, fragURL, true
}

// stalenessFragment re-checks the freshness banner (R9): the polled region
// re-renders the banner, or nothing once the index is fresh again.
func (s *server) stalenessFragment(w http.ResponseWriter, r *http.Request) {
	filters, err := parseFilters(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	page, err := ListCalls(r.Context(), s.sys, filters)
	if err != nil {
		http.Error(w, "reading the call index failed: "+err.Error(), http.StatusInternalServerError)

		return
	}

	staleFor, stale := staleness(filters, page)

	render(w, r, staleLive(staleFor, stale))
}

func (s *server) detail(w http.ResponseWriter, r *http.Request) {
	view, found, err := CallDetail(r.Context(), s.sys, r.PathValue("kind"), r.PathValue("id"))
	if err != nil {
		http.Error(w, "reading the call index failed: "+err.Error(), http.StatusInternalServerError)

		return
	}

	if !found {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)

		render(w, r, notFoundPage("No indexed call with that id."))

		return
	}

	render(w, r, detailRoute(view))
}

// serveStyles serves the Tailwind-built stylesheet (styles.css, produced by
// the build pipeline). Source css files are never served.
func serveStyles(w http.ResponseWriter, _ *http.Request) {
	styles, err := fs.ReadFile(staticFS, "static/styles.css")
	if err != nil {
		http.Error(w, "styles not built (run the tailwind build step)", http.StatusNotFound)

		return
	}

	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(styles)
}

// serveHtmx serves the self-hosted htmx runtime (R9): the polled regions
// need it, and a LAN-bound ops tool must not reach a public CDN for its
// JavaScript. Pinned at htmx.org 2.0.10 — the exact version templ-components
// layout.DefaultPageProps pins upstream.
func serveHtmx(w http.ResponseWriter, _ *http.Request) {
	src, err := fs.ReadFile(staticFS, "static/htmx.min.js")
	if err != nil {
		http.Error(w, "htmx runtime missing from the build", http.StatusNotFound)

		return
	}

	w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")

	_, _ = w.Write(src)
}

// ops serves the poller's per-source ingest counters (R15): offered,
// recorded, corrected, and rejected per source from the latest pass, so
// ingest health is visible without shelling into the host. The route sits
// behind the same auth middleware as every other route.
func (s *server) opsHandler(w http.ResponseWriter, _ *http.Request) {
	if s.ops == nil {
		http.Error(
			w,
			"ops counters are only served while the poller runs (start serve with polling configured)",
			http.StatusServiceUnavailable,
		)

		return
	}

	payload, err := json.Marshal(s.ops())
	if err != nil {
		http.Error(w, "encoding ops counters failed: "+err.Error(), http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	_, _ = w.Write(append(payload, '\n'))
}
