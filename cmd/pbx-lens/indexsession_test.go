package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/system/v4"
	"github.com/larsartmann/pbx-lens/internal/calls"
)

// TestOpenIndexSessionMemoryAndCleanup covers the one-shot session glue the
// serve tests never touch: the memory DSN boots, the combined cleanup tears
// down in order, and a second cleanup stays safe.
func TestOpenIndexSessionMemoryAndCleanup(t *testing.T) {
	ctx, sys, cleanup, err := openIndexSession("")
	if err != nil {
		t.Fatalf("openIndexSession(memory): %v", err)
	}

	if ctx == nil || sys == nil {
		t.Fatal("openIndexSession returned nil handles")
	}

	if ctx.Err() != nil {
		t.Errorf("session context is already done: %v", ctx.Err())
	}

	cleanup()

	if ctx.Err() == nil {
		t.Error("cleanup must cancel the session context")
	}

	cleanup()
}

// TestFinalizeIngestDrainsAndRebuilds covers the shared ingest tail: drain
// succeeds on a fresh memory index, and a real DSN rebuild opens, rebuilds,
// and closes the search index without leaking errors.
func TestFinalizeIngestDrainsAndRebuilds(t *testing.T) {
	ctx, sys, cleanup, err := openIndexSession("")
	if err != nil {
		t.Fatalf("openIndexSession: %v", err)
	}

	defer cleanup()

	if err := finalizeIngest(ctx, sys, ""); err != nil {
		t.Errorf("finalizeIngest(memory dsn): %v", err)
	}

	if err := finalizeIngest(ctx, sys, filepath.Join(t.TempDir(), "fts.db")); err != nil {
		t.Errorf("finalizeIngest(file dsn): %v", err)
	}
}

// TestGateAndFinalizeIngestsAndFoldsFetchRejects covers the linear shape
// three ingest verbs share: a fetch stub's observations are age-gated in,
// the stream finalizes, and the fetch-level rejections ride the returned
// stats (after finalize, per the T12 order).
func TestGateAndFinalizeIngestsAndFoldsFetchRejects(t *testing.T) {
	db := filepath.Join(t.TempDir(), "gate.db")

	obs := calls.Observation{
		NativeID:   "gate-1",
		Kind:       calls.SourceTelnyxConversation,
		Direction:  calls.DirectionOutbound,
		StartedAt:  time.Now().UTC(),
		FromNumber: "+17287289311",
		ToNumber:   "+15555550123",
	}

	stats := gateAndFinalize(db, 0, func(context.Context, *system.System) ([]calls.Observation, []string, error) {
		return []calls.Observation{obs}, []string{"fetch-note"}, nil
	})

	if stats.Recorded != 1 || stats.Total != 1 {
		t.Errorf("stats = %+v, want 1 recorded", stats)
	}

	if len(stats.Rejected) != 1 || stats.Rejected[0] != "fetch-note" {
		t.Errorf("rejected = %v, want the fetch-level note folded in", stats.Rejected)
	}

	if _, err := os.Stat(db); err != nil {
		t.Errorf("gate session left no db at %s: %v", db, err)
	}
}

// TestGateIngestAgeGateRejects pins the age gate through the cmd glue: an
// observation older than the window is refused, not ingested.
func TestGateIngestAgeGateRejects(t *testing.T) {
	db := filepath.Join(t.TempDir(), "gate-age.db")

	old := calls.Observation{
		NativeID:  "gate-old",
		Kind:      calls.SourceTelnyxConversation,
		Direction: calls.DirectionOutbound,
		StartedAt: time.Now().UTC().Add(-72 * time.Hour),
	}

	stats := gateAndFinalize(
		db,
		24*time.Hour,
		func(context.Context, *system.System) ([]calls.Observation, []string, error) {
			return []calls.Observation{old}, nil, nil
		},
	)

	if stats.Recorded != 0 {
		t.Errorf("stats = %+v, want the over-age observation refused", stats)
	}

	if len(stats.Rejected) == 0 {
		t.Error("age refusal must be reported as a rejection")
	}

	if stats.Total != 1 {
		t.Errorf("total = %d, want 1 (refusal still counts as offered)", stats.Total)
	}
}

// TestPrintRejectedWritesEachReason pins the ingest summary's rejection
// lines (stdout capture, pipe restored).
func TestPrintRejectedWritesEachReason(t *testing.T) {
	original := os.Stdout

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}

	os.Stdout = w

	printRejected([]string{"boom-a", "boom-b"})

	if err := w.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	os.Stdout = original

	var captured bytes.Buffer

	if _, err := captured.ReadFrom(r); err != nil {
		t.Fatalf("read captured stdout: %v", err)
	}

	for _, want := range []string{"rejected: boom-a", "rejected: boom-b"} {
		if !strings.Contains(captured.String(), want) {
			t.Errorf("stdout missing %q (got %q)", want, captured.String())
		}
	}
}

// TestRequireFlagHelpersHappyPaths covers the shared flag validators' happy
// paths (the fatal paths exit the process and belong to the e2e lane).
func TestRequireFlagHelpersHappyPaths(t *testing.T) {
	fs := flag.NewFlagSet("test-verb", flag.ContinueOnError)
	fs.String("db", "", "")

	if err := fs.Parse([]string{"-db", "/tmp/x.db"}); err != nil {
		t.Fatalf("parse: %v", err)
	}

	requireDB(fs, "/tmp/x.db")

	args := flag.NewFlagSet("test-file", flag.ContinueOnError)
	if err := args.Parse([]string{"Master.csv"}); err != nil {
		t.Fatalf("parse: %v", err)
	}

	requireOneArg(args, "CSV file")
}

// TestSmokeSeedsAndBackupVerifies walks two real command flows against a
// temp index: smoke seeds (idempotent double-dispatch included), backup
// snapshots and verifies the result — the wiring the cmd package never
// exercised in-process.
func TestSmokeSeedsAndBackupVerifies(t *testing.T) {
	db := filepath.Join(t.TempDir(), "smoke.db")

	smoke([]string{"-db", db, "-age", "0"})

	if _, err := os.Stat(db); err != nil {
		t.Fatalf("smoke left no db at %s: %v", db, err)
	}

	out := filepath.Join(t.TempDir(), "snapshot.db")
	backupCmd([]string{"-db", db, "-out", out})

	if _, err := os.Stat(out); err != nil {
		t.Fatalf("backup left no snapshot at %s: %v", out, err)
	}
}

// TestIngestCDRDryRunReportsWithoutWrites pins the F63 rehearsal flag: a
// dry-run against a real fixture CSV parses, applies the age gate, and
// prints the would-record report — with NO -db and no index created.
func TestIngestCDRDryRunReportsWithoutWrites(t *testing.T) {
	dir := t.TempDir()
	csv := filepath.Join(dir, "Master.csv")
	row := `"John Doe","17287289311","2000","public","2026-10-09 19:51:25","2026-10-09 19:51:26","2026-10-09 19:51:51","26","25","NORMAL_CLEARING","f0e1d2c3-1111-4aaa-9aaa-000000000001","b0a9-2222",""`

	if err := os.WriteFile(csv, []byte(row+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	original := os.Stdout

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}

	os.Stdout = w

	ingestCDR([]string{"-dry-run", csv})

	if err := w.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	os.Stdout = original

	var captured bytes.Buffer

	if _, err := captured.ReadFrom(r); err != nil {
		t.Fatalf("read captured stdout: %v", err)
	}

	for _, want := range []string{
		"1 rows parsed, 0 parse-skipped, 1 would-record, 0 age-refused",
		"nothing written",
	} {
		if !strings.Contains(captured.String(), want) {
			t.Errorf("dry-run stdout missing %q (got %q)", want, captured.String())
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(entries) != 1 {
		t.Errorf("dry-run must leave no index behind, dir now holds %d entries", len(entries))
	}
}

// TestIngestTelnyxDryRunReportsWithoutWrites pins the T14 rehearsal flag:
// a dry-run against a stub API fetches (the counts ARE the deliverable),
// walks every conversation's messages, and prints the would-fetch report —
// with NO -db, no index created, and no ingest/enrichment side effects.
func TestIngestTelnyxDryRunReportsWithoutWrites(t *testing.T) {
	dir := t.TempDir()

	keyFile := filepath.Join(dir, "key")
	if err := os.WriteFile(keyFile, []byte("test-key\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var messagesWalked []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeStubJSON := func(v any) {
			w.Header().Set("Content-Type", "application/json")

			if err := json.NewEncoder(w).Encode(v); err != nil {
				t.Errorf("encode stub response: %v", err)
			}
		}

		switch {
		case r.URL.Path == "/ai/conversations":
			writeStubJSON(map[string]any{"data": []map[string]any{
				{"id": "conv-1", "created_at": "2026-10-17T10:00:00Z"},
				{"id": "conv-2", "created_at": "2026-10-17T11:00:00Z"},
			}})
		case strings.HasPrefix(r.URL.Path, "/ai/conversations/") && strings.HasSuffix(r.URL.Path, "/messages"):
			messagesWalked = append(messagesWalked, r.URL.Path)
			writeStubJSON(map[string]any{"data": []map[string]any{}})
		case r.URL.Path == "/recordings":
			writeStubJSON(map[string]any{"data": []map[string]any{}})
		default:
			t.Errorf("unexpected stub hit: %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	original := os.Stdout

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}

	os.Stdout = w

	ingestTelnyx([]string{"-dry-run", "-key-file", keyFile, "-api-base", srv.URL, "-since", "48h"})

	if err := w.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	os.Stdout = original

	var captured bytes.Buffer

	if _, err := captured.ReadFrom(r); err != nil {
		t.Fatalf("read captured stdout: %v", err)
	}

	for _, want := range []string{
		"2 conversations fetched, 2 would-record, 0 age-refused, 0 fetch-rejected",
		"nothing written",
	} {
		if !strings.Contains(captured.String(), want) {
			t.Errorf("dry-run stdout missing %q (got %q)", want, captured.String())
		}
	}

	if len(messagesWalked) != 2 {
		t.Errorf("dry-run walked %d conversation message lists, want 2 (the fetch is real)", len(messagesWalked))
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(entries) != 1 {
		t.Errorf("dry-run must leave no index behind, dir now holds %d entries", len(entries))
	}
}
