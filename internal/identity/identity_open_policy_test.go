package identity

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/larsartmann/pbx-lens/internal/identity/identitytest"
)

// The OpenExisting policy (2026-11-21) exists because the identity opener
// creates its tables in WHICHEVER SQLite file it is handed — the mirror
// image of pointing an index verb at the identity file. A tool that only
// reads (identity-export) must never bootstrap a database or write into a
// foreign one, so these tests pin both refusals, the default posture, and
// the table names the guard leans on.

func existingOnlyConfig(dsn string) IdentityConfig {
	return IdentityConfig{
		DSN:       dsn,
		RPID:      "localhost",
		RPOrigins: []string{"http://localhost:8300"},
		Policy:    OpenExisting,
	}
}

// TestIdentityTablesExistAfterOpen pins the names requireIdentityDB relies
// on against a freshly opened database. They are upstream's, so a pin bump
// that renames one must fail HERE — not quietly turn the OpenExisting guard
// into a refusal of every real identity database.
func TestIdentityTablesExistAfterOpen(t *testing.T) {
	_, dsn := openFileBackedIdentity(t)

	tables := identitytest.TableNames(t, dsn)

	for _, want := range identityTables() {
		if !slices.Contains(tables, want) {
			t.Errorf(
				"a freshly opened identity database lacks %q (has %v) — update identityTables",
				want,
				tables,
			)
		}
	}
}

// TestOpenExistingRefusesAMissingDatabase pins the refusal that makes an
// empty "success" impossible: under the default posture a typo in the path
// bootstraps a brand-new identity database, which a reader would then
// export as zero users.
func TestOpenExistingRefusesAMissingDatabase(t *testing.T) {
	dsn := filepath.Join(t.TempDir(), "missing.db")

	_, err := OpenIdentity(t.Context(), existingOnlyConfig(dsn))
	if errorfamily.Code(err) != "lens.web.identity_db_missing" {
		t.Fatalf(
			"OpenIdentity(OpenExisting) on a missing file: err = %v, want lens.web.identity_db_missing",
			err,
		)
	}

	if _, statErr := os.Stat(dsn); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("the refusal must not bootstrap the file: stat err = %v", statErr)
	}

	// The default posture is untouched: the same call bootstraps.
	bootstrapped, err := OpenIdentity(t.Context(), IdentityConfig{
		DSN:       dsn,
		RPID:      "localhost",
		RPOrigins: []string{"http://localhost:8300"},
	})
	if err != nil {
		t.Fatalf("OpenIdentity(OpenOrCreate) on a missing file: %v", err)
	}

	t.Cleanup(func() {
		if cerr := bootstrapped.Close(); cerr != nil {
			t.Errorf("identity close: %v", cerr)
		}
	})
}

// TestOpenExistingRefusesAForeignDatabase pins the other half: pointed at a
// SQLite file that is not an identity database, the default posture would
// add the identity tables to it. OpenExisting refuses and leaves the file
// byte-for-byte the database it was.
func TestOpenExistingRefusesAForeignDatabase(t *testing.T) {
	dsn := filepath.Join(t.TempDir(), "notes.db")

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open %s: %v", dsn, err)
	}

	if _, err := db.Exec("CREATE TABLE notes (id INTEGER PRIMARY KEY, body TEXT)"); err != nil {
		t.Fatalf("seed the foreign database: %v", err)
	}

	if err := db.Close(); err != nil {
		t.Fatalf("close the foreign database: %v", err)
	}

	before := identitytest.TableNames(t, dsn)

	_, err = OpenIdentity(t.Context(), existingOnlyConfig(dsn))
	if errorfamily.Code(err) != "lens.web.identity_not_identity_db" {
		t.Fatalf(
			"OpenIdentity(OpenExisting) on a foreign database: err = %v, want lens.web.identity_not_identity_db",
			err,
		)
	}

	if after := identitytest.TableNames(t, dsn); !slices.Equal(before, after) {
		t.Fatalf("the refusal wrote into the foreign database: %v -> %v", before, after)
	}
}
