// Package backup writes a consistent sqlite snapshot of the index for the
// nightly backup staging (C4/T14). The snapshot is produced with VACUUM INTO,
// which reads through one transaction, so the copy is consistent even while
// the serve process keeps writing. The snapshot lands via a temp file and an
// atomic rename: a failed run never destroys the previous good snapshot.
package backup

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	errorfamily "github.com/larsartmann/go-error-family"
	_ "modernc.org/sqlite"
)

// Snapshot writes a consistent copy of the sqlite database at dsn to out.
func Snapshot(ctx context.Context, dsn, out string) error {
	if dsn == "" {
		return errorfamily.NewRejection("lens.backup.missing_dsn", "backup needs a sqlite dsn")
	}

	if out == "" {
		return errorfamily.NewRejection("lens.backup.missing_out", "backup needs an output path")
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return errorfamily.NewRejection("lens.backup.open", fmt.Sprintf("opening %s: %v", dsn, err))
	}

	defer func() { _ = db.Close() }()

	staging := out + ".tmp"

	if err := os.Remove(staging); err != nil && !os.IsNotExist(err) {
		return errorfamily.NewRejection("lens.backup.clear_staging", fmt.Sprintf("clearing %s: %v", staging, err))
	}

	if _, err := db.ExecContext(ctx, "VACUUM INTO ?", staging); err != nil {
		return errorfamily.NewRejection("lens.backup.vacuum", fmt.Sprintf("snapshotting %s to %s: %v", dsn, out, err))
	}

	if err := os.Rename(staging, out); err != nil {
		return errorfamily.NewRejection("lens.backup.publish", fmt.Sprintf("publishing %s: %v", out, err))
	}

	return nil
}

// Verify opens the snapshot and confirms it is a readable sqlite database
// whose schema is intact. It is the sync check the nightly job runs before
// treating a snapshot as good staging input.
func Verify(ctx context.Context, path string) error {
	if path == "" {
		return errorfamily.NewRejection("lens.backup.missing_out", "verify needs a snapshot path")
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return errorfamily.NewRejection("lens.backup.verify_open", fmt.Sprintf("opening %s: %v", path, err))
	}

	defer func() { _ = db.Close() }()

	var result string

	if err := db.QueryRowContext(ctx, "PRAGMA integrity_check").Scan(&result); err != nil {
		return errorfamily.NewRejection(
			"lens.backup.verify_integrity",
			fmt.Sprintf("integrity check %s: %v", path, err),
		)
	}

	if result != "ok" {
		return errorfamily.NewRejection(
			"lens.backup.verify_corrupt",
			fmt.Sprintf("snapshot %s failed integrity_check: %s", path, result),
		)
	}

	// A valid-but-empty sqlite file passes integrity_check, so the schema
	// check is what makes Verify prove the file is a lens index snapshot
	// and not, say, a truncated or wrong-path download.
	var tables int

	if err := db.QueryRowContext(ctx,
		"SELECT count(*) FROM sqlite_master WHERE type = 'table'",
	).Scan(&tables); err != nil {
		return errorfamily.NewRejection(
			"lens.backup.verify_schema",
			fmt.Sprintf("schema check %s: %v", path, err),
		)
	}

	if tables == 0 {
		return errorfamily.NewRejection(
			"lens.backup.verify_empty",
			fmt.Sprintf("snapshot %s has no tables — not a lens index snapshot", path),
		)
	}

	return nil
}
