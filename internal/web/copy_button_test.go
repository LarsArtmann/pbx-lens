package web

import (
	"net/http"
	"strings"
	"testing"
)

// TestDetailCopyButtons verifies the T20 copy affordances: the native id
// (PageHeader action) and both facts numbers ride data-tc-copy on BUTTON
// elements — the library's href/anchor variant is deliberately unused here,
// so any anchor carrying the attribute is a regression.
func TestDetailCopyButtons(t *testing.T) {
	started := testCallStarted()
	mux := newTestServerWithSearch(t, webObservation("conv-copy", started))

	status, body := getBody(t, mux, "/calls/telnyx-conversation/conv-copy")
	if status != http.StatusOK {
		t.Fatalf("GET detail status = %d, want 200", status)
	}

	for _, want := range []string{
		`data-tc-copy="conv-copy"`,
		`data-tc-copy="+17287289311"`,
		`data-tc-copy="+15555550123"`,
		`aria-label="Copy call ID conv-copy"`,
		`aria-label="Copy from number"`,
		`aria-label="Copy to number"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("detail page missing %q", want)
		}
	}

	buttons := strings.Count(body, "<button type=\"button\" data-tc-copy=")
	if buttons != 3 {
		t.Errorf("copy buttons rendered = %d, want 3 (id + two numbers)", buttons)
	}

	if strings.Contains(body, "<a href=\"") && strings.Contains(body, "data-tc-copy=") {
		// Only a regression if the SAME element carries both; the page has
		// ordinary links (breadcrumbs, rows), so scan element-locally.
		for chunk := range strings.SplitSeq(body, "<a ") {
			if strings.Contains(strings.SplitN(chunk, ">", 2)[0], "data-tc-copy=") {
				t.Error("anchor copy variant rendered; button variant is the contract")
			}
		}
	}
}
