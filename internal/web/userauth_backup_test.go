package web

import (
	"context"
	"database/sql"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/larsartmann/pbx-lens/internal/backup"
)

// Identity DB durability (2026-11-12 plan A5): the identity database is the
// first NEW durable state since the index — the operator's account and
// sessions are NOT re-derivable, so the nightly snapshot must cover it. The
// backup command is DSN-agnostic by construction (`pbx-lens backup -db
// <identity.db>`), so these tests pin the property rather than add wiring:
// the snapshot runs against a LIVE identity handle (serve keeps its
// connection while the nightly job reads), verifies, and the published file
// carries the usermgmt schema and rows — restoring it restores logins.

// openFileBackedIdentity is openTestIdentity with the DSN surfaced: the
// backup flow needs the file path, not just the handle.
func openFileBackedIdentity(t *testing.T) (*Identity, string) {
	t.Helper()

	dsn := filepath.Join(t.TempDir(), "identity.db")

	identity, err := OpenIdentity(context.Background(), IdentityConfig{
		DSN:       dsn,
		RPID:      "localhost",
		RPOrigins: []string{"http://localhost:8300"},
	})
	if err != nil {
		t.Fatalf("OpenIdentity: %v", err)
	}

	t.Cleanup(func() {
		if cerr := identity.Close(); cerr != nil {
			t.Errorf("identity close: %v", cerr)
		}
	})

	return identity, dsn
}

func identityTableNames(t *testing.T, dsn string) []string {
	t.Helper()

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open %s: %v", dsn, err)
	}

	defer func() { _ = db.Close() }()

	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type = 'table' ORDER BY name")
	if err != nil {
		t.Fatalf("list tables: %v", err)
	}

	defer func() { _ = rows.Close() }()

	var names []string

	for rows.Next() {
		var name string

		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan table name: %v", err)
		}

		names = append(names, name)
	}

	if err := rows.Err(); err != nil {
		t.Fatalf("iterate tables: %v", err)
	}

	return names
}

func TestIdentityDBSnapshotRoundTripWhileLive(t *testing.T) {
	_, dsn := openFileBackedIdentity(t)

	out := filepath.Join(t.TempDir(), "identity-snapshot.db")

	ctx := context.Background()

	// Snapshot WHILE the identity handle is open — the production posture
	// (the serve process holds its connection; VACUUM INTO reads through
	// one consistent transaction). Closing first would prove nothing about
	// the live path.
	if err := backup.Snapshot(ctx, dsn, out); err != nil {
		t.Fatalf("Snapshot over a live identity DB: %v", err)
	}

	if err := backup.Verify(ctx, out); err != nil {
		t.Fatalf("Verify on the identity snapshot: %v", err)
	}

	sourceTables := identityTableNames(t, dsn)

	snapshotTables := identityTableNames(t, out)

	if !slices.Equal(snapshotTables, sourceTables) {
		t.Fatalf("snapshot tables diverge from source:\n source:  %v\n snapshot: %v", sourceTables, snapshotTables)
	}

	if !slices.Contains(snapshotTables, "user_sessions") {
		t.Fatalf("snapshot is missing user_sessions — restoring it would not restore sessions (tables: %v)",
			snapshotTables)
	}
}

// TestIdentityDBSnapshotCarriesSessionRows closes the loop one step further:
// after a session row EXISTS (written through the identity handle into the
// SQL session store's own table), the snapshot still carries it — a restore
// brings the login state back, not just an empty schema.
func TestIdentityDBSnapshotCarriesSessionRows(t *testing.T) {
	identity, dsn := openFileBackedIdentity(t)

	ctx := context.Background()

	const wantSessions = 1

	// One session row in the REAL session-store table (its SQLite schema:
	// token/user_id/created_at/expires_at/origin_type/origin_data), written
	// through the same handle the store uses.
	now := time.Now().UTC().Format(time.RFC3339Nano)

	_, err := identity.db.ExecContext(ctx,
		"INSERT INTO user_sessions (token, user_id, created_at, expires_at, origin_type, origin_data) "+
			"VALUES (?, ?, ?, ?, ?, ?)",
		"test-token", "test-user", now, now, "", "",
	)
	if err != nil {
		t.Fatalf("seed session row: %v", err)
	}

	out := filepath.Join(t.TempDir(), "identity-snapshot.db")

	if err := backup.Snapshot(ctx, dsn, out); err != nil {
		t.Fatalf("Snapshot: %v", err)
	}

	snapshot, err := sql.Open("sqlite", out)
	if err != nil {
		t.Fatalf("open snapshot: %v", err)
	}

	defer func() { _ = snapshot.Close() }()

	var sessions int

	if err := snapshot.QueryRow("SELECT count(*) FROM user_sessions").Scan(&sessions); err != nil {
		t.Fatalf("count user_sessions in snapshot: %v", err)
	}

	if sessions != wantSessions {
		t.Fatalf("snapshot carries %d sessions, source wrote %d — restore would lose login state",
			sessions, wantSessions)
	}
}
