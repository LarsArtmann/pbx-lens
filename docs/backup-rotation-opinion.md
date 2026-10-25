# Backup Rotation — Opinion (plan 05-40 M18/F57)

**Opinion: pbx-lens never rotates. The host owns retention.** This doc
records the boundary so neither side grows the other's job by accident.

## What pbx-lens owns

`pbx-lens backup -db <dsn> -out <path>` produces ONE consistent snapshot
per invocation (`internal/backup`): VACUUM INTO through a single read
transaction, staged under a unique per-run name (`<out>.tmp-<pid>-<nanos>`),
fsynced, atomically renamed onto `<out>`, then verified on the published
file. A failed run never destroys the previous good snapshot. That is the
app's entire backup contract — produce a trustworthy file at a path it was
told to use.

Consequence for retention (corrected 2026-10-25): the published path is a
SINGLE FILE (`<stateDir>/backups/index.db` in the sibling module), replaced
each run. There are no accumulation snapshots inside the app's directory, so
"rotate the app's staging files" is a no-op — the versioned copies exist
only on the host backup tool's side, which ships the file nightly.

It deliberately has NO: retention counts, age-based deletion, remote
shipping, or knowledge of the directory beyond the `-out` path it was
given. The index DB is disposable and always re-derivable from the
sources (FreeSWITCH CDRs, Telnyx, the webhook JSONL), so snapshots are a
re-derivation shortcut, not irreplaceable state — which is exactly why
rotation can stay dumb.

## What the host owns

The NixOS module (`nix-international-telephony`
`modules/telephony/lens.nix`) already states the boundary in its backup
options: the host backup tool owns retention and off-host shipping, and
must include `/var/lib/pbx-lens/backups`. Concretely:

1. **Cadence:** the `pbx-lens-backup` systemd timer runs the app's
   backup command nightly (`-out /var/lib/pbx-lens/backups/index.db`).
2. **Versioned retention:** the host backup tool keeps the versioned
   copies (restic/borg snapshots, or the provider's backup history) and
   applies the retention window there. No `-mtime` sweep of the app's
   directory is needed or useful: the directory holds one file, overwritten
   atomically after verification.
3. **off-host:** restic/borg/etc. includes the staging directory; shipped
   snapshots inherit the host's encryption and storage posture.

## Why the split

- **Doctrine:** the repo is read-only over every source and the DB is
  disposable. Adding deletion to the app would put the only destructive
  filesystem code in the one component whose whole design promise is
  "never damages anything".
- **Policy lives with the operator:** retention length is a deployment
  property (disk budget, host backup cadence), not a domain property. An
  app-side default would be wrong somewhere on day one.
- **Testability:** the app side is fully tested for what it does
  (consistency, atomicity, verify); the host side is declarative Nix the
  operator can read. A half-and-half (app deletes "old" files by age)
  would be tested by neither.

## Revisit trigger

If snapshots stop being re-derivation shortcuts — e.g. the Telnyx
recordings cache ever moves INTO the index DB — the disposable-DB
premise weakens, and rotation deserves a real design (documented
generation sets, verify-before-delete). Until then: host rotates, app
never deletes.
