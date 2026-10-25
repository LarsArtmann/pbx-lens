package web

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/a-h/templ"
	"github.com/larsartmann/httputil"
	servertiming "github.com/larsartmann/httputil/server_timing"
	"github.com/larsartmann/pbx-lens/internal/calls"
	"github.com/larsartmann/pbx-lens/internal/search"
)

// nonce pulls the per-request CSP nonce for template rendering; empty when
// nonce mode is off (CSP_NONCE unset).
func nonce(r *http.Request) string {
	return httputil.NonceFromRequest(r)
}

// searchModeTranscript selects the per-turn result view (?mode=transcript):
// "what was SAID" — one ranked hit per matching turn, anchored.
const searchModeTranscript = "transcript"

// searchModeOf reads the requested result mode; anything but the
// transcript value is the call view, so a garbage mode degrades to the
// default instead of erroring.
func searchModeOf(r *http.Request) string {
	if r.URL.Query().Get("mode") == searchModeTranscript {
		return searchModeTranscript
	}

	return ""
}

// searchPage serves the FTS content-search page (B3/m28-m29). The page size
// is search.DefaultLimit — one source of truth for what one result page is.
func (s *server) searchPage(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	mode := searchModeOf(r)

	if query == "" {
		render(w, r, searchRoute(nonce(r), "", nil, nil, "", mode))

		return
	}

	if s.search == nil {
		http.Error(
			w,
			"search needs a sqlite index: restart serve with -db <file> (memory deployments have no FTS)",
			http.StatusServiceUnavailable,
		)

		return
	}

	stop := servertiming.MeasureServerTiming(r.Context(), "fts")

	hits, turnHits, err := s.runSearch(r.Context(), query, mode)

	stop()

	if err != nil {
		http.Error(w, "search failed: "+err.Error(), http.StatusBadRequest)

		return
	}

	render(w, r, searchRoute(nonce(r), query, hits, turnHits, "", mode))
}

// runSearch dispatches by mode: the call view (with the substring-number
// merge) or the per-turn view. Exactly one slice is non-nil.
func (s *server) runSearch(ctx context.Context, query, mode string) ([]search.Hit, []search.TurnHit, error) {
	if mode == searchModeTranscript {
		turnHits, err := s.search.QueryTurns(ctx, query, search.DefaultLimit)

		return nil, turnHits, err
	}

	hits, err := s.search.Query(ctx, query, search.DefaultLimit)

	return hits, nil, err
}

// detailURL is the canonical detail deep link for one indexed call — the
// same path list rows, search hits, and the search-redirect (T21) build on.
func detailURL(kind calls.SourceKind, id calls.NativeID) string {
	return "/calls/" + url.PathEscape(string(kind)) + "/" + url.PathEscape(string(id))
}

// hitHref deep-links a search hit to its call page, carrying the query
// along (plan T21): the detail page re-runs the match server-side, marks
// the matching turns, and redirects once to the first match's anchor, so
// the click lands at the evidence — no JS required.
func hitHref(hit search.Hit, query string) string {
	path := detailURL(hit.Kind, hit.NativeID)
	if query == "" {
		return path
	}

	return path + "?q=" + url.QueryEscape(query)
}

// searchJumpTarget is the one redirect the search deep link produces: the
// detail page re-served with the query plus the first match's turn anchor.
// Same-origin by construction — fixed /calls/ prefix, path-escaped identity,
// query-escaped search text — and unit-tested on exactly that property.
func searchJumpTarget(view calls.CallView, query string, firstTurn int) string {
	return detailURL(view.Kind, view.NativeID) +
		"?q=" + url.QueryEscape(query) +
		"&at=" + strconv.Itoa(firstTurn) +
		"#turn-" + strconv.Itoa(firstTurn)
}

// turnHitHref deep-links a transcript-turn hit to the exact turn: the same
// ?q=&at=#turn-N contract the call-mode jump builds, but anchored at the
// hit's own turn instead of the call's first match.
func turnHitHref(hit search.TurnHit, query string) string {
	return detailURL(hit.Kind, hit.NativeID) +
		"?q=" + url.QueryEscape(query) +
		"&at=" + strconv.Itoa(hit.TurnIndex) +
		"#turn-" + strconv.Itoa(hit.TurnIndex)
}

// hitCardClass is the shared result-card styling for both search modes;
// the focus-visible ring is the keyboard-walk contract the UI probe
// asserts.
func hitCardClass() string {
	return "block rounded-lg border border-gray-200 p-4 transition-colors motion-reduce:transition-none hover:bg-gray-100 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-tc-primary dark:border-gray-800 dark:hover:bg-gray-800 dark:focus-visible:outline-tc-info"
}

// searchTabClass is the mode-switch pill styling; variants arrive as KV
// parts so exactly one of active/inactive renders.
func searchTabClass(variants ...templ.KeyValue[string, bool]) templ.CSSClasses {
	const pillBase = "rounded-full border px-3 py-1 transition-colors motion-reduce:transition-none focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-tc-primary dark:focus-visible:outline-tc-info"

	parts := make(templ.CSSClasses, 0, 1+len(variants))
	parts = append(parts, pillBase)

	for _, variant := range variants {
		parts = append(parts, variant)
	}

	return parts
}
