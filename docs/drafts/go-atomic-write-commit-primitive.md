> [!NOTE]
> This filing was drafted by GLM-5.3-flash via Crush from an AI-run investigation, not at my request.
>
> - [ ] MANUALLY REVIEWED by `@Lars Artmann` at `[<date-time>]`

# go-atomic-write: expose a rename/commit primitive for producers that own their staging file

## The consumer shape

`pbx-lens`'s `internal/backup` produces a sqlite snapshot via `VACUUM INTO ?` — the staging file is written by **sqlite itself**, not by Go code. The rest of the flow is exactly what go-atomic-write implements: fsync the staged file, atomic rename, fsync the parent directory. Today that tail is hand-rolled (25 lines in `internal/backup/backup.go`) because the library's entry points all take `data []byte` or an `io.Writer` producer:

- `Write` / `WriteWithPerm` / `WriteVerified` — byte payloads
- `WriteFunc` / `WriteFuncVerified` — Go writer callbacks

None can hand a foreign producer (sqlite, or any process/tool that writes its own output file) an already-staged file.

## The ask

An exported primitive that commits an existing staged file:

```go
// Commit publishes an already-written-and-fsynced staging file at target
// via atomic rename plus directory fsync.
func Commit(target, stagingPath string) error
```

(Optionally a `SyncFile(path string) error` companion; both are the two unexported pieces — `atomicRename` + `syncDir` in `rename_unix.go` — that outside consumers end up re-implementing.)

## Why it fits the library's mission

The README's pitch is "crash-safe, race-free file writes". Consumers whose staging writer is an external process currently get nothing and re-derive the durability chain badly (the pbx-lens version existed for a day before it grew the dir-fsync and unique-staging-name disciplines the library bakes in). A `Commit` primitive moves that correctness upstream.

## What pbx-lens does meanwhile

Keeps the hand-rolled chain, documented as mirroring the library's semantics, with the unique-staging-name discipline applied. Adoption is a delete when `Commit` (or equivalent) lands.
