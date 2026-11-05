# Status — SUPERB fleet-audit closeout execution (plan T01–T27)

**Written:** 2026-11-04 20:55 CET · **Session:** execution of
`docs/planning/2026-11-04_18-41_SUPERB-fleet-audit-closeout.md` (27 tasks,
109 micro-tasks) after the user's "NOW GET SHIT DONE — the WHOLE TODO LIST"
order.

**Scope line — what this report VERIFIED vs not.** VERIFIED live this
session, each with its command/artifact: `go build ./...` exit 0 at report
time (the mid-flight T16 code compiles; the gopls typecheck warnings on
`internal/app/app.go` are stale); full `go test` suites green per landed
task (internal/web, cmd/pbx-lens, internal/cli, internal/app,
internal/ingest/telnyx — run after each task batch); goldens regenerated
once (page_ops gained the Write errors row) and green; vendor hash updated
via `nix run .#update-vendor-hash` on a clean tree AND verified BOTH ways
(`nix build` ROOT-BUILD-OK + `nix develop -c echo ok` DEVSHELL-OK);
browser lane `verify-all.sh --fast` run twice — first red on a PRE-EXISTING
docs-gate drift (fixed), second red on MY Contract F (see d1). NOT
verified: the final green browser-lane pass (blocked behind d1's fix), the
T25 quality gate (dedup gate ran once green inside the lane, but
poll-gate-SOLO + golangci-lint not yet run this session), golangci on the
new files (only LSP diagnostics seen, several stale), the T24 index
re-score/screenshot, and everything in section c. No `gh`-verified filing
was made this session.

## Context

The session executed the closeout plan in order. 17 of 27 tasks are fully
landed with per-task verification, 3 are partial, 7 not started. Six
receipt-grade commits: `4b5ca5b` (T01/T02/T09/T10/T04), `d39c242`
(T05/T06/T07/T11), `56369ca` (T08/T14/T17/T18), `742a84b` (T12), `334cbfb`
(T13/T19), `8e07d79` (T15) — plus daemon heuristic commits sweeping the
docs. Working tree at report time: `internal/app/app.go` dirty (T16 wiring,
compiles).

## a) FULLY DONE (verified, receipts above)

1. **T02 doctrine truth** — the server_timing split brain is closed at
   every layer: `timing.go` hand-wrap comment first corrected then deleted
   entirely (by T01), `docs/http-stack.md` rewritten to the verified truth
   (constructor since v1.0.1), AGENTS.md shrunk to a pointer, guard test
   `TestServerTimingHeaderOnPlainRoute` pins header presence, `rg "middleware
   constructor"` sweep clean (remaining hits are archived/history + audit
   artifacts that own their own annotation).
2. **T01 middleware swap** — `serve.go` wires
   `servertiming.ServerTimingMiddleware()`; the duplicate hand-wrap is
   DELETED; `When()` gating decided and recorded (unconditional —
   single-operator htpasswd surface; `ServerTimingMiddlewareWhen` named as
   the public-route escape); the swap also gains `r.Pattern` propagation
   the hand-wrap lacked. A/B bench via a baseline worktree at `c723dca`:
   allocs/op identical (1216–1218 both trees), ns/op inside box noise —
   performance-neutral, written into http-stack.md.
3. **T04 error-family CLI** — `fatalCode` splits the boundary:
   family-classified errors map through the library sysexits (projection
   timeout now exits 75 EX_TEMPFAIL), non-family keep legacy exit 1 (the
   registry fail-opens unclassified errors to Transient — a plain bug must
   never read as retryable). Pinned by `TestFatalCodeFamilyMapping`
   (6 cases incl. the fail-closed Compose case).
4. **T05 error-family web** — one family-aware writer
   (`internal/web/errors.go` `writeIndexError`): classified errors take the
   family's HTTP status (Transient scan → honest 503, not a lying 500),
   unclassified keep fallback; in BOTH shapes the raw `err.Error()` never
   ships (crafted text only, detail logged). Ten sites swapped; the three
   recording-proxy upstream leaks (Telnyx URLs) gated with the pinned 502
   proxy contract preserved; `render.go` stops shipping template internals.
5. **T06 503 classification** — search-without-sqlite and ops-unwired are
   `errorfamily.NewInfrastructure` with their actionable text intact (the
   third plan site, the ops marshal 500, rides the writer's fallback).
6. **T07 errorfamilytest** — three hand-rolled assertions swapped onto
   `AssertFamily` (depth_trio, poller); telnyx keeps the `AsType` guard +
   contains-code check with a comment documenting why (AssertFamily alone
   would pass a fail-open plain error; AssertCode is exact, the code embeds
   the HTTP status).
7. **T08 canonical markers/logs** — the web layer adapts its injected log
   func to `*slog` (`logFuncHandler`); classified errors emit the library's
   structured record (family/code/retryable/exit_code/context.*, retryable
   at WARN) via `LogErrorContext` at the writer, both export stream
   failures, and the recording metadata failure; unclassified sites stay on
   plain `logf` — `LogErrorContext` would fail-open them to "transient",
   which would lie. Export stream markers now carry `WithContext` keys
   (`call_id`) so a truncated download says WHERE it stopped.
8. **T09 templ #52 fold** — ops last-pass renders
   `RelativeTimeProps.EmptyText: "never"`; the pre-upstream IsZero guard is
   deleted; page test updated to the fold; goldens unaffected (the golden's
   fixture has a non-zero LastPass).
9. **T10 truth ledgers** — filings ledger: #52 row added after LIVE
   verification (`gh issue view 52` → CLOSED 2026-10-29, COMPLETED) plus a
   new state-check block; adoption map rows (EmptyText fold + amber
   re-skin); TODO_LIST row 124 updated (remaining nine).
10. **T11 etag OnError** — `write_errors` counter wired through OnError,
    exposed on `/api/ops` JSON and the ops page card (definition row +
    warning alert when non-zero), pinned by
    `TestETagMiddlewareWriteErrorCounts` (failing-after-header recorder),
    http-stack.md hook note updated.
11. **T12 etag metrics adoption** — `go-etag/metrics@v0.6.1` (matching
    server pin) adopted: `Attach`'s preserve-and-append contract keeps the
    local overflow warning; CacheCounters is now a thin ops façade (inner
    metrics counters + the local OnError atomic the module doesn't cover);
    `/api/ops` JSON shape unchanged; suites green unchanged; `go mod tidy`;
    vendor hash `sha256-ECM62IKLWHw04534CMcfqkJLAOG4G3Cko6hDKCFHFOM=`
    landed and verified root + devShell.
12. **T13 ratio guard** — `cacheRatioText`: pre-first-tag reads "n/a — no
    tags resolved yet" (not a fake 0.0%), the library's documented
    SkipIfPresent caveat (>1) renders ">100% — adopted handler tags"
    instead of an arithmetic lie; normal window keeps the exact dialect
    bytes (golden safe). Three-case test.
13. **T14 fresh pin receipts** — all 13 direct go-cqrs-lite pins: real
    annotated tags, commits match the module proxy `Origin.Hash` exactly
    (spot-checked 3), versions.json agrees 13/13 TODAY (the 2026-10-25
    drift was regenerated away), zero pseudo-version pins. Structural
    finding: 12/13 tag commits are NOT ancestors of upstream master
    (release-line divergence; merge-base exists; id/v4.7.4 the lone
    exception) — "tag reachable from master" is the WRONG receipt for this
    repo; the right one is recorded in docs/gotchas.md.
14. **T15 Reset runbook** — `docs/runbooks/reset-rebuild.md`: six-step
    procedure (stop writer → `metaengine.Store.Reset` — the journal is
    never touched; `ResetResult.Partial()` loud-stop → `DrainProjections`
    replay → count parity → restart), rollback (nothing to restore;
    re-reset + re-drain; pre-rebuild `pbx-lens backup` for comparison),
    owner gates. Proven by
    `TestResetRebuildsReadModelFromJournal` (memory engine: fold 2 streams
    → reset to 0, never partial → replay → full parity, bare keys).
15. **T18 stop-order contract** — `collectRetention`'s `defer stop()` is
    commented as an ORDERING CONTRACT (the metric dies if anyone writes
    before it returns) and pinned e2e by
    `TestOpsAPICarriesRetentionTiming` through the real middleware.
16. **T19 watchlists** — adoption-map bump checklist gains step 7
    (filings-ledger state check — the step the v1.26.0 sweep skipped) and
    step 8 (two-module templ gate: ≥v1.27 requires templ ≥v0.3.1100, one
    coordinated bump); AGENTS UI-trio corrected v1.23.0 → the real v1.26.0
    and carries the appkit v0.10 watch (delete the serve Server-Timing
    wiring on bump).
17. **Docs-gate drift repair (bonus)** — the lane's docs gate was red on
    PRE-EXISTING drift (3 live reports unindexed in docs/status/README.md,
    count 23 vs 26); all three indexed (incl. this fleet-audit status),
    gate green.

## b) PARTIALLY DONE

1. **T17 /ready — wired, tested, and WRONG for the real lifecycle (see
   d1).** All artifacts exist (`ProjectionWorkersReady` + 8-case table
   test, `ReadyCheck` wiring in serve.go, http-stack readiness section with
   cqrs-htmx attribution, Contract F in the UI-contract probe, cli suite
   green) — but the probe PROVED the predicate misreads the worker
   lifecycle: `/health/ready` = 503 on a healthy serve instance. The
   predicate and its test need the semantics fix (below), plus the
   http-stack paragraph rewritten.
2. **T16 WithOnFailed — half-landed.** `DomainConfig.ProjectionHostOptions`
   wiring + `OnProjectionFailed` (slog.Error alert) are in `app.go` and the
   tree compiles (build exit 0 verified) — but the M16.3 test (callback
   emits the structured record), the M16.4 gotchas note, and the final
   build+test+commit receipt are missing. NOT run through the app suite
   since the edit.
3. **M5.9 browser lane** — ran twice (`--fast`); discovered and fixed the
   pre-existing docs-gate red, then failed on d1's 503. The green lane pass
   is outstanding and is the final UI gate for everything above.
4. **M3.3 index row recompute** — deliberately folded into T24 (one
   recompute after all code changes instead of two; recorded deviation).
   The per-lib server_timing report itself IS re-scored (33/50 → 67/67
   with a delta appendix).

## c) NOT STARTED

T20 (upstream filings: go-atomic-write ask + versions.json evidence —
note: T14's finding CHANGES the second filing's premise, see g3), T21
(HARVEST of the 50-item backlog into TODO_LIST/ROADMAP), T22 (process
decisions: audit-execution policy Q1 + PNG policy Q3 + implementation),
T23 (full-read the four prior deep-dive HTMLs, retire inherited-claim
caveats), T24 (fleet index refresh: recompute affected scores, delta
appendices, mean, re-render screenshot), T25 (the solo quality gate:
check-dedup-gate + poll-gate SOLO + golangci-lint + receipts), T26
(cross-project library-policy verdicts + refresh-rule pointer lines),
T27 (leftover batch: N/A notes, cosmetics, watchlist rows, curated-commit
close-out).

## d) TOTALLY FUCKED UP

1. **The big one — T17 built on an ASSUMED lifecycle instead of reading
   the real one.** I designed "ready = every worker running/live" from the
   projectionhost state names, wrote an 8-case test that bakes the
   assumption in (including "stopped votes not ready"), documented it in
   http-stack.md — and the browser probe immediately proved pbx-lens
   workers STOP after each drain (the poller re-drains per pass), so a
   perfectly healthy serve answers 503. The failure mode I lecture about
   (test pins the assumption, not the truth) executed by me, caught by the
   lane. Fix: ready := no worker Failed and none in Backoff (idle during
   startup and stopped-after-drain are healthy); rewrite the test cases,
   the http-stack paragraph, and re-run Contract F.
2. **multiedit hallucinated an import block** in recording_test.go —
   replaced the pbx-lens imports with a fabricated go-appkit/systemd line,
   breaking 12 references. Caught by gopls, restored by exact edit.
3. **Wrong package identifier after import** — imported
   `go-etag/metrics`-style paths and called `server_timing.` unaliased in
   export_test.go; the package's real name is `servertiming`. Caught by
   diagnostics.
4. **`writeIndexError` designed without the request parameter**, then
   request-context logging required it — retrofitted the signature across
   12 call sites with perl. Worked, but a design miss that cost a pass;
   the helper should have carried `r` from birth.
5. **The caching.go rewrite dropped `skipBodyCaching`** — a whole-file
   write that carried only the new content. Build caught it; restored.
   Lesson recorded in e).
6. **T16's first multiedit half-applied** (stray `")` in the old_string of
   the import edit; the wiring edit landed) — the tree sat half-edited
   between tool calls; fixed by a follow-up exact edit, then build-verified.
7. **Three failed iterations against the systemscenario harness** for the
   Reset test: vacuous-pass guard (no Then*), `Then` takes an event TYPE
   not a command, and two When phases leak the act-event window. The
   working pattern (Given multi-seed + idempotent-replay
   `When.ThenNoEvents.ThenQueryFunc`) exists one file over in app_test.go —
   I should have read it first.
8. **The plan's phantom API carried into my docs** — the plan names
   `ProjectionHost.Reset`; no such method exists at v4.7.1 (the real
   surface is `metaengine.Store.Reset` + drain). I verified at the pin
   before writing the runbook, but the row-113 text and the test comment
   first echoed the phantom name (row 113 now corrected).
9. **Bench methodology churn** — the first three BenchmarkConcurrentPoll
   invocations produced interleaved/parse-hostile output; solved only by
   the A/B worktree + file-redirection. Two runs wasted before the honest
   comparison.

## e) WHAT WE SHOULD IMPROVE

1. Read the harness/library API surface BEFORE writing tests that use it —
   systemscenario's guard and chain semantics cost three iterations that a
   two-minute read of app_test.go would have saved.
2. Whole-file rewrites must carry the ENTIRE original file, or not happen —
   prefer multiedit for edits-with-additions (the skipBodyCaching loss).
3. Design helper signatures with every parameter the known next step needs
   (`r` for context-aware logging) at birth.
4. Verify a build after EVERY edit batch, not at task boundaries — the T16
   window sat unverified between calls.
5. When a plan names an API, treat the name as unverified until the pin
   proves it (ProjectionHost.Reset; EmptyText line numbers were also stale).
6. Collect background jobs promptly — the lane's first red (docs gate) was
   known before the second run launched; folding both discoveries into one
   re-run would have saved a cycle.
7. Lifecycle assumptions need a probe, not just a unit test — Contract F
   catching d1 within minutes of T17 landing is exactly the lane's value;
   run the lane EARLIER in web-touching tasks (the plan's M5.9 deferral is
   right for cost, wrong for discovery latency).
8. The fleet reports' stale line references (ops.go:296-298, ops.templ:205)
   should be annotated with "verified at audit time" so future sessions
   don't chase moved code (plan rows carry raw numbers).

## f) Up to 50 things to get done next (impact-ordered)

1. **Fix T17's readiness semantics** (d1): predicate → not-ready only on
   Failed/Backoff; rewrite the table test's four wrong cases; rewrite the
   http-stack paragraph; re-run Contract F green.
2. **Finish T16**: run the app suite on the dirty tree, add the
   OnProjectionFailed log-capture test + DomainConfig options pin, gotchas
   note, commit receipt.
3. **Green browser lane** (`--fast` first, then full before closeout) as
   the UI gate over T05–T19.
4. **T25 the solo quality gate**: `scripts/check-dedup-gate.sh`, poll-gate
   SOLO (BUILDFLOW_NO_RESULT_CACHE=1, load<15min<40 doctrine), real
   `golangci-lint run ./...` (clear or confirm the pending golines/err113
   diagnostics on the new test files), zero-findings receipt.
5. **T24 fleet index refresh**: re-score server_timing 67, go-etag
   (post-OnError/metrics), templ-components (post-fold), error-family
   (post-T04/T05/T08), appkit; index cards + mean + delta appendices +
   fresh screenshot AFTER final content.
6. **T22 process decisions** (Q1/Q3 defaults recorded, then implemented):
   audit-execution policy into AGENTS.md; PNG policy — compress/keep/
   out-of-git for the 1.4 MB screenshot.
7. **T21 HARVEST**: route this report's (f) + the prior 50-item backlog
   into TODO_LIST/ROADMAP with routing rigor; link the plan.
8. **T20 upstream filings** — but re-verify premises first (g2/g3): the
   go-atomic-write commit-primitive ask (verify API absence at latest tag,
   github-voice), and the versions.json ask whose premise T14 just
   materially changed.
9. **T23 prior-report reads**: the four deep-dive HTMLs in full; retire
   inherited-claim caveats in the fleet report.
10. **T26 cross-project**: library-policy home verdicts (19), refresh-rule
    pointer lines in every per-lib report header, AGENTS server_timing
    trim post-landing.
11. **T27 leftover batch**: RecordServerTiming/MeasureWithDesc N/A note;
    docs/research README cosmetics; curated-commit close-out; rate-limit
    abuse-evidence definition; N/A confirmations (Pragmas, schema-v2,
    retry, daemon, credentials-store); go-retry trigger row.
12. Annotate the executed plan rows (T01–T19) with receipts + real line
    refs (docs-health ANNOTATE norm; the plan's stale numbers included).
13. Index THIS report into docs/status/README.md (done with the report).
14. Re-run `nix build` + devShell on the FINAL clean tree after T16/T17
    land (the two verifications ran on mid-session trees).
15. Full (non-`--fast`) browser lane once before closeout (41-PNG sweep).
16. Update AGENTS.md's fleet-audit verdict block with the new scores
    post-T24 (server_timing 33→67, error-family movement).
17. Check AGENTS.md against its 220-line budget after today's edits.
18. `go vet ./...` + `-race` on the touched packages as part of T25.
19. gofmt/golines sweep over every file this session touched (pending LSP
    warnings on main_test.go, errors_test.go, ops_timing_test.go).
20. Consider family-wrapping the ops coverage/retention scan errors at
    their source so those failures also surface 503-honest (currently
    logged raw, response continues — deliberate, but a candidate).
21. Route parseFilters Rejection errors through the writer on the JSON
    twins for family-shaped bodies (today: crafted text, safe, 400 —
    consistency candidate).
22. templ whitespace: the collapsed WriteErrors `if` block left a " <dl"
    double-space artifact in the golden — tidy with an if/else structure or
    accept + note.
23. Propose upstream `Attach` covering `OnError` (would delete the local
    writeErrors atomic) — bundle with the T20 filings round.
24. Watchlist note: go-etag's HitRatio "dedicated hook" caveat — if any
    handler ever sets tags, install the exact-accounting hook (T13's
    display guard is the tripwire).
25. Document the ProjectionWorkersReady empty-list vacuous-ready invariant
    (currently impossible: serve always registers workers).
26. FEATURES.md rows: /health/ready readiness, family error responses,
    write_errors counter, ratio guard, Reset runbook.
27. CHANGELOG Unreleased: the closeout batch.
28. `rg "#52 OPEN"` sweep for any remaining stale #52-OPEN phrasings in
    living docs (the adoption map + filings are done; check the rest).
29. Delete the /tmp scratch clones (go-cqrs-lite-receipt, go-etag-check,
    pbx-baseline already removed).
30. Re-verify `nix flake check` after the vendor-hash change (root +
    devShell done; flake check for completeness).
31. Add /health/ready to the other probe scripts if they boot serve
    (staleness-live probe) for parity.
32. Consider a guard test pinning `skipBodyCaching` against route-table
    drift (new route → must decide cache skip; today comment-only).
33. Pin the drain-before-serve ordering contract in serve.go with a comment
    (drain → rebuild → Run) — T18's sibling ordering hazard.
34. The `-dirty` derivation names in today's two nix verifications: re-run
    both on the final clean tree (item 14) so the receipt names a clean
    drv.
35. Extend the T14 receipt style to the other first-party pins (httputil,
    go-etag, templ-components, appkit): tag-vs-proxy-hash spot receipts.
36. Verify `ops.go` still uses `format.Percent` (it does, via
    cacheRatioText) and ops.templ has no unused templ import after the T13
    edit (next templ generate + lint).
37. Add the amber-re-skin semantics row check to the next real bump's
    checklist execution (step 7 now exists — first run will prove it).
38. Save the A/B bench receipt (baseline vs swapped allocs/ns) into
    docs/review/libs/httputil-server-timing.md's delta appendix (it lives
    in http-stack.md now; one pointer line for the report).
39. Consider `errors.AsType` modernization sweep for remaining
    `errors.As` call sites repo-wide (the modernize linter will flag them).
40. Decide whether `OnProjectionFailed` should also bump an ops-visible
    counter (T16's M16.2 decision was log + readiness probe; an ops row is
    the remaining surface if wanted).
41. Re-check `docs/templ-components-adoption.md` intro comment against the
    v1.26.0 re-audit (done in T10/T19; confirm nothing else references
    v1.24.0).
42. TODO_LIST: row 124's strikethrough of #52 references the fold — verify
    the rendered strikethrough reads correctly in the table (markdown
    inside table cell).
43. Sweep living docs for `ProjectionHost.Reset` phantoms after row 113's
    fix (the runbook + gotchas are clean; check planning/status history
    stays as history).
44. Guard the drain's `workerRoundState` "allDone requires STOPPED"
    semantics with a comment cross-linking the readiness predicate (the
    two read the same states with OPPOSITE goals — the split-brain trap
    d1 fell into).
45. Move the canned `-wired-ops` fixture note if ops payload gained fields
    (write_errors) — verify the fixture still satisfies the ops tests.
46. Verify page_ops golden's relative-time scrubber still covers the
    EmptyText path (never-ran state has no data-tc-relative — confirm the
    scrubber regex list doesn't need a new entry).
47. Consider recording the release-line divergence as a go-cqrs-lite
    upstream question (git workflow hygiene; evidence banked in gotchas).
48. Add `internal/web/errors.go` to the adoption map's error-page row if
    the family writer becomes the HTML error path too (currently text/JSON
    only).
49. Post-closeout: refresh `docs/research/README.md` index rows for any
    new artifacts (the runbook is docs/runbooks — check whether the index
    tracks it).
50. After everything: the curated final commit sweep + push decision (g1).

## g) Three questions I cannot answer myself

1. **Push authorization:** the closeout batch (6 receipt commits + daemon
   sweeps) is local-only on main. Push to `origin/main` now, or only after
   d1's fix + the T25 solo gate are green? (The earlier authorization
   covered the plan commit; this batch doubles the delta.)
2. **PNG policy (Q3, still open):** `docs/review/libs/index-screenshot.png`
   is ~1.4 MB in git. For T22 I need your call: compress in-git (oxipng/
   pngquant availability?), keep as-is, or move out of git (regenerate on
   demand). My default if unanswerable: keep as-is and record the decision
   — but a compress saves ~1 MB per retake and T24 will retake it.
3. **The versions.json filing (Q2-adjacent, premise changed):** T14 proved
   versions.json now agrees 13/13 — the 2026-10-25 "lies" state appears
   regenerated away, BUT the release-line divergence (12/13 tag commits not
   ancestors of master) is NEW, arguably worse evidence of the same
   workflow hazard. File the upstream ask on the divergence evidence,
   retract the filing intent, or stage it for the next drift sighting?
