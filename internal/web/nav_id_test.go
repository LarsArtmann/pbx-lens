package web

import (
	"net/http"
	"regexp"
	"slices"
	"testing"
)

// The nav-id determinism gate (plan M8/F28, rewritten 2026-11-20): a nav id
// that varies per render makes every page body unique, so the ETag
// revalidation contract can never 304 — every navigation and htmx poll pays
// a full 200. The library used to mint a random utils.EnsureID id per
// render, and the shell pinned ID="pbx-lens-nav" as the workaround;
// templ-components v1.27.x derives a CONTENT-HASHED menu id instead, so the
// pin retired. This gate renders the page twice and fails if any
// aria-controls id ever varies between renders.
func TestNavIDIsDeterministic(t *testing.T) {
	mux := newTestServer(t)

	status, first := getBody(t, mux, "/")
	if status != http.StatusOK {
		t.Fatalf("GET / status = %d", status)
	}

	_, second := getBody(t, mux, "/")

	ids := func(body string) []string {
		matches := regexp.MustCompile(`aria-controls="([^"]+)"`).FindAllStringSubmatch(body, -1)

		values := make([]string, 0, len(matches))
		for _, match := range matches {
			values = append(values, match[1])
		}

		return values
	}

	firstIDs, secondIDs := ids(first), ids(second)
	if len(firstIDs) == 0 {
		t.Fatal("no aria-controls id on the nav; the mobile-menu hook is gone")
	}

	if !slices.Equal(firstIDs, secondIDs) {
		t.Errorf(
			"nav ids vary per render: %v vs %v — ETag revalidation would degrade to full 200s",
			firstIDs,
			secondIDs,
		)
	}
}
