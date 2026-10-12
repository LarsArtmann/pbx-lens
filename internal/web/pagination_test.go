package web

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/pbx-lens/internal/calls"
)

// seedCalls fans out n observations, one per hour ending at base, so the
// keyset order is total (no timestamp ties across page boundaries).
func seedCalls(n int, base time.Time) []calls.Observation {
	out := make([]calls.Observation, 0, n)

	for i := range n {
		out = append(out, webObservation(fmt.Sprintf("call-%03d", i), base.Add(time.Duration(i)*time.Hour)))
	}

	return out
}

// extractHref returns the href value of the first anchor whose text matches
// (the pager buttons render as anchors with visible text).
func extractHref(t *testing.T, body, text string) string {
	t.Helper()

	anchor := fmt.Sprintf(`>%s</a>`, text)

	before, _, ok := strings.CutLast(body, anchor)
	if !ok {
		return ""
	}

	segment := before

	_, after0, ok0 := strings.CutLast(segment, `href="`)
	if !ok0 {
		return ""
	}

	rest := after0
	end := strings.Index(rest, `"`)

	return rest[:end]
}

// TestPagerSinglePageRendersNothing pins the exactly-one-page edge: with
// fewer rows than a page there is no pager at all — disabled buttons would
// only add noise to a list that is complete.
func TestPagerSinglePageRendersNothing(t *testing.T) {
	seeded := seedCalls(3, time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC))
	mux := newTestServerWithSearch(t, seeded...)

	_, body := getBody(t, mux, "/")
	if strings.Contains(body, `aria-label="Pagination"`) {
		t.Error("single page should not render a pagination nav")
	}
}

// TestPagerWalksBothDirections covers the deep-page edges and the noscript
// walk: page 1 offers only Older, the walked page offers Newer back to the
// same content, and the last page offers only Newer.
func TestPagerWalksBothDirections(t *testing.T) {
	seeded := seedCalls(pageSize+5, time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC))
	mux := newTestServerWithSearch(t, seeded...)

	status, body := getBody(t, mux, "/")
	if status != http.StatusOK {
		t.Fatalf("GET / status = %d, want 200", status)
	}

	if !strings.Contains(body, "Older calls") {
		t.Fatal("page 1 missing Older calls affordance")
	}

	if strings.Contains(body, `href="/?after=`) {
		t.Error("page 1 must not offer a Newer link (head of walk)")
	}

	olderHref := extractHref(t, body, "Older calls")
	if !strings.HasPrefix(olderHref, "/?before=") {
		t.Fatalf("Older href = %q, want a before-cursor URL", olderHref)
	}

	// Walk to the deep page, noscript-style: a plain GET of the href.
	status, deep := getBody(t, mux, olderHref)
	if status != http.StatusOK {
		t.Fatalf("GET %s status = %d, want 200", olderHref, status)
	}

	if !strings.Contains(deep, "Newer calls") {
		t.Fatal("deep page missing Newer calls affordance")
	}

	newerHref := extractHref(t, deep, "Newer calls")
	if !strings.HasPrefix(newerHref, "/?after=") {
		t.Fatalf("Newer href = %q, want an after-cursor URL", newerHref)
	}

	if strings.Contains(deep, `href="/?before=`) {
		t.Error("last page must not offer an Older link")
	}

	// The walk back lands on the head page: same newest row, no Newer link.
	status, back := getBody(t, mux, newerHref)
	if status != http.StatusOK {
		t.Fatalf("GET %s status = %d, want 200", newerHref, status)
	}

	if !strings.Contains(back, "call-054") {
		t.Error("walk back should land on the head page with the newest call")
	}

	if strings.Contains(back, `href="/?after=`) {
		t.Error("walked-back head page must not offer a Newer link")
	}
}

// TestPagerPreservesFilters pins the filter contract: a filtered page's
// Older/Newer hrefs carry the same filters so the walk stays filtered.
func TestPagerPreservesFilters(t *testing.T) {
	seeded := seedCalls(2*(pageSize+5), time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC))

	for i := range seeded {
		if i%2 == 0 {
			seeded[i].Direction = calls.DirectionInbound
		}
	}

	mux := newTestServerWithSearch(t, seeded...)

	_, body := getBody(t, mux, "/?direction=inbound")

	olderHref := extractHref(t, body, "Older calls")
	if !strings.Contains(olderHref, "direction=inbound") || !strings.Contains(olderHref, "before=") {
		t.Errorf("filtered Older href = %q, want direction filter + cursor", olderHref)
	}

	status, deep := getBody(t, mux, olderHref)
	if status != http.StatusOK {
		t.Fatalf("GET %s status = %d, want 200", olderHref, status)
	}

	newerHref := extractHref(t, deep, "Newer calls")
	if !strings.Contains(newerHref, "direction=inbound") || !strings.Contains(newerHref, "after=") {
		t.Errorf("filtered Newer href = %q, want direction filter + cursor", newerHref)
	}
}
