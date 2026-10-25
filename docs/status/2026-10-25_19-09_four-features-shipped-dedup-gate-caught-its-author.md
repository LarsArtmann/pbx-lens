<!-- Status report 2026-10-25 19:09 — session 4 of the day. Input: the
     17-24 SUPERB plan execution (7 of 19 coarse tasks), the browser lane,
     the second calm-window FULL receipt, and a CI-red dedup gate catch
     that is the session's honest headline. §f ranked for the next session;
     §g the same three owner questions, now overdue. -->

# Session 2026-10-25 19:09 — four features shipped, the dedup gate caught its author again

**Session window:** ~17:30–19:09 CET
**Plan executed:** docs/planning/2026-10-25_17-24_SUPERB-release-trust-and-operator-depth.md (7 of 19 coarse tasks)
**Tree at report time:** `ec3be41` == origin/main, clean — **BUT CI IS RED** (see §d1)
**AGENTS budget:** 219/219 held; docs gates green at report time (59/10 archived, 20 live reports, 0 broken links, 71 test claims / 0 phantom)

## a) Fully done (receipts)

1. **P1 — the browser lane, first pass over the analytics surface**:
   `bash scripts/verify-all.sh` GREEN end-to-end — statix no-op tripwire ok;
   the three docs-truth gates (counts 59/10/20, links 20 checked / 0 broken,
   test claims 62 checked / 0 phantom); all FOUR contract probes ALL PASS
   (ui-contract, style-contract, staleness-live, ops-verdict); 42 PNGs
   captured. The screenshot diff vs `~/lens-baselines/v4` raised 25 alerts
   and EVERY one was classified PNG-by-PNG: the staleness banner + age-text
   drift (the seed corpus carries fixed 05:18 timestamps — the banner
   rendering live is correct behavior, a free proof), dashboard +24/+40 px
   from the anomaly note card, detail +32/+52 px from the History row
   (history-collision feature), mobile calls +94 px from the banner. ZERO
   regressions. Fresh baseline saved to `~/lens-baselines/v5` (42 PNGs).
2. **P2 — the second calm-window FULL-gate receipt**:
   `scripts/poll-gate.sh` verified START LOAD **1m=16.10 5m=15.43
   15m=18.09** (policy 15m < 40) and exec'd the gate in the same process →
   **60 success / 0 failed / 0 skipped**, exit 0, wall 1m1.2s, run
   `20261025-165637-dbd9b0d2`, tree frozen at `c597c77`. golangci-lint 0
   issues, test-race ✔, test-coverage ✔, nix-build ✔, art-dupl zero-group
   ✔ (at that tree). One real finding — dprint-format on FEATURES.md table
   padding — fixed in the same commit as the receipt row
   (docs/buildflow-dev-gate.md, `282565f`).
3. **P8 — test-depth closeout** (`c597c77`):
   `BenchmarkIngestCDRReplayNoOp` pins the idempotent re-dispatch hop at
   **≈5.13 µs/row vs ≈6.40 µs fresh** (76 vs 85 allocs, ≈3.3 KB vs ≈7.9 KB)
   — a source re-offering indexed rows costs just above the parse floor;
   receipt row in docs/http-stack.md beside the dispatch numbers with the
   honest load caveat (15-min ≈ 20, quiet-box expected lower).
   `go test -race ./internal/ingest/` green — the shared scanner/parser
   plumbing is race-clean. CHANGELOG section added.
4. **P5 — transcript-mode search** (`7ee64d9`): a third FTS shadow table
   (`lens_turns`, one row per transcript turn, same wholesale rebuild);
   `QueryTurns` ranks per TURN with role + turn index; every hit deep-links
   `?q=…&at=N#turn-N` (the existing no-JS jump contract — verified: the
   detail page's `at`-presence rule suppresses the redirect loop and the
   fragment lands the browser at the exact utterance); a Calls/Turns link
   pair toggles views; numbers stay call-mode-only BY DESIGN (a number is
   not something a turn said — `TestQueryTurnsHasNoNumberSidePath` pins the
   honesty). Golden `page_search_transcript`; browser probe Contract B2
   (turn chip href + active tab + fragment landing); new styles.css classes
   rebuilt via `nix run .#update-styles`; the search page hint table row in
   FEATURES + CHANGELOG.
5. **P6 — per-number segment pages** (`5911261`): `/segments/{number}`
   re-renders the dashboard's own stat cards, volume chart, and quality
   card over every call involving the number on either side — the SAME
   `DashboardStatsQuery` with a `Number` filter, so the page can never
   disagree with the table that linked here. `Segment.Href()` swapped from
   the flat filtered list to the trend page (the list stays one click away
   on the page); honest empty state for unknown numbers; golden
   `page_segments`; `TestSegmentPageRendersNumberTrend`,
   `TestSegmentHrefPointsAtTrendPage` (PathEscape keeps a literal `+` in
   path segments — only query strings read it as a space), and the
   dashboard drill-through test updated to the new href.
6. **P7 — saved views + canonical share links** (`71f981e`): the calls list
   records every FILTERED view in localStorage (`lens.recentViews`, cap 5,
   deduped, most-recent-first) and renders one-click chips via a first-party
   same-origin script (`/static/recent-views.js`, CSP `'self'`-clean, no
   nonce needed, no-JS visitors lose nothing). The server ships only the
   stable container so the page body stays byte-stable for the ETag
   contract. The dashboard and segment pages each carry a server-built
   canonical `Copy link` (CopyButton carrying path+query: window preset +
   date range only — never volatile params). Filter JSON rides a
   `data-filters` attribute (encoding/json over a string map = sorted keys
   = deterministic). Golden re-pinned; probe Contract C (chip recorded on
   the filtered view, survives navigation to the unfiltered head, click
   navigates); `TestRecentViewsContainerCarriesFilters`,
   `TestCanonicalShareLinksOnCards`, `TestCanonicalDashboardPath…`.
7. **P10 — ingest coverage analytics** (`ec3be41`): `/ops` gains an "Index
   coverage" card and `/api/ops` a `coverage` key: per source kind the RAW
   indexed-row count (siblings included — coverage is about what each
   SOURCE delivered, not distinct calls), the oldest/newest span, the
   newest-row age against the frozen-clock `CoverageNow`, and a `stale`
   gap flag against the 48 h SLO ceiling (`coverageStaleAfter`). A kind
   with zero rows is ABSENT ("never seen" ≠ "went quiet" — the card says
   so); a failed walk renders "unavailable" instead of a fake empty table.
   `TestIngestCoverageQueryFoldsPerKind` (2-kind seed, span + flag),
   `TestOpsPageAndAPICarryCoverage` (page + JSON), golden re-pinned, and
   the verify-ops-verdict probe still ALL PASS (wire keys untouched — the
   503-when-unwired contract preserved, coverage is additive).

Session totals: 7 receipt-grade feature/doc commits pushed; 9 goldens now
in testdata; test claims in living docs 62 → 71 (all real, gate-checked).

## b) Partially done

1. **The 17-24 plan execution — 7 of 19 coarse tasks** (P1, P2, P5, P6,
   P7, P8, P10 done). The plan doc header STILL says "Status: READY —
   execution not started" — annotation owed.
2. **P15 pixel truth**: the machine-local baseline IS refreshed (v5) and
   every alert classified; the plan's "refresh COMMITTED baselines" step
   collided with the README doctrine (baselines are machine-local by
   decision; the in-repo strategy stays a ROADMAP owner question) — so
   this is re-classified, not dropped. The print audit ran inside the lane
   (chrome hidden ✔, break-inside avoid ×3).
3. **P16 ops observability**: the coverage half landed via P10; the ETag
   hook counters (`OnETagGenerated`/`On304`) and retention SLO fields
   (oldest unretired view, purge lag) are NOT started.
4. **Report annotations**: the 15-39/16-58/17-12 §f items this session
   resolved (lane, FULL receipt, doc-claims gate, 10-46 header, P25
   bodies, P19.2/P19.3/P19.4, P14/P15/P16/P17/P20.4-partial) are NOT yet
   marked inline in those reports (P12's job).

## c) Not started (all four owner-gated + the 20% tail)

- **P3 release execution** — owner gate §g2 (v0.2.1 vs v0.3.0); bodies +
  CHANGELOG mechanics drafted in docs/drafts/, `cut-release.sh --dry-run`
  rehearsal still owed.
- **P4 dependency lane** — owner gate §g1 (PR #3 rebase vs close+hand-roll).
- **P9 daemon norm** — owner gate §g3; see §d2 for why it is now urgent.
- **P11** answer-rate / short-call analysis; **P12** backlog harvest;
  **P13** sibling folds; **P14** lint residue (gofumpt parity,
  unknown-`//nolint` grep, AGENTS-budget CI job); **P17** two decision
  notes (PII enforcement, export compression); **P18** robustness drills
  (FTS tokenizer table test, backup restore drill); **P19** watchlists
  (MOS poller prep T15, Telnyx re-verify m27.9, art-dupl suppressed-bucket
  audit, versions.json re-check, CSP enforce flip checklist).

## d) Totally fucked up

1. **CI IS RED on the last three pushes — the art-dupl ZERO-GROUP gate
   caught this session's own clones.** Run 40219010329 (and the two
   before it): "2 new clone group(s) introduced". Root cause, no excuse:
   the features each ran per-package `golangci-lint run` (0 issues — dupl
   is NOT a golangci-lint linter here) but the art-dupl gate itself was
   never run locally before pushing; the local FULL gate at 17:56 ran on
   the PRE-feature tree, and buildflow reports art-dupl as advisory
   findings while CI gates them. The two groups, both mine:
   (a) `internal/search/search.go` — `Query`/`QueryTurns` were born with
   duplicated prologues (MatchExpression + limit clamp);
   (b) `internal/web/detail.templ` — `transcriptMatchViews` and
   `firstMatchedTurn` repeat the range+turnMatches loop shape.
   The extraction is DESIGNED (a `normalizedQuery` prologue helper;
   `firstMatchedTurn` derived from `transcriptMatchViews`) but NOT yet
   landed — the session was ordered to write this report first, so the
   tree ships red-by-gate. This is the THIRD incident of the class
   (17-12's claim gate, 13-44's red push, now this): the gate works; the
   author keeps skipping the local run of exactly that gate.
2. **The daemon raced every feature commit.** Four interception barrages;
   the first was squashed cleanly while unpushed (fetch-first, reset
   --soft 45de690, one receipt-grade commit — the house fix working);
   the later ones got PUSHED because I pushed immediately after each
   commit to shrink the race window, so main's history now carries ~14
   `chore: auto-commit` entries interleaved with the four feature
   commits. Content verified complete after every split (diff vs last
   real commit), but history hygiene lost. §g3 is no longer hypothetical:
   today's pushes carried daemon-authored commits.
3. **Smaller, paid for in minutes**: poll-gate `--` misuse doubled the
   args → exit 69 (the script's default gate was already right; read the
   usage before decorating); a manual `"github.com/a-h/templ"` import in
   .templ files broke codegen TWICE (templ injects its own); `templ.KV`'s
   generic typing took three compile rounds (`KeyValue[string, bool]`, and
   `templ.CSSClasses` is `[]any` — string + KV items, not CSSClass); the
   new browser probe initially failed on the turn deep link (a 304 ETag
   revalidation of a URL Contract B had already loaded — correct caching
   behavior, probe corrected to accept 200/304) and on Puppeteer's
   `getAttribute` (ElementHandles need `$eval`); two edit-tool mod-time
   bounces (daemon/formatter touching files between read and edit —
   always re-view immediately); one stale LSP diagnostic (`rapid.Bytes`
   claimed undefined — `go vet` disproved it; the toolchain, not the
   editor, is the truth).

## e) What we should improve

1. **Run the dedup gate before EVERY push**: `nix develop -c art-dupl
   check -t 4 --min-lines 2 . --baseline-path scripts/art-dupl-baseline.json`
   is not in `.githooks/pre-commit` and not in my personal pre-push
   ritual — three red CI runs (~7 min of CI + three fix rounds) are the
   bill. Wire it into the hook AND the ritual.
2. **Design shared prologues before writing parallel shapes**: Query and
   QueryTurns were born duplicated. The plan-first instinct ("two result
   shapes → one normalized prologue") costs minutes and saves a gate
   cycle.
3. **The FULL receipt's shelf life**: the 17:56 receipt verified a tree
   that none of the four features touched — a fresh FULL receipt on the
   post-feature tree is OWED before any release (P3 depends on it).
4. **Push immediately after each receipt-grade commit** (shrinks the
   daemon window) — worked when remembered, forgotten twice; §g3's
   ratification decides whether squash-repair stays legal at all.
5. **The gate-wiring checklist is now SIX places** (script+chmod, CI
   Guard Self-Tests, verify-all.sh, scripts/README.md, pre-commit hook,
   and for code gates: the LOCAL pre-push ritual) — update
   scripts/README.md when the dedup gate lands in the hook.
6. ** templ helper types**: prefer returning plain data + `templ.KV` usage
   inline, or `[]any`-backed CSSClasses; the KV-generic dance cost three
   rounds. A one-line note in the adoption map would spare the next
   session.

## f) Next (ranked; provenance: the 17-24 plan + this session's findings)

1. **Land the dedup extraction** (§d1): `normalizedQuery` prologue +
   `firstMatchedTurn` via `transcriptMatchViews`; re-run art-dupl locally
   → 0 groups; full tests; push; WATCH CI GREEN before anything else.
2. Wire the art-dupl gate into `.githooks/pre-commit` + scripts/README.md
   (the gate-wiring checklist, sixth place).
3. Re-run `bash scripts/verify-all.sh` after the dedup commit (page bodies
   should be byte-identical; the lane proves it).
4. Fresh FULL-gate receipt on the post-feature tree (poll-gate →
   buildflow full; the e3 shelf-life item) — REQUIRED before P3.
5. Annotate the 17-24 plan header (7/19 executed; the CI-red erratum).
6. **§g2 answer** → execute the release (CHANGELOG split mechanics →
   `cut-release.sh vX.Y.Z` → body → `go list -m` proxy check). P25
   rehearsal (`--dry-run`) first.
7. **§g1 answer** → PR #3 rebase or close + hand-rolled sqlite bump (tidy,
   vendorHash re-derive on a CLEAN tree, gates).
8. **§g3 answer** → codify docs/concurrency.md + implement the chosen
   norm (daemon claim-file or never-reset), with a seeded proof.
9. P12: harvest this session's §f into TODO_LIST/ROADMAP; annotate the
   15-39/16-58/17-12 reports inline where items resolved.
10. P11 answer-rate: billsec-0 × hangup-cause definition, honest
    denominators row, QualityRollup extension, dashboard section.
11. P16 remainder: ETag hook counters (pinned v0.6.1 hooks) into /api/ops;
    retention SLO fields (oldest unretired view, purge lag).
12. P13 sibling folds: call_direction into the CDR template, lens.nix
    backup-rotation wording, webphone deep link (check sibling window
    first).
13. P14 lint residue: gofumpt/v2.14 parity, unknown-`//nolint` grep,
    AGENTS-budget CI job (220-line seed).
14. P17 decision notes: PII enforcement + export compression (one message,
    two questions, §g-style).
15. P18 drills: FTS tokenizer coverage table test (gotchas traps as
    cases); backup restore drill (VACUUM INTO under the current schema).
16. P19 watchlists: MOS poller fixture prep (2027-01), Telnyx quarterly
    re-verify schedule (m27.9), art-dupl suppressed-bucket audit after the
    next templ bump, versions.json re-check after the next tag wave, CSP
    enforce flip checklist.
17. docs/http-stack.md: name the coverage endpoint in the ops section.
18. docs/DOMAIN_LANGUAGE.md: coverage denominator wording (raw vs
    representative walks — the distinction P10 introduced deserves the
    glossary line).
19. Consider a `-coverage-slo` knob if 48 h fixed proves wrong in
    practice (YAGNI until then — the constant is documented).
20. Consider absolute-URL canonical links (host-aware) if pasting paths
    proves awkward — owner question, not assumed.
    21–50. The remaining 17-24 plan tail (P11–P19 fine steps), the 10-46 open
    items already folded there, and the §f lists of 15-39 (items 24–30),
    16-58 (21–30), 17-12 (20–25) — all harvested into TODO_LIST by item 9
    above; nothing new invented here.

## g) Questions for the owner (cannot be figured out)

1. **Release shape (§g2, asked three times now)**: v0.2.1 patch (fixes +
   test/gate depth only) or v0.3.0 minor (anomaly baseline, firstpage,
   transcript search, segment pages, saved views, coverage — the two days
   of features)? Both bodies + the CHANGELOG split mechanics are drafted;
   your pick is a five-minute cut. The CI-red dedup fix lands before the
   tag regardless.
2. **PR #3 (dependabot sqlite 1.60.1→1.61.0, §g1)**: `@dependabot rebase`
   (retry through the pipeline, tidy + vendorHash pass accepted) or close
   and hand-roll the bump in a normal PR? The classification comment is
   posted; only the disposition is yours.
3. **The daemon norm (§g3, now with today's evidence)**: today the auto-
   commit daemon interleaved ~14 `chore:` commits with four feature
   commits on main (pushed deliberately to shrink the race window; the
   alternative — squash while unpushed — is what the house did earlier
   today). Do you want (a) the claim-file protocol (daemon refuses under
   an active claim) or (b) never-reset (sessions amend forward, daemon
   commits are permanent)? Today proved both norms have real costs; your
   call decides which one gets implemented with a seeded proof.
