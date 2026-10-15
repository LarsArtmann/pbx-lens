# Backup Rotation — Opinion (plan 05-40 M18/F57)

**Opinion: pbx-lens never rotates. The host owns retention.** This doc
records the boundary so neither side grows the other's job by accident.

## What pbx-lens owns

`pbx-lens backup -db <dsn> -out <path>` produces ONE consistent snapshot
per invocation (`internal/backup`): VACUUM INTO through a single read
transaction, staged under a unique per-run name, fsynced, atomically
renamed, then verified on the published file. A failed run never destroys
the previous good snapshot. That is the app's entire backup contract —
produce a trustworthy file at a path it was told to use.

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
   backup command nightly (staging area = `/var/lib/pbx-lens/backups`).
2. **Rotation:** a host-side rotation (second timer or tmpfiles age rule)
   deletes staging snapshots past the retention window. A hardened service
   unit running `find /var/lib/pbx-lens/backups -name '*.db' -mtime +30
   -delete`, or `systemd-tmpfiles` with an age entry on the directory, is
   enough — the files are self-contained sqlite DBs with no inter-file
   references.
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
