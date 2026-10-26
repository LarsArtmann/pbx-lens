<!-- docs/drafts/2026-10-26_advisory-triage-jsonv2-inventory.md
     Plan T13 (2026-10-26): the detect-only advisory lanes dispositioned —
     branching-flow hotspots, file-size residue, and the jsonv2 flip
     surface ordered by risk. Nothing here is a defect; every row is
     accept-with-reason or a parked watch. The TODO detect-only row cites
     this file. -->

# Advisory triage — branching hotspots, file sizes, jsonv2 boundaries (2026-10-26)

## 1. Branching-flow hotspots — ACCEPT (both)

- **projection.go**: today 136 lines, three `OnRecordTyped` folds
  (create/recreate/full-recompute, the pairing rule the cqrs-lint F021 row
  owns). The "18 fields" figure from the 15-09 audit counts the `CallView`
  struct: one field per rendered figure is the denominators doctrine made
  type-shaped, not a smell. No decision boundary lives in those fields.
- **dashboard.go**: `DashboardStats` carries one field per card figure for
  the same reason (each rate names its own denominator; folding fields
  would re-attach the wrong-denominator lie the type exists to prevent).

Verdict: both are view models whose width IS the model. Revisit only if a
field ever duplicates another's meaning (the split-brain test), not on
count.

## 2. File-size residue — ACCEPT (all four, re-measured)

| File                              | 15-09 figure | Now  | Verdict                                                                                  |
| --------------------------------- | ------------ | ---- | ---------------------------------------------------------------------------------------- |
| `internal/web/server.go`          | 715          | 723  | ACCEPT — the HTTP composition root; route registration reads top-to-bottom, splitting it scatters the chain order the docs teach |
| `internal/web/dashboard.go`       | 551          | 550  | ACCEPT — one query family (stats + accumulator + trends); its private accumulator has no second consumer |
| `internal/web/search.go`          | 570          | 149  | RESOLVED — the split already happened (mode handlers extracted); the residue row is stale |
| `internal/ingest/telnyx/telnyx_test.go` | 836    | 835  | ACCEPT — the fixture families already split out (enrich/silence_only/wave_live); what remains is the client+poller table suite |

## 3. jsonv2 flip surface — ordered by risk (verified 2026-10-26)

GOEXPERIMENT=jsonv2 is set fleet-wide but nothing imports
encoding/json/v2 yet; the env var becomes load-bearing when a surface
needs v2. The flip surface, measured today: **15 encoding/json call sites
across 9 files, ~107 json struct tags across 10 files** (the 15-09
audit's "39 findings" counted types, not sites; both are inventories of
the same surface).

| Tier | Surface                                                      | Why it risks the flip                                                                     |
| ---- | ------------------------------------------------------------ | ----------------------------------------------------------------------------------------- |
| P0   | `internal/calls/events.go` + `domain.go` tags (24)           | ContentHash is encoded payload bytes — a v2 encoding change forks every stored hash and forces the re-ingest wave. Hash-compat tests are the flip's green gate. |
| P1   | `web/ops.go` (13), `web/export.go` (3), `web/freshness.go` (4), `web/server.go` (6) | Wire-exact keys are test-pinned (retention SLOs pin exact values); the export stream shape is operator-consumed |
| P2   | `ingest/telnyx/dto.go` (35 + 3 calls), `webhook` (13), `agent` (7), `telnyx.go` (2) | Decode-side of upstream JSON — v2 decoders read v1 output tolerantly; lowest risk of the four tiers |
| P3   | `cmd/pbx-lens/ops.go` (3 calls, no tags)                     | CLI-only output; re-render freely                                                        |

Verdict: stay on v1 semantics until a surface demands v2; when one does,
flip P3 → P2 first (decode-only), P1 behind its exact-wire tests, and P0
LAST with a stored-hash compatibility proof. This table is the standing
order of operations.

## 4. Lint-tool residue (T7, 2026-10-26)

- **`//nolint` census**: 15 directives across 7 files, EVERY one carries a
  reason. `gochecknoglobals`/`gosec`/`nilerr`/`nilnil`/`gosmopolitan` are
  enabled golangci linters (directives live); `erraudit` (×2,
  cmd/pbx-lens/catalog.go) is a CROSS-TOOL directive — erraudit is the
  fleet's error-handling linter outside .golangci.yml, so golangci cannot
  see or warn about it. Not dead; kept.
- **Version skew (the real find)**: the devShell takes `pkgs.golangci-lint`
  UNPINNED and today resolves v2.15.0, while CI pins v2.14 (ci.yml
  GOLANGCI_LINT_VERSION). The 16-00 §e2 rule says gate binaries at the CI
  pin are step zero. v2.14 verdicts on this tree: probe pending — the
  alignment decision (flake pin down to v2.14, or CI up to v2.15) is
  recorded as its own TODO row rather than chosen unilaterally.
