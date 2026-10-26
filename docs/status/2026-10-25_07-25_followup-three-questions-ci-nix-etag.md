# Session status — follow-up: the three report questions, CI/nix repair, ETag economy (2026-10-25 07:25)

**Input:** the operator's "READ, UNDERSTAND, EXECUTE" on the 06-51 report's
open items. Continues `docs/status/2026-10-25_06-51_wave-systemscenario-catalog-execution.md`
(§g held three unanswered questions; §f item 1-2 the two CI/gate blockers).

**One-line verdict: all three blockers and two of the three operator
questions are resolved and pushed (main CI fully green on `ffff24c`); one
genuine product bug fell out of the CI red — the dashboard liveness pill
was rotating the ETag every second, silently killing the poll route's
304 revalidation economy — fixed at the call site with the library's own
clock hook; the buildflow dev-gate receipt stays deferred under the
documented load policy (box at loadavg 55/92/97, not this session's load).**

## a) DONE (each verified, not self-reported)

1. **Styles Freshness Guard red → GREEN** (report §f1, §g Q2): regenerated
   via the lawful `nix run .#update-styles`, committed, pushed — run
   40167626514 full success. Decision on Q2: this session took the fix
   rather than leaving it to the concurrent session (it was blocking main).
2. **Interrupted nix repair repaired** (found during the dprint step): the
   concurrent session's cqrs-lint 4.16.0 pin landed with BOTH the
   `fetchFromGitHub` source hash AND `vendorHash.nix` at the all-zero
   placeholder — every `nix develop` eval failed at HEAD. Restored the
   source hash (nix's own error output computed it; tag dereference
   verified rev 5a08b0e = cmd/cqrs-lint/v4.16.0) and re-ran
   `nix run .#update-vendor-hash` on a clean tree (recomputed the exact
   pre-zeroing value). DevShell + `nix build` green again.
3. **§g Q1 decided AND shipped — corrections after retire now reject**
   (`call.corrected.after_retired`): the projection's `EventCorrected` fold
   is a full-recompute create, so a grown snapshot on a retired stream
   re-created the purged view row — the same retention-leak class as the
   enrich guard, and the planned `refresh -since 0` wave would have
   mass-resurrected every purged call whose source grew. Decide-level
   guard chosen over age-gate-as-contract: the age gate is deployment
   config, the invariant lives in the aggregate; unchanged re-polls stay
   the pinned idempotent no-op. Two new scenario paths (14 total), ingest
   already files decide rejections non-fatally. Docs: CHANGELOG bullet
   extended, DOMAIN_LANGUAGE tombstone row carries the nothing-appends-
   after-retire invariant, TODO row resolved.
4. **§g Q3a — AGENTS.md budget extracted 233 → 218/219**: watchlist,
   harness semantics, adoption-map mechanics, format dialect, goldens
   gotcha now point at docs/gotchas.md and docs/templ-components-adoption.md
   (which already owned them); zero concept loss, budget script green.
   Q3b (activate `core.hooksPath .githooks`) deliberately left to the
   owner — it flips commit-gate behavior for the daemon and both sessions.
5. **The CI red that exposed a product bug — liveness pill vs ETag**
   (`TestCacheSemanticsHardening` failed in CI, then reproduced ~1-in-10
   locally with -count loops): a body-drift probe diffed two
   `/frag/dashboard` renders 1.25 s apart and pinpointed the
   `Updated HH:MM:SS` stamp (templ-components `PolledRegion`
   `ShowTimestamp`) as the only changing byte. A second-resolution stamp
   inside the swapped region rotated the body hash every second → every
   conditional replay answered 200 → the 30 s htmx poll could never 304.
   Fix: minute-quantize the pill via the component's `Now` hook (shipped
   for exactly this; the pill still visibly ticks). 40× -race green,
   full suite + lint green, goldens unchanged, CI run 40172327698 full
   success.
6. **Plan file stamped EXECUTED** (report §f30): header now points at the
   06-51 execution report.

## b) OPEN / DEFERRED

1. **buildflow dev-gate green receipt** — deferred under the documented
   load policy (docs/buildflow-dev-gate.md: 1-min < 15 AND 15-min < 40;
   box sat at 55-95). Every gate step has run green individually this
   session (build, full -race suite, coverage, gofmt, tidy -diff, golangci
   full repo, art-dupl gate, cqrs-lint, nix build, devShell eval); only
   the single receipt is missing. Run it at the next calm window.
2. ~~**Hooks activation** (`git config core.hooksPath .githooks`) — owner~~ done — git config core.hooksPath = .githooks (active in this clone)
   ~~call; the budget gate it would enforce now passes (218/219).~~
3. Concurrent session's status report (08-08) landed mid-session via the
   daemon; this session's work is disjoint from it (verified per commit).

## c) Process notes

- The daemon absorbed two receipt commits again (AGENTS.md extraction,
  templ fix + the concurrent session's report in one sweep); receipts live
  here and in CHANGELOG. LSP typecheck "errors" were stale gopls cache —
  ground-truth everything with `go vet`/`go build` before acting.
- Lesson re-confirmed: a flaky test is a finding, not noise. The cache
  test flaked because the PRODUCT had a per-second body mutation; the fix
  belongs in the product, not in test retries.
