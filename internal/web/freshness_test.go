package web

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

func TestFreshnessReportsIndexState(t *testing.T) {
	started := time.Date(2026, 10, 9, 19, 51, 25, 0, time.UTC)
	mux := newTestServer(t,
		webObservation("conv-a", started),
		webObservation("conv-b", started.Add(time.Minute)),
	)

	status, body := getBody(t, mux, "/api/freshness")
	if status != http.StatusOK {
		t.Fatalf("GET /api/freshness status = %d, want 200", status)
	}

	var fresh Freshness
	if err := json.Unmarshal([]byte(body), &fresh); err != nil {
		t.Fatalf("parse freshness JSON: %v (body: %s)", err, body)
	}

	if fresh.CallsIndexed != 2 {
		t.Errorf("calls_indexed = %d, want 2", fresh.CallsIndexed)
	}

	if !fresh.NewestStartedAt.Equal(started.Add(time.Minute)) {
		t.Errorf("newest_started_at = %v, want %v", fresh.NewestStartedAt, started.Add(time.Minute))
	}

	if !fresh.Stale {
		t.Error("a 2026-10-09 index is far past the 6h threshold; stale must be true")
	}

	if fresh.AgeSeconds <= 0 {
		t.Errorf("age_seconds = %f, want a positive age", fresh.AgeSeconds)
	}
}

func TestFreshnessOnEmptyIndexIsNotStale(t *testing.T) {
	status, body := getBody(t, newTestServer(t), "/api/freshness")
	if status != http.StatusOK {
		t.Fatalf("GET /api/freshness status = %d, want 200", status)
	}

	var fresh Freshness
	if err := json.Unmarshal([]byte(body), &fresh); err != nil {
		t.Fatalf("parse freshness JSON: %v", err)
	}

	if fresh.Stale {
		t.Error("an empty index must not report stale (the empty state explains itself)")
	}

	if fresh.CallsIndexed != 0 {
		t.Errorf("calls_indexed = %d, want 0", fresh.CallsIndexed)
	}
}
