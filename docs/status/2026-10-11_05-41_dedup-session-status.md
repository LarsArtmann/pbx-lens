# Session status — art-dupl deduplication pass

**Date/time:** 2026-10-11 05:41 CEST
**Author:** dedup session (Crush), auto-daemon commits throughout
**Scope:** the art-dupl deduplication run only (user demand: report this session, no unrelated research). Cross-referenced, not superseded: `docs/status/2026-10-11_05-21_SUPERB-session-status-where-are-we.md` (the parallel SUPERB session that owns `internal/web` + `internal/search`).
**Format note:** skill default is a styled HTML dashboard; the explicit `.md` instruction in the prompt wins (recorded standing preference for `docs/status/*.md`).
**Repo state at writing:** working tree clean except `AGENTS.md` (this session's structure-doc update, daemon will commit). All code changes already committed by the daemon (`10a144d` → `be9fdf3`).

---

## Executive summary

Ran the user's `art-dupl -t 1` report to zero harmful duplication: 14 shown clone groups triaged, 4 substantive families extracted into shared code, 10 accepted as intentional (idioms, distinct markup, deliberate design). `go build` + `go test ./...` green (9/9 packages, twice). art-dupl re-run: 14 → 11 shown groups, every survivor a one-liner idiom or distinct templ markup. BuildFlow gate still exits 69 — root-caused this session to a **Go toolchain skew family** (go.mod floor `1.27.1` vs pinned toolchain `1.27.0`, sandbox can't download, templ formatter dies), **not** caused by and not fixable within the dedup scope.

---

## Brutal self-review (user's opening questions, answered honestly)

**What did I forget?**

1. **Direct unit tests for the new `internal/ingest` package.** It has zero test files; `EachLine`/`NewScanner`/`Result` are covered only indirectly through webhook/agent tests. Blank-line, 1 MiB-line, and scanner-error paths deserve their own table-driven tests.
2. **Durable accept-rationales.** The dedup-code skill says "when accepting, leave a one-line rationale so the next reader knows it was deliberate." I delivered rationales in chat only. They exist nowhere in the repo — the next `-t 1` run will re-triage the same 11 groups from scratch.
3. **Gate verification closure.** I initially reported "my files clean, gate findings pre-existing" without confirming the nix pre-commit check went green after the format fixes. Closed only when writing this report — and the answer was "still red, for a deeper reason."
4. **The smoke double-dispatch** (`main.go` two identical `Dispatch+fatal` blocks) is deliberate (ContentHash idempotency is the point of smoke) but carries no comment — the next dedup pass will flag it again.
5. **Claim precision.** I said "zero findings from my files" — the go-auto-upgrade jsonv1 advisories literally name `main.go`, `agent.go`, `webhook.go`. The truth: no NEW findings (the flagged JSON boundaries existed before my refactor, byte-identical calls), but the phrasing overclaimed. Corrected here.

**What could I have done better?**

1. Checked `docs/status/` for the concurrent SUPERB session BEFORE editing `cmd/pbx-lens/main.go` — that file was under active edit by the other session (serve, RecordingOptions). No collision occurred, but it was luck, not process.
2. Separated "new vs pre-existing" findings in the gate summary instead of a blanket verdict.
3. Run the final gate verification in the same session as the change, not as report-writing cleanup.

**Did I lie?** No deliberate lie; one imprecise claim (above), corrected.

**Ghost systems / split brains?**

- **Split brain found (pre-existing, now worse-knowable):** `agent.excerpt()` says "same semantics as the Telnyx mapper" — the first-user-turn-else-first-turn doctrine is implemented TWICE (agent package + telnyx mapper). Two implementations of one doctrine will drift. Queued as task #43.
- **Ghost link found:** nav renders a `/dashboard` link; no dashboard route exists yet (SUPERB F-tier work, not wired). Clicking it 404s today.
- No ghost system created by this session: everything extracted is wired and consumed by 3+ packages.

---

## a) FULLY DONE

| What                                                                                                                                                                                                                                                                                                                                                                                                                           | Where                                                                                                                           | Proof                                                                                                                                 |
| ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------- |
| Clone triage at user's threshold (`-t 1`, "show everything")                                                                                                                                                                                                                                                                                                                                                                   | 14 shown groups judged, 131 detected / 117 suppressed                                                                           | user's art-dupl output + re-run                                                                                                       |
| **Shared ingest scaffold extracted**: `Result` (observations + skipped-row count), `NewScanner` (64 KiB→1 MiB bounded line scan), `EachLine` (trim + skip blanks)                                                                                                                                                                                                                                                              | `internal/ingest/ingest.go` (new root package)                                                                                  | kills clone groups #3/#7/#9 (scanner consts ×3, scanner setup ×2, trim-skip ×2) and #10 (duplicate `Result` structs in cdr + webhook) |
| `webhook.Parse` / `cdr.Parse` return `ingest.Result` directly (no per-package alias — one type, no split brain); webhook tests updated                                                                                                                                                                                                                                                                                         | `internal/ingest/webhook/webhook.go`, `webhook_test.go`, `internal/ingest/cdr/cdr.go`                                           | `go test ./internal/ingest/...` ok                                                                                                    |
| `agent.Parse` adopted `EachLine` (kept its own Transcript-shaped `Result` — different concept, deliberate)                                                                                                                                                                                                                                                                                                                     | `internal/ingest/agent/agent.go`                                                                                                | agent tests ok                                                                                                                        |
| **CLI validation helpers**: `requireDB`/`requireOneArg`, messages derived from `fs.Name()` so every subcommand inherits consistent rejections                                                                                                                                                                                                                                                                                  | `cmd/pbx-lens/main.go:79-93`                                                                                                    | replaces 5× copied validation blocks (groups #1/#2); `smoke` correctly untouched (db optional there)                                  |
| Fixed on sight: misindented `defaultAddr` const, two misaligned `Printf` arg lists, `strings.SplitSeq` modernization, agent struct-literal alignment                                                                                                                                                                                                                                                                           | `main.go`, `cdr.go`, `agent.go` (remainder via BuildFlow go-fix/golines)                                                        | LSP clean on touched hunks                                                                                                            |
| **AGENTS.md updated** (memory maintenance): scaffold package, validation helpers, no-split-brain rule, CDR-keeps-own-loop rationale                                                                                                                                                                                                                                                                                            | `AGENTS.md` "Stack decisions → Structure"                                                                                       | uncommitted at writing, daemon picks up                                                                                               |
| **Verified**: build + full suite + art-dupl delta                                                                                                                                                                                                                                                                                                                                                                              | `GOEXPERIMENT=jsonv2 go test ./...` → 9/9 ok (twice: before and after formatting); art-dupl `-t 1`: 14 → 11 shown, all accepted | tool output this session                                                                                                              |
| **Gate root cause identified** (verification debt paid): nix pre-commit `templ` formatter fails because processing generated Go needs toolchain `go1.27.1`; sandbox Go is 1.27.0 and the download to proxy.golang.org is refused (no network). Same skew family as the earlier host-side license-check failure (go-appkit needs ≥1.27.1, GOTOOLCHAIN=local runs 1.27.0) and the go-version-auto-configure patch-floor warning. | `nix log <pre-commit-run.drv>`                                                                                                  | full stderr captured this session; **red before my session** (templ files aren't mine, go.mod untouched by me)                        |

## b) PARTIALLY DONE

1. **BuildFlow gate green-ness.** Root cause for the templ/pre-commit + license-check + patch-floor family identified; the FIX (toolchain unification) is a fleet decision, not applied. Additionally the final gate run surfaced two NEW-not-diagnosed failures: `govalid-generate` exit 1 and `golangci-lint:verify` "no go files to analyze" in 1 module.
2. **Accepted-duplication documentation.** Rationales delivered in chat + partially in AGENTS.md; not yet a durable dedup baseline/policy.
3. **This report's section (f)** — not yet harvested into `TODO_LIST.md` (docs-health HARVEST awaits your instruction per skill contract).
4. **LSP trust.** gopls reported phantom compile errors in `internal/web/recording_test.go`/`server_test.go` all session while `go test ./internal/web` passes; stale-cache noise not chased down.
5. **SUPERB B3 search** — other session's ~90%; the missing `cli.RebuildSearchIndex` wiring is NOT mine and NOT started by me.

## c) NOT STARTED

- Toolchain unification (1.27.0 pin vs 1.27.1 floor) — the gate's root fix.
- `govalid-generate` + golangci-lint "no go files" diagnosis.
- TODO_LIST harvest of this report.
- Direct unit tests for `internal/ingest`.
- art-dupl as a standing gate/CI step with an accepted baseline.
- Comment on smoke's deliberate double-dispatch.
- jsonv2 migration decision (go-auto-upgrade advisories across the repo, incl. untouched `telnyx.go`).
- nix-checker hash extraction + reconciling AGENTS.md's "vendorHash.nix carries the hash" claim against the checker still seeing it inlined at `flake.nix:102`.
- Everything in the SUPERB frontend/backend plans (other session's lane).

## d) TOTALLY FUCKED UP

1. **The quality gate contradicts itself.** Host-side `buildflow format` says the tree is clean; the nix-sandbox pre-commit check says "formatting failures detected" — because templ formatting needs a Go toolchain the sandbox must download and can't. Two sources of truth for "is this repo formatted" disagree, and the pre-commit flake check CANNOT go green in this environment as-is. (Pre-existing — red before this session — but it is the biggest process breakage visible, and it poisons every gate run's signal.)
2. **Gate exit 69 is now ambient noise.** With nix-hash-fix at a 26/28 (93%) failure rate and advisory findings (jsonv1 posture, patch floor) unresolved-policy, every run exits non-zero — which is exactly how a real regression (or the govalid one) hides in the red.
3. **`nix log` output is a firehose.** The templ formatter error embeds entire generated Go files in its message; triaging took a forensic dig. (Tool UX bug worth filing upstream.)
4. **My own reporting slip** ("zero findings from my files") — owned and corrected above; the standard for this repo's error doctrine applies to my summaries too.

## e) WHAT WE SHOULD IMPROVE

1. **One toolchain, one truth** — align devShell pin, go.mod floor, and sandbox Go; the skew family (license-check, templ-fmt, patch-floor warning) collapses into one fix.
2. **Durable dedup baseline** — record the 11 accepted groups + rationale in the repo (AGENTS.md "Duplication policy" section or an `art-dupl` baseline file), so future passes triage deltas, not absolutes.
3. **Test what you extract** — new shared packages ship with their own tests, not only indirect coverage.
4. **Comment deliberate clones at the site** (smoke double-dispatch) — the cheapest possible accept-rationale.
5. **Gate signal hygiene** — skip/fix the 93%-failure step; a gate that always fails trains everyone to ignore it.
6. **Cross-session protocol** — read the newest `docs/status/` report before touching shared files; two sessions edited `main.go` today by luck, not coordination.
7. **Split-brain sweep for "same doctrine, two implementations"** — `excerpt()` is the known instance; the pattern class deserves a dedup pass of its own (art-dupl can't see semantic clones across packages).
8. **Don't trust cached green steps** (govulncheck finishing in 0ms) — BuildFlow's own anti-pattern list warns this; verify scan counts periodically.

## f) UP TO 50 THINGS TO GET DONE NEXT

_(Brainstorm per skill contract — most belong in ROADMAP/TODO_LIST via HARVEST, sorted roughly by impact.)_

**Dedup thread (this session):**

1. Finish B3 search wiring: `cli.RebuildSearchIndex` + serve/poller call (SUPERB session's missing 10%).
2. Implement `/dashboard` route — nav already links to a 404 (ghost link).
3. Unit-test `internal/ingest` directly (blank lines, 1 MiB line, scanner error, EachLine trim semantics).
4. Comment smoke's deliberate double-dispatch.
5. Record the 11-group dedup accept-baseline durably.
6. Wire art-dupl into the routine (CI warning-only or a BuildFlow provider ask — fleet question).
7. Re-run art-dupl at `-t 5` to confirm the actionable-threshold baseline is clean.

**Gate / toolchain (root cause family):**
8. Bump Go toolchain everywhere to 1.27.1 (devShell pin + sandbox story) or drop the go.mod patch floor — pick one.
9. Re-run `nix build` + pre-commit check green after #8.
10. Diagnose `govalid-generate` exit 1.
11. Diagnose golangci-lint "no go files to analyze" in 1 module (root `.` package trap?).
12. Investigate or `skip_steps` nix-hash-fix (93% failure rate, documented pattern still bleeding).
13. Decide go-version-auto-configure policy: fix go.mod or suppress the step.
14. Reconcile AGENTS.md "vendorHash.nix" claim vs nix-checker seeing inline hashes at `flake.nix:91/102`; extract or annotate.
15. Verify govulncheck actually scans files (0ms cached greens).
16. File upstream: templ/treefmt error message flooding `nix log` with generated code.
17. Confirm CI lane (`.github/workflows/ci.yml`) mirrors or deliberately diverges from the local gate.

**Docs:**
18. docs-health VERIFY pass on AGENTS.md claims (the `internal/cli` structure claim vs main-package reality; vendorHash claim).
19. TODO_LIST.md HARVEST of this report's section (f).
20. FEATURES.md refresh (search, recordings, serve, dashboard status).
21. README check: 6 verbs, serve flags, current screenshots.
22. Check `docs/DOMAIN_LANGUAGE.md` needs the now-shared `Result`/`Skipped` vocabulary.
23. ANNOTATE the 03-55 "gate-truth" report with today's root cause (templ fmt + toolchain skew).

**jsonv2 / modernization:**
24. Decide jsonv2 migration timing (go-auto-upgrade advisories: main, agent, webhook, telnyx, +6 more).
25. If migrating: audit `wire.FlexTime` + wire DTO boundaries first (runtime-invisible failure class).
26. Reuse the ContentHash stability tests as the jsonv2 canary.

**Domain (planned, untouched by this session):**
27. PII policy: enforce `E164`/`ParticipantHash` (types exist, unenforced).
28. T9 agent-transcript growth; `agent.Result` shape decision rides on it.
29. Inbound AI interception → revisit caller-declared Direction for Telnyx + agent.
30. Resolve E003 owner question: split `internal/calls` or document flat-domain policy permanently.
31. Plan schema v2 upcasters (F005: mandatory at v2).
32. Quarterly Telnyx live-API re-verification (filters/metadata/timestamps cluster).
33. Watch Telnyx docs for recordings-list filter keys (client-side correlation is the current workaround).
34. Revisit P006 (poll vs Watcher) if serve grows hot paths.
35. F007 revisit trigger: any handler growing an external side effect.

**UI:**
36. templ-components >v1.21.0 watch: adopt `errorpage`/`FilterBar`/`SegmentedControl`/`utils/format` when published (replaces hand-rolled equivalents).
37. Deliberate templ-components bump routine (stay on published versions, no local replace).
38. Dashboard page design (depends on #2).
39. Keyset pagination UX consistency once dashboard lands.

**Testing:**
40. E2E serve test automated in CI (currently manual-evidence only, per 05-21 report).
41. Property/order-independence test for webhook aggregation (load-bearing for ContentHash).
42. Split-brain fix: unify `excerpt()` doctrine between agent + telnyx mapper (one function, one package, both call it).
43. `call.record.mutated` conflict-path test at the app layer (parsing-bug detector).
44. Coverage report on the ingest packages post-refactor.

**Nix / build:**
45. Verify templ `preBuild` codegen stays sandbox-safe after the toolchain change (#8).
46. cqrs-lint vendored rev bump routine note (two hash iterations expected).

**Ops / product:**
47. Thin NixOS deploy module for nix-international-telephony (AGENTS.md plan).
48. webphone read-only deep link (planned, no data dependency).
49. Staleness threshold: make T12's constant configurable or document it.
50. Serve log rotation (proactive-maintenance rule).

## g) QUESTIONS I CANNOT FIGURE OUT MYSELF

1. **Is the SUPERB session still active?** B3 search sat at ~90% (missing `cli.RebuildSearchIndex` wiring) and `/dashboard` is a live 404. Should I stay hands-off `main.go`/`internal/web`/`internal/search` (single-writer), or am I clear to finish B3 + dashboard in a follow-up?
2. **Which side of the toolchain skew do we fix?** Root cause of the whole gate-red family: go.mod floor `go 1.27.1` (forced by go-appkit v0.8.0) vs the pinned 1.27.0 toolchain (devShell `GOTOOLCHAIN=local` + sandbox without network). Bump the fleet toolchain pin to 1.27.1, or is there a reason to stay on 1.27.0?
3. **Policy call on the ambient red:** the 11 accepted duplicate groups, the jsonv1 advisories, the patch-floor warning, and nix-checker's extraction suggestions — accept-and-document, fix now, or file upstream? (I can execute any of the three; I should not choose the repo's risk posture unilaterally.)

---

_WAITING FOR INSTRUCTIONS._
