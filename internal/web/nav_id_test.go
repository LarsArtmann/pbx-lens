package web

import (
	"net/http"
	"regexp"
	"strings"
	"testing"
)

// The nav-id determinism gate (plan M8/F28): with the nav's BaseProps.ID
// empty, the library's utils.EnsureID mints a RANDOM id per render, every
// page body is unique, and the ETag/no-cache revalidation contract can never
// 304 — every navigation and htmx poll pays a full 200. The shell pins
// ID="pbx-lens-nav"; this gate fails if the id is ever randomized again
// (library default restored, prop dropped, or a second random id added).
func TestNavIDIsDeterministic(t *testing.T) {
	mux := newTestServer(t)

	status, body := getBody(t, mux, "/")
	if status != http.StatusOK {
		t.Fatalf("GET / status = %d", status)
	}

	if !strings.Contains(body, `aria-controls="pbx-lens-nav"`) {
		t.Error("page does not carry the pinned deterministic nav id — ETag revalidation will silently degrade to full 200s")
	}

	randomID := regexp.MustCompile(`tc-mobile-menu-[0-9a-f]{16}`)
	if loc := randomID.FindStringIndex(body); loc != nil {
		t.Errorf("randomized mobile-menu id at offset %d", loc[0])
	}
}
