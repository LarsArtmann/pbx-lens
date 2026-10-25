package cli

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// openTestSystem boots the shared memory-engine system for CLI tests and
// wires the cleanup — the one boot idiom every ingest/enrich/poller test
// used to repeat.
func openTestSystem(t *testing.T) *system.System {
	t.Helper()

	sys, cleanup, err := OpenSystem(context.Background(), "")
	if err != nil {
		t.Fatalf("OpenSystem: %v", err)
	}

	t.Cleanup(cleanup)

	return sys
}
