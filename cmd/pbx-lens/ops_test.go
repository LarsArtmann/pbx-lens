package main

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/pbx-lens/internal/calls"
	"github.com/larsartmann/pbx-lens/internal/cli"
	"github.com/larsartmann/pbx-lens/internal/retention"
)

func TestRunPurgeDryRunJSONListsCandidates(t *testing.T) {
	// m27.8: -dry-run -json answers with a machine-readable candidate list,
	// and the real purge's JSON report carries what retired.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	sys, cleanup, err := cli.OpenSystem(ctx, filepath.Join(t.TempDir(), "purge-test.db"))
	if err != nil {
		t.Fatalf("OpenSystem: %v", err)
	}

	t.Cleanup(cleanup)

	old := calls.Observation{
		NativeID:     "conv-old",
		Kind:         calls.SourceTelnyxConversation,
		Direction:    calls.DirectionOutbound,
		StartedAt:    time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC),
		FromNumber:   "+17287289311",
		DurationSecs: 42,
	}
	fresh := old
	fresh.NativeID = "conv-fresh"
	fresh.StartedAt = time.Now().UTC()

	if stats := cli.Ingest(ctx, sys, []calls.Observation{old, fresh}); len(stats.Rejected) > 0 {
		t.Fatalf("ingest rejected: %v", stats.Rejected)
	}

	if err := cli.DrainProjections(ctx, sys); err != nil {
		t.Fatalf("drain: %v", err)
	}

	var dry bytes.Buffer

	policy := retention.Policy{MaxAge: 30 * 24 * time.Hour}
	if err := runPurge(ctx, sys, "unused-dsn", policy, true, true, "720h", &dry); err != nil {
		t.Fatalf("runPurge(dry): %v", err)
	}

	var dryReport struct {
		DryRun      bool     `json:"dry_run"`
		OverAge     int      `json:"over_age"`
		WouldRetire []any    `json:"would_retire"`
	}

	if err := json.Unmarshal(dry.Bytes(), &dryReport); err != nil {
		t.Fatalf("dry-run JSON invalid: %v (%s)", err, dry.String())
	}

	if !dryReport.DryRun || dryReport.OverAge != 1 || len(dryReport.WouldRetire) != 1 {
		t.Fatalf("dry-run report = %+v, want dry_run with exactly conv-old", dryReport)
	}

	if !strings.Contains(dry.String(), "conv-old") || strings.Contains(dry.String(), "conv-fresh") {
		t.Errorf("candidate list must name only the over-age call: %s", dry.String())
	}

	var real bytes.Buffer
	if err := runPurge(ctx, sys, "unused-dsn", policy, false, true, "720h", &real); err != nil {
		t.Fatalf("runPurge: %v", err)
	}

	var realReport struct {
		Retired int `json:"retired"`
		OverAge int `json:"over_age"`
	}

	if err := json.Unmarshal(real.Bytes(), &realReport); err != nil {
		t.Fatalf("purge JSON invalid: %v (%s)", err, real.String())
	}

	if realReport.Retired != 1 || realReport.OverAge != 1 {
		t.Fatalf("purge report = %+v, want retired=1 over_age=1", realReport)
	}
}
