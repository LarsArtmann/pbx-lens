package backup

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

// seedDB creates a sqlite file with one table and one row.
func seedDB(t *testing.T) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "source.db")

	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open source: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	if _, err := db.ExecContext(context.Background(),
		"CREATE TABLE calls (id TEXT PRIMARY KEY, n INTEGER)"); err != nil {
		t.Fatalf("create table: %v", err)
	}

	if _, err := db.ExecContext(context.Background(),
		"INSERT INTO calls (id, n) VALUES ('c1', 7)"); err != nil {
		t.Fatalf("insert: %v", err)
	}

	return path
}

func TestSnapshotProducesReadableCopy(t *testing.T) {
	ctx := context.Background()
	source := seedDB(t)
	out := filepath.Join(t.TempDir(), "snapshot.db")

	if err := Snapshot(ctx, source, out); err != nil {
		t.Fatalf("Snapshot: %v", err)
	}

	if err := Verify(ctx, out); err != nil {
		t.Fatalf("Verify: %v", err)
	}

	db, err := sql.Open("sqlite", out)
	if err != nil {
		t.Fatalf("open snapshot: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	var n int

	if err := db.QueryRowContext(ctx, "SELECT n FROM calls WHERE id = 'c1'").Scan(&n); err != nil {
		t.Fatalf("read snapshot row: %v", err)
	}

	if n != 7 {
		t.Fatalf("snapshot row n = %d, want 7", n)
	}
}

func TestSnapshotOverwritesPreviousSnapshot(t *testing.T) {
	ctx := context.Background()
	source := seedDB(t)
	out := filepath.Join(t.TempDir(), "snapshot.db")

	if err := Snapshot(ctx, source, out); err != nil {
		t.Fatalf("first Snapshot: %v", err)
	}

	if err := Snapshot(ctx, source, out); err != nil {
		t.Fatalf("second Snapshot must replace the first, got: %v", err)
	}

	if err := Verify(ctx, out); err != nil {
		t.Fatalf("Verify after overwrite: %v", err)
	}
}

func TestSnapshotRejectsMissingArguments(t *testing.T) {
	ctx := context.Background()

	if err := Snapshot(ctx, "", "out.db"); err == nil {
		t.Error("empty dsn must be rejected")
	}

	if err := Snapshot(ctx, "source.db", ""); err == nil {
		t.Error("empty out must be rejected")
	}
}

func TestVerifyRejectsNonDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "not-a-db")

	if err := os.WriteFile(path, []byte("just text"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	if err := Verify(context.Background(), path); err == nil {
		t.Error("Verify must reject a file that is not a sqlite database")
	}
}

func TestVerifyRejectsValidButEmptySqlite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty.db")

	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open empty db: %v", err)
	}

	if err := db.Close(); err != nil {
		t.Fatalf("close empty db: %v", err)
	}

	err = Verify(context.Background(), path)
	if err == nil {
		t.Fatal("a valid sqlite file with no lens tables must not verify as a snapshot")
	}

	if want := "no tables"; !strings.Contains(err.Error(), want) {
		t.Errorf("error should explain %q, got: %v", want, err)
	}
}
