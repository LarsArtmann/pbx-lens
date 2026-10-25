package app_test

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	sqliteengine "github.com/larsartmann/go-cqrs-lite/metaengine/sqliteengine/v4"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// SCRATCH verification for the 2026-10-25 wave: prove metaengine v4.20.0's
// layout-stamp drift WARN actually fires for a same-names/changed-TYPES plan
// across two boots of one file-backed engine. Deleted after the run (M09).

type scratchInput struct{}

type scratchString struct {
	ID       string
	Priority string
}

type scratchInt struct {
	ID       string
	Priority int
}

type scratchEvent struct {
	ID       string
	Priority int
}

func declString() metaengine.QueryDecl[scratchInput, scratchString] {
	return metaengine.Query[scratchInput, scratchString](
		"scratch_drift",
		metaengine.OnRecord(
			scratchEvent{},
			func(_ record.Record, e scratchEvent) (string, scratchString) {
				return e.ID, scratchString{ID: e.ID, Priority: "x"}
			},
		),
	)
}

func declInt() metaengine.QueryDecl[scratchInput, scratchInt] {
	return metaengine.Query[scratchInput, scratchInt](
		"scratch_drift",
		metaengine.OnRecord(
			scratchEvent{},
			func(_ record.Record, e scratchEvent) (string, scratchInt) {
				return e.ID, scratchInt{ID: e.ID, Priority: 1}
			},
		),
	)
}

func openFileEngine(t *testing.T, dsn string) metaengine.Engine {
	t.Helper()

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	eng, err := sqliteengine.NewSQLiteEngine(db)
	if err != nil {
		t.Fatalf("NewSQLiteEngine: %v", err)
	}
	t.Cleanup(func() { _ = eng.Close() })

	return eng
}

func mustBoot(t *testing.T, eng metaengine.Engine, args ...any) {
	t.Helper()

	store, err := metaengine.Plan([]metaengine.Engine{eng}, args...)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	_ = ctx()
}

func ctx() context.Context { return context.Background() }

func TestScratchLayoutStampDriftWarns(t *testing.T) {
	dsn := filepath.Join(t.TempDir(), fmt.Sprintf("drift_%d.db", time.Now().UnixNano()))

	mustBoot(t, openFileEngine(t, dsn), declString())

	buf := &bytes.Buffer{}
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelWarn})))
	defer slog.SetDefault(previous)

	mustBoot(t, openFileEngine(t, dsn), declInt())

	logs := buf.String()
	t.Logf("captured logs:\n%s", logs)
	if !strings.Contains(logs, "layout stamp drift") {
		t.Fatalf("expected layout stamp drift WARN, got:\n%s", logs)
	}
}
