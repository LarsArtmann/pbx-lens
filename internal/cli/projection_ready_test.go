package cli

import (
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
)

// TestProjectionWorkersReady pins the /health/ready predicate (plan T17):
// only running/live workers keep the serve claiming ready — a worker that
// exhausted its restart budget (failed) or sits in backoff mid-restart
// votes the whole host NOT ready, and startup/shutdown states never pass.
// An empty worker list is vacuously ready (nothing to be not-ready about).
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
			name: "running counts as ready (mid-replay is serving)",
			states: []projectionhost.WorkerState{
				{Name: "call_views", Status: projectionhost.WorkerRunning},
			},
			want: true,
		},
		{
			name: "one failed worker votes not ready",
			states: []projectionhost.WorkerState{
				{Name: "call_views", Status: projectionhost.WorkerLive},
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
		{
			name: "idle (launched, not drained) votes not ready",
			states: []projectionhost.WorkerState{
				{Name: "call_views", Status: projectionhost.WorkerIdle},
			},
			want: false,
		},
		{
			name: "draining (shutdown) votes not ready",
			states: []projectionhost.WorkerState{
				{Name: "call_views", Status: projectionhost.WorkerDraining},
			},
			want: false,
		},
		{
			name: "stopped votes not ready",
			states: []projectionhost.WorkerState{
				{Name: "call_views", Status: projectionhost.WorkerStopped},
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
