package cli

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/pbx-lens/internal/app"
	"github.com/larsartmann/pbx-lens/internal/calls"
)

func TestProbeLatency(t *testing.T) {
	for i := 0; i < 3; i++ {
		t.Run("iter", func(t *testing.T) {
			start := time.Now()
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			sys, err := app.New(ctx, "")
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = sys.Close() }()

			cmd, err := calls.NewRecordCommand(observation("lat", 26))
			if err != nil {
				t.Fatal(err)
			}
			if err := sys.CommandDispatcher().Dispatch(ctx, cmd); err != nil {
				t.Fatal(err)
			}

			if err := sys.ProjectionHost().Start(ctx); err != nil {
				t.Fatalf("start: %v", err)
			}

			reader := metaengine.NewReader[calls.CallView](sys.MetaEngine(), calls.Collection)
			var convergedAt time.Duration
			deadline := time.Now().Add(20 * time.Second)

			for {
				_, found, err := reader.Get(ctx, "telnyx-conversation:lat")
				if err != nil {
					t.Fatalf("get: %v", err)
				}
				if found {
					convergedAt = time.Since(start)
					break
				}
				if time.Now().After(deadline) {
					break
				}
				time.Sleep(5 * time.Millisecond)
			}

			if convergedAt == 0 {
				t.Fatalf("no convergence in 20s (native id collision across iters is fine)")
			}
			t.Logf("converged after %v", convergedAt)
		})
	}
}
