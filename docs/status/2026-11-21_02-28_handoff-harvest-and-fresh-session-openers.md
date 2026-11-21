<!-- docs/status/2026-11-21_02-28_handoff-harvest-and-fresh-session-openers.md —
     point-in-time handoff; TODO_LIST.md stays the living source. -->

# Status — Handoff harvest + fresh-session openers (2026-11-21, 02:28 CET)

**Session scope:** the promised follow-through on the 2026-11-20 execution
wave — update the todo list to the true session-end state, write the
handoff, then execute the unblocked openers. Starting tree: HEAD `a82a283`
(the wave report), clean, CI-green, v0.4.0 released.

## a) DONE in this session

1. **TODO_LIST.md reflects the wave's end state.** Harvested against the
   wave report with code verification first (go.mod pins: templ-components
   v1.27.2 / httputil v1.9.0; `NoRegistration`+`AccentColor` live in
   internal/identity/identity.go:244,247; the ingest-edge guard
   `TestWebProdFilesNeverImportIngest`; `govulncheck@v1.9.0` in CI):
   - Deleted as landed → CHANGELOG [v0.4.0]/[Unreleased]: the deps train
     (T06/T12), the templ-components retirement row (T05), the
     recording-correlation move (T10), the govulncheck pin (T13), the
     loginpage honesty row (T08), and the shipped v0.4.0 release row.
   - Deleted per the docs-health model: the five stale 🟢 DONE rows from
     the 2026-11-12 pass (their receipts already live in the refresh notes
     and CHANGELOG).
   - Routed as new Unblocked rows: **T17 query split** (High),
     **T18 serve composition move** (Medium), **identity-export CLI
     verb** (Low).
   - Rewritten: the FM#11 row now anchors on the two routed moves; the
     PC-1 row is now the **PC-2 remainder only** (PC-1 landed, PC-3
     no-op by design); the upstream-filings row carries the four
     2026-11-20 dependency-hygiene drafts; a decision-pack row (T26)
     joins the Blocked table; docs/drafts/README.md gained the batch row.

## b) PARTIALLY DONE

1. The openers below are this session's execution queue; PC-2, the
   ScanPage spike, and the deepening candidates 2–4 remain future work
   by deliberate disposition (wave report §b).

## c) THE OPENERS THIS SESSION EXECUTES (in order)

1. **T18 serve composition move** — LANDED 2026-11-21 (commit 46f31bf):
   serve.go + ops.go out of cmd into internal/serve behind
   `serve.Run(Flags)`; the ~8 test-assembly sites deduped via
   `serveTestStack` (F51).
2. **T17 query split** — LANDED 2026-11-21 (commit 211a302 + its untagged
   closure): internal/query with exported view models + the querytest
   fixtures; request parsing stayed HTTP-layer in web (the ratified
   interface surgery).
3. **identity-export CLI verb** — REMAINS OPEN: ExportUsersToJSON wired
   beside `backup` (S effort, runbook says "runbook-only today").
4. **ScanPage spike** — REMAINS OPEN; branch-only by disposition.

## d) Gates at handoff

docs counts + doc links to run after every docs batch (the 2026-11-12
convention); each code move lands behind the full local gate: build,
`go test ./... -count=1`, golangci-lint, cqrs-lint, dedup gate, `nix
build` on touch points.
