// Restore drill (R16): the end-to-end proof that "backups work" — a
// snapshot taken from a live index restores into a fresh DSN, passes the
// integrity gate, and serves the same calls and FTS hits as the source
// after the serve-startup rebuild. This is the scripted drill the README
// backup section describes, executed as a test so it can never rot.
package backup_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/larsartmann/pbx-lens/internal/backup"
	"github.com/larsartmann/pbx-lens/internal/calls"
	"github.com/larsartmann/pbx-lens/internal/cli"
	"github.com/larsartmann/pbx-lens/internal/search"
	"github.com/larsartmann/pbx-lens/internal/web"
)

func TestRestoreDrillSnapshotToFreshIndex(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	sourceDSN := filepath.Join(dir, "source.db")
	snapshot := filepath.Join(dir, "snapshot.db")
	restoredDSN := filepath.Join(dir, "restored.db")

	source, cleanupSource, err := cli.OpenSystem(ctx, sourceDSN)
	if err != nil {
		t.Fatalf("OpenSystem: %v", err)
	}

	t.Cleanup(cleanupSource)

	started := calls.Observation{
		NativeID:     "conv-drill-1",
		Kind:         calls.SourceTelnyxConversation,
		Direction:    calls.DirectionOutbound,
		StartedAt:    time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC),
		DurationSecs: 42,
		FromNumber:   "+17287289311",
		ToNumber:     "+15555550123",
	}

	other := started
	other.NativeID = "conv-drill-2"
	other.StartedAt = time.Date(2026, 10, 9, 13, 0, 0, 0, time.UTC)

	if stats := cli.Ingest(ctx, source, []calls.Observation{started, other}); len(stats.Rejected) > 0 {
		t.Fatalf("ingest rejected: %v", stats.Rejected)
	}

	if err := cli.DrainProjections(ctx, source); err != nil {
		t.Fatalf("drain: %v", err)
	}

	// Drill step 1: snapshot the live index.
	if err := backup.Snapshot(ctx, sourceDSN, snapshot); err != nil {
		t.Fatalf("Snapshot: %v", err)
	}

	// Drill step 2: the staging gate — a snapshot that fails Verify never
	// becomes a restore candidate.
	if err := backup.Verify(ctx, snapshot); err != nil {
		t.Fatalf("Verify: %v", err)
	}

	// Drill step 3: restore = place the snapshot at the fresh DSN.
	payload, err := os.ReadFile(snapshot)
	if err != nil {
		t.Fatalf("read snapshot: %v", err)
	}

	if werr := os.WriteFile(restoredDSN, payload, 0o600); werr != nil {
		t.Fatalf("write restored db: %v", werr)
	}

	// Drill step 4: the restored index opens and serves the same rows.
	restored, cleanupRestored, err := cli.OpenSystem(ctx, restoredDSN)
	if err != nil {
		t.Fatalf("OpenSystem(restored): %v", err)
	}

	t.Cleanup(cleanupRestored)

	views, err := web.ExportCalls(ctx, restored, web.ListCallsQuery{})
	if err != nil {
		t.Fatalf("ExportCalls(restored): %v", err)
	}

	if len(views) != 2 {
		t.Fatalf("restored index holds %d calls, want 2", len(views))
	}

	seen := make(map[string]bool, len(views))
	for _, view := range views {
		seen[string(view.NativeID)] = true
	}

	for _, want := range []string{"conv-drill-1", "conv-drill-2"} {
		if !seen[want] {
			t.Errorf("restored index missing %s", want)
		}
	}

	// Drill step 5: search recovers via the serve-startup rebuild — the
	// snapshot carries no FTS shadow table, the rebuild recreates it.
	index, err := search.Open(ctx, restoredDSN)
	if err != nil {
		t.Fatalf("search.Open(restored): %v", err)
	}

	t.Cleanup(func() { _ = index.Close() })

	if err := cli.RebuildOpenIndex(ctx, restored, index); err != nil {
		t.Fatalf("RebuildOpenIndex(restored): %v", err)
	}

	hits, err := index.Query(ctx, "7289", 10)
	if err != nil {
		t.Fatalf("Query(restored): %v", err)
	}

	if len(hits) != 2 {
		t.Fatalf("restored FTS hit %d calls, want 2", len(hits))
	}
}
