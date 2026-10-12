package web

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/pbx-lens/internal/calls"
)

// TestOneFlowFromSeedToDetail is the whole user loop in one flow: seed two
// calls, list them, filter, open a detail page, search a transcript phrase,
// and hit the freshness/staleness surface — the routes the serve binary
// composes, exercised in the order an operator walks them. Auth gating is
// layered on at the end (one middleware wraps the same mux).
func TestOneFlowFromSeedToDetail(t *testing.T) {
	started := testCallStarted()

	inbound := webObservation("conv-in", started.Add(time.Minute))
	inbound.Direction = calls.DirectionInbound

	seeded := []calls.Observation{webObservation("conv-out", started), inbound}

	mux := newTestServerWithSearch(t, seeded...)

	// 1. Seed visible on the list page, oldest-date index flags stale.
	status, body := getBody(t, mux, "/")
	if status != http.StatusOK {
		t.Fatalf("GET / status = %d, want 200", status)
	}

	for _, want := range []string{"conv-out", "conv-in", "Index may be stale"} {
		if !strings.Contains(body, want) {
			t.Errorf("list page missing %q", want)
		}
	}

	// 2. Direction filter narrows the same data.
	status, body = getBody(t, mux, "/?direction=inbound")
	if status != http.StatusOK {
		t.Fatalf("GET filtered list status = %d, want 200", status)
	}

	if !strings.Contains(body, "conv-in") || strings.Contains(body, "conv-out") {
		t.Error("inbound filter should keep conv-in and hide conv-out")
	}

	// 3. Detail page serves the full view of one call.
	status, body = getBody(t, mux, "/calls/telnyx-conversation/conv-out")
	if status != http.StatusOK {
		t.Fatalf("GET detail status = %d, want 200", status)
	}

	for _, want := range []string{"conv-out", "+17287289311", "hello there"} {
		if !strings.Contains(body, want) {
			t.Errorf("detail page missing %q", want)
		}
	}

	// 4. Search finds the transcript phrase and deep-links the detail page.
	status, body = getBody(t, mux, "/search?q=hello")
	if status != http.StatusOK {
		t.Fatalf("GET /search status = %d, want 200", status)
	}

	if !strings.Contains(body, "/calls/telnyx-conversation/conv-out") {
		t.Error("search must deep-link the matching call")
	}

	// 5. Freshness endpoint agrees with what the list page just showed.
	status, body = getBody(t, mux, "/api/freshness")
	if status != http.StatusOK || !strings.Contains(body, "age_seconds") {
		t.Fatalf("GET /api/freshness = %d (%s), want 200 with age", status, body)
	}

	// 6. The auth gate wraps the same surface: refused without credentials,
	// served with them.
	htpasswdPath := writeHtpasswd(t, "lars", authTestPassword)

	gate, err := HtpasswdMiddleware(htpasswdPath)
	if err != nil {
		t.Fatalf("HtpasswdMiddleware: %v", err)
	}

	gated := gate(mux)

	if rec := getWithBasicAuth(gated, "/", "", ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status = %d, want 401", rec.Code)
	}

	if rec := getWithBasicAuth(
		gated,
		"/calls/telnyx-conversation/conv-out",
		"lars",
		authTestPassword,
	); rec.Code != http.StatusOK {
		t.Fatalf("authenticated detail status = %d, want 200", rec.Code)
	}
}
