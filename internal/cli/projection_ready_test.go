package cli

import (
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
)

// TestProjectionWorkersReady pins the /health/ready predicate (plan T17).
// It asks "is anything broken?" — never "is a round in flight?". A worker in
// backoff or failed votes NOT ready; idle, running, live, draining and
// stopped are all healthy worker states for a batch-drain host whose poller
// re-drains on every tick (so a healthy serve spends most of its life in
// WorkerStopped — assuming otherwise made /health/ready answer 503 forever,
// caught by scripts/verify-ui-contract.js Contract F).
func TestProjectionWorkersReady(t *testing.T) {
	tests := []struct {
		name   string
		states []projectionhost.WorkerState
		want   bool
	}{
		{
			name:   "no workers is vacuously ready",
			states: nil,
			want:   true,
		},
		{
			name: "all live is ready",
			states: []projectionhost.WorkerState{
				{Name: "call_views", Status: projectionhost.WorkerLive},
				{Name: "fts", Status: projectionhost.WorkerLive, Lag: 2 * time.Millisecond},
			},
			want: true,
		},
		{
			name: "running counts as ready (mid-drain is serving)",
			states: []projectionhost.WorkerState{
				{Name: "call_views", Status: projectionhost.WorkerRunning},
			},
			want: true,
		},
		{
			name: "idle (registered, not yet started) is ready",
			states: []projectionhost.WorkerState{
				{Name: "call_views", Status: projectionhost.WorkerIdle},
			},
			want: true,
		},
		{
			name: "draining (finishing in-flight events) is ready",
			states: []projectionhost.WorkerState{
				{Name: "call_views", Status: projectionhost.WorkerDraining},
			},
			want: true,
		},
		{
			name: "stopped (drain complete, waiting for next tick) is ready",
			states: []projectionhost.WorkerState{
				{Name: "call_views", Status: projectionhost.WorkerStopped},
				{Name: "fts", Status: projectionhost.WorkerStopped},
			},
			want: true,
		},
		{
			name: "one failed worker votes not ready",
			states: []projectionhost.WorkerState{
				{Name: "call_views", Status: projectionhost.WorkerStopped},
				{Name: "fts", Status: projectionhost.WorkerFailed, LastError: "boom"},
			},
			want: false,
		},
		{
			name: "backoff mid-restart votes not ready",
			states: []projectionhost.WorkerState{
				{Name: "fts", Status: projectionhost.WorkerBackoff},
			},
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := ProjectionWorkersReady(tc.states); got != tc.want {
				t.Errorf("ProjectionWorkersReady(%v) = %v, want %v", tc.states, got, tc.want)
			}
		})
	}
}
