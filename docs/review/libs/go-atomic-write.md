# go-atomic-write — adoption-fit report (pbx-lens)

**Date:** 2026-11-04 · **Upstream:** `github.com/larsartmann/go-atomic-write` v0.6.2 (checkout last commit 2026-11-04; single module) · **Status in pbx-lens:** **NOT USED** (not in the graph — but explicitly mirrored in code comments)
**Verdict:** **DEFER** — the duplication is real but small, and the library currently _cannot own the flow_. Adopt the moment it grows an exported rename/commit primitive (upstream ask drafted).

> **Refreshed 2026-11-12:** still absent from go.mod/go.sum (NOT USED). Latest tag re-verified: **v0.6.2** (tags fetched).
> Upstream master still carries `WriteFuncWithPerm` beyond the tag (`atomicwrite.go:151`) — re-verified; still no exported path-based commit primitive, so the DEFER verdict and adopt trigger stand.

**Fleet report:** [index.html](index.html)
**Refresh rule:** re-run the census and the score on this library's next pin bump or when its flip-trigger fires (policy + prior-report reconciliation: [docs/research/README.md](../../research/README.md)).

## What the library offers

- `Write`/`WriteVerified` (xxhash fingerprint + flock + unique tmp + fsync + rename + dir-fsync), `WriteIfChanged`, streaming `WriteFunc(Verified)`, `WriteWithPerm`, `ErrConcurrentModification`.

## The duplication, quantified

`internal/backup/backup.go` re-implements the discipline in ~30 of 168 lines: unique-staging naming (`backup.go:54-57`) + the `syncFile`/`syncDir`/rename chain (`backup.go:70-112`) — and **says so** in its header comment (`backup.go:13-18`): "The stage → fsync → rename → dir-fsync chain mirrors the semantics of go-atomic-write … but if it grows an exported rename/commit primitive, adopt it here and delete this hand-rolled chain". `backup.go:52` also documents exactly which failure mode the unique-temp discipline exists to prevent.

## Why not adopt today

1. **API shape mismatch:** every go-atomic-write entry point takes Go data or an `io.Writer` for content _it_ writes. The pbx-lens staging file is produced by **sqlite itself** (`VACUUM INTO`, `backup.go:63`) — the library has no exported primitive that takes an existing file path and performs the fsync/rename/dir-fsync commit. Adoption today would add the module (xxhash + flock deps, hermetic-nix vendorHash cost) for **zero usable surface**.
2. **The escape hatch is already prepared:** the hand-rolled chain is fenced by a comment contract and a draft — `docs/drafts/go-atomic-write-commit-primitive.md` records the upstream ask and the adoption plan.
3. Scale argument: ~30 lines, one call site, test-pinned — below the threshold where a new module pays for itself.

## When this verdict flips

When upstream exports a `Commit(srcPath, dstPath)`-shaped primitive (or `WriteFromExistingFile`): adopt in `backup.go`, delete lines 54-57 + 70-112, keep the VACUUM INTO + integrity-verify steps. This is the only library in the fleet whose adoption would _remove_ hand-rolled code this quarter.

## How to verify

```console
sed -n '13,18p' internal/backup/backup.go       # the mirror contract
sed -n '52,63p' internal/backup/backup.go       # unique-temp + VACUUM INTO
go doc github.com/larsartmann/go-atomic-write   # from a module that has it; confirm no path-based commit API
ls docs/drafts/ | grep -i atomic
```
