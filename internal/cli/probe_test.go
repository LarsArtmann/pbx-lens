package cli

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/pbx-lens/internal/calls"
)

func probeConv(t *testing.T, doPreGet bool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	sys, cleanup, err := OpenSystem(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	if doPreGet {
		r := metaengine.NewReader[calls.CallView](sys.MetaEngine(), calls.Collection)
		r.Get(ctx, "telnyx-conversation:pget") // same-key miss before append
	}

	cmd, err := calls.NewRecordCommand(observation("pget", 26))
	if err != nil {
		t.Fatal(err)
	}
	if err := sys.CommandDispatcher().Dispatch(ctx, cmd); err != nil {
		t.Fatal(err)
	}

	if err := DrainProjections(ctx, sys); err != nil {
		t.Fatalf("drain: %v", err)
	}

	r2 := metaengine.NewReader[calls.CallView](sys.MetaEngine(), calls.Collection)
	deadline := time.Now().Add(3 * time.Second)
	for {
		_, found, err := r2.Get(ctx, "telnyx-conversation:pget")
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		if found {
			t.Logf("preGet=%v: CONVERGED", doPreGet)
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("preGet=%v: no convergence in 3s", doPreGet)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestProbeA(t *testing.T) { probeConv(t, false) }
func TestProbeB(t *testing.T) { probeConv(t, true) }
