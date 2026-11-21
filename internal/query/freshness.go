package query

import "time"

// Freshness is the /api/freshness payload: one honest glance at how
// current the index is (C2). The machine-readable twin of the staleness
// banner — monitors and the dashboard read this, humans read the banner.
type Freshness struct {
	NewestStartedAt time.Time `json:"newest_started_at"`
	AgeSeconds      float64   `json:"age_seconds"`
	Stale           bool      `json:"stale"`
	CallsIndexed    int       `json:"calls_indexed"`
}
