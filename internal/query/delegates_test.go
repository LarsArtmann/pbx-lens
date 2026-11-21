package query

import (
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/system/v4"
	"github.com/larsartmann/pbx-lens/internal/calls"
	"github.com/larsartmann/pbx-lens/internal/query/querytest"
)

// testCallStarted and newTestSystem delegate to the shared querytest
// fixtures — one copy serves the query, web, and serve suites.
func testCallStarted() time.Time { return querytest.CallStarted() }

func newTestSystem(t *testing.T, observations ...calls.Observation) *system.System {
	t.Helper()

	return querytest.System(t, observations...)
}

func webObservation(nativeID string, started time.Time) calls.Observation {
	return querytest.Observation(nativeID, started)
}

func aiCallFixture(nativeID string, started time.Time) (calls.Observation, calls.Observation) {
	return querytest.AICallPair(nativeID, started)
}

func aiAgentCDR() calls.Observation        { return querytest.AgentCDR() }
func aiAgentTranscript() calls.Observation { return querytest.AgentTranscript() }
