package identity

import (
	"context"
	"slices"
	"testing"
)

// The identity checkpoint table (2026-11-20 cqrs-htmx conformance audit):
// openIdentityStores applies storage.SQLiteCheckpointSchema() BY HAND
// (userauth.go:330) because NewSQLiteCheckpointStore self-migrates nothing —
// every other identity store owns its own DDL, the checkpoint store does not.
// The table's existence was guarded only indirectly until now: a missing
// table first fails the projections later, flipping Ready() far from the
// cause. This test opens the identity DB through the same production
// OpenIdentity path every other identity test boots and asserts via
// sqlite_master that the table exists BY NAME.

func TestIdentityDBCarriesCheckpointTable(t *testing.T) {
	identity, dsn := openFileBackedIdentity(t)

	// Probe through the LIVE production handle — the same *sql.DB whose
	// openIdentityStores call ran storage.SQLiteCheckpointSchema()'s
	// `CREATE TABLE IF NOT EXISTS checkpoints (...)` DDL.
	var found int

	if err := identity.db.QueryRowContext(context.Background(),
		`SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name = ?`,
		"checkpoints",
	).Scan(&found); err != nil {
		t.Fatalf("probe sqlite_master on the live identity handle: %v", err)
	}

	if found != 1 {
		t.Fatalf("identity DB lacks the `checkpoints` table (tables: %v) — "+
			"storage.SQLiteCheckpointSchema() (userauth.go:330) is not applied at "+
			"open time; projections would fail later and flip Ready()",
			identityTableNames(t, dsn))
	}

	// Belt-and-braces on the same listing the snapshot tests use: the table
	// must be a real sqlite_master row, not a view or shadow name.
	if !slices.Contains(identityTableNames(t, dsn), "checkpoints") {
		t.Fatal("checkpoints present by count but absent from the table listing")
	}
}
