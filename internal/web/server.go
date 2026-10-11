package web

import (
	"context"
	"embed"
	"io/fs"
	"net/http"
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
}

// Searcher is the content-query surface the web UI needs.
type Searcher interface {
	Query(ctx context.Context, query string, limit int) ([]search.Hit, error)
}

// Register mounts every pbx-lens route on mux. The handler surface is
// strictly read-only over the read model.
func Register(mux *http.ServeMux, sys *system.System, opts Options) {
	web := &server{sys: sys, recordings: opts.Recordings, search: opts.Search}

	mux.HandleFunc("GET /{$}", web.list)
	mux.HandleFunc("GET /search", web.searchPage)
	mux.HandleFunc("GET /api/freshness", web.freshness)
	mux.HandleFunc("GET /calls/{kind}/{id}", web.detail)
	mux.HandleFunc("GET /calls/{kind}/{id}/recording", web.recording)
	mux.HandleFunc("GET /app.css", serveStyles)
}

type server struct {
	sys        *system.System
	recordings RecordingOptions
	search     Searcher
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

// parseDateParam accepts a bare YYYY-MM-DD (flagged dateOnly, so callers
// can widen the bound to the whole day) or an RFC3339 timestamp.
func parseDateParam(v string) (time.Time, bool, error) {
	if t, err := time.Parse("2006-01-02", v); err == nil {
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
