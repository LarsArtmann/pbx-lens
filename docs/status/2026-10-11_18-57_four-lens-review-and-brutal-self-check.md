<!-- docs/status/2026-10-11_18-57_four-lens-review-and-brutal-self-check.md
     Point-in-time status snapshot (session: four-lens review, 2026-10-11
     ~18:20-18:57). Explicit .md format per operator demand in the dispatch
     prompt (status-report skill's honored override). Cross-references:
     docs/reviews/2026-10-11_data-model-review.html,
     docs/reviews/2026-10-11_full-code-review.html,
     docs/reviews/2026-10-11_naming-review-companion.html,
     docs/reviews/2026-10-11_18-45_naming-review.html (concurrent session),
     docs/planning/2026-10-11_18-30_SUPERB-four-lens-review.html -->

# Status: Four-Lens Review Session (+ brutal self-check)

Date: 2026-10-11 18:57 CEST. Scope: this session's work only (data-model-review,
full-code-review, go-modularize, naming-review), no repo-wide re-audit.

## a) FULLY DONE

| Work | Evidence |
| --- | --- |
| Every file visited (32 src + 24 test + 5 templ) | review reports, session read log |
| StreamMarker projection-key fix in `cli/agent.go` + test fixture | `grep StreamMarker internal/cli/` = 0 hand-built literals; tests green |
| htpasswd timing-pad cost fix (`padCost` matches loaded entries) | `web/auth.go`; claimed property now actually true |
| `usage()` gained the missing `purge` subcommand | `cmd/pbx-lens/main.go:84` |
| `search.Hit` identity typed (`SourceKind`/`NativeID`) | swap now a compile error; search+web tests green |
| Agent excerpt capped at domain ceiling (200 runes) + regression test | `calls.DefaultExcerptMax` owned by domain; telnyx aliases it |
| `direction(context)` shadow + `uuid` param renamed | `cdr.go` / `cli/agent.go` |
| Dead `fallthrough` case removed; export CSV/JSONL comment corrected | `web/recording.go`, `web/export.go` |
| `internal/wire` directly tested (8 subtests + round-trip) | `wire/flextime_test.go`; was zero-coverage package |
| 4 stale `DOMAIN_LANGUAGE.md` claims fixed (hash-conflict → correction semantics, one-observed-event, FlexTime location, AI badge) + tombstone term added | doc matches code again |
| go-modularize verdict: HOLD single module, 3 revisit triggers | AGENTS.md + full-code-review report |
| 3 review reports + plan written (HTML, kit template) | `docs/reviews/`, `docs/planning/` |
| AGENTS.md + TODO_LIST.md updated (structure, decisions, 3 new tickets) | both files |
| Final verification: 12/12 test packages green, golangci-lint 0 issues, buildflow gate 73 steps 0 failed | gate log: "passed with warnings" (all remaining findings documented deliberate) |

## b) PARTIALLY DONE

- **Naming-review skill process (mine):** automated detection + fixes done, but the
  Step-2 glossary artifact was never written out (done mentally); the primary
  naming pass ended up being the concurrent session's report (18-45). My
  companion report covers the delta only.
- **data-model-review problem-catalog format:** the skill's graded P1-P12 table
  shape was rendered as prose issue cards instead; severities are stated but not
  in the canonical catalog table.
- **docs/reviews cross-check:** I checked `docs/reviews/` at recon (empty), but
  did NOT re-check before writing my own reports and only discovered the
  concurrent session's naming report by accident via `git status`.
- **Plan report statuses:** `docs/planning/...four-lens-review.html` task table
  still shows all rows "Open" — never annotated to done.
- **CHANGELOG.md / FEATURES.md:** NOT updated with this session's fixes
  (AGENTS.md and TODO_LIST.md were). Repo convention says shipped changes log
  in CHANGELOG.md.
- **UI e2e probe:** AGENTS.md prescribes probing `/`, `/dashboard`, `/search`
  with a Go http probe for UI work; web package tests cover the handlers, but
  the live serve probe was not run this session.

## c) NOT STARTED (known, deliberately out of scope)

- `TurnRole` typed enum (hash-compatible; ready-when-touched)
- ContentHash v2 length-prefixed encoding (blocked on v2 gate; ticketed)
- `RecordingFetcher` consumer-side interface (ticketed, no second impl)
- Recording-proxy upstream timeout knob (ticketed, needs product bound)
- Pre-existing TODO_LIST items (upstream filings D6/C009, SegmentBar,
  buildflow binary rebuild) — untouched, not mine this session

## d) TOTALLY FUCKED UP

Nothing destroyed; tree and gate end green. But three real own-goals:

1. **Published wrong numbers in a report that preaches verified claims.** The
   full-code-review HTML said "114→125 tests" (actual: 114→**117**; I
   double-counted table-driven subtests as functions) and "56 files visited"
   (actual: **61** = 32+24+5). Caught in this self-check by re-running the
   counts; the HTML is now corrected. Lesson: the report's own footer ("every
   claim verified") applied only to findings, not to its own stat cards.
2. **Self-inflicted gate OOM:** the first buildflow run's nix-build step was
   killed because I ran two gate invocations plus golangci-lint concurrently —
   memory pressure caused by my own parallelism, then briefly misread as an
   environment fault. It is partly environment (stale buildflow binary,
   documented), partly me.
3. **Botched edit (caught immediately):** an edit that was meant to INSERT a
   test accidentally deleted `t.Parallel()` from the next test; repaired
   within two edits, tests green. No damage, but the edit was sloppy.

## e) WHAT WE SHOULD IMPROVE

- **Verify stat cards with the same rigor as findings** — run the counting
  command for every number that goes into a report.
- **Re-check the report series directory immediately before writing**, not
  only at recon — concurrent sessions land files mid-flight.
- **Ghost-system scan as an explicit review step:** `retention.Policy.Admits`
  is exported, tested, documented — and has ZERO production callers (the
  poller never applies an ingest-side cutoff). I had the information (it is
  in TODO_LIST) and still failed to flag it in either review report. That is
  exactly the ghost-system class the brutal-self-review skill warns about.
- **Close the loop on noted-not-ticketed nits** (freshness error-after-write,
  backup fsync-before-rename, exportViews double `Get("pii")`, webhook map
  order, twin `excerpt()` implementations, `searchResultLimit` vs
  `search.DefaultLimit`): either fix, ticket, or write the deliberate-non-fix
  down — "noticed in passing" is not a state.
- **Update plan-report statuses** when the planned work finishes, or say the
  plan is a frozen snapshot in the final message.
- **Single-writer discipline:** two sessions edited overlapping files
  (agent.go, dashboard.go); it converged cleanly, but a deconflict check
  (`git status` before each batch) should be a habit, not a reaction.

## f) Up to 50 things to get done next

Sorted by impact; the first ~12 are actionable TODO_LIST material, the rest
are ROADMAP-fuel / polish. (Per docs-health: extra items are a brainstorm,
not commitments.)

1. Wire `retention.Policy.Admits` into the poller as an ingest-side cutoff
   (the ghost system: kills future over-age rows at the door).
2. CHANGELOG.md entry for this session's fixes (typed Hit, excerpt ceiling,
   auth timing pad, StreamMarker purge, doc corrections).
3. Recording-proxy upstream timeout knob (needs the answer to Q1 below).
4. Annotate the plan report's task table to done (docs-health ANNOTATE).
5. Harvest sections e/f of this report into TODO_LIST/ROADMAP properly.
6. Fix `freshness.go` error-after-headers path (log-and-abort, not
   `http.Error` after body started).
7. `backup.Snapshot`: fsync the staging file before the atomic rename.
8. Dedupe `exportViews`' double `r.URL.Query().Get("pii")` read.
9. Ticket or fix the twin `excerpt()` implementations (shared helper in
   `calls` or accept as sibling-mapper twins — decide, then write it down).
10. Reconcile `web.searchResultLimit` vs `search.DefaultLimit` (both 50;
    naming report #14 — decide one source of truth).
11. Document webhook `Parse`'s map-iteration nondeterminism (harmless for
    hash stability; say so where the map is iterated).
12. Add `TurnRole` typed enum next time `domain.go` is touched.
13. Schedule ContentHash v2 canonical encoding decision (drop vs schedule).
14. Run the AGENTS.md UI e2e probe once against a seeded sqlite index.
15. Live-verify Telnyx Detail Records + insights DTOs (pre-existing TODO).
16. Upstream filings D6 (sqliteengine time bind, memory-engine field
    resolution, treefmt templ go pin).
17. Verify + file cqrs-lint C009 `len(name) > 4` upstream.
18. Rebuild/reinstall buildflow binary (200ab99 → HEAD); re-run gate; drop
    the AGENTS.md license-check note if it goes green.
19. Investigate `nix-build-verify` 6/6 failure inside buildflow (works via
    plain `nix build`/flake check) — likely buildflow sandbox memory limit.
20. Decide the E003 owner question (flat `internal/calls` split) — see Q3.
21. Hand `deploy/nixos/module-draft.nix` status forward (module landed in
    sibling repo; mark the draft as historical in FEATURES).
22. Quarterly re-verify cluster: calendar the Telnyx live-API re-check.
23. Add `?pii=hash` example to README's export section (PII mode is
    load-bearing but under-documented for users).
24. `web` package: extract the repeated `"reading the call index failed: "`
    500-string into one helper (5 call sites).
25. Consider `http.NewServeMux` route table test (every registered route
    answers; catches route-typo regressions).
26. `cli/system.go`: consider a `DrainProjections` watchdog log line when a
    drain exceeds a threshold (silent-stall diagnosability).
27. `search.Query`: cap `limit` at a hard max (caller could pass 1e9).
28. `retention.Retire`: batch-size guard for absurd views counts (maxScan
    caps input, but a log line at >1000 tombstones would help operators).
29. `wire.FlexTime`: consider rejecting year < 2000 (garbage guard) — only
    if a real source ever produces one; otherwise noise.
30. `telnyx.fetchAll`: propagate `Retry-After` on 429s (today it surfaces a
    generic http.429 rejection).
31. `cdr.ParseWithOptions`: unit-test the 18-column + MOS extra-column combo
    (13+MOS has a test; 18+MOS path shares code but no named test).
32. `web/auth.go`: test that `padCost` picks up a cost-12 file (lock the new
    timing property).
33. `backup.Verify`: also assert at least one expected table exists
    (integrity_check alone passes on an unrelated valid sqlite file).
34. Excerpt truncation marker (…) for capped excerpts? Domain decision —
    current truncation is silent by design; document or add the ellipsis.
35. `dashboard.go`: extract the `2006-01-02` day-format literal (3 uses)
    into a named constant.
36. `poller.go`: `PollOnce` runs sources sequentially; fine now, but a
    per-source timeout would stop one hung file read from stalling a pass.
37. Serve: `-recordings-dir` default is a real path while help says
    "empty = disabled" — align help text with the default.
38. `main.go`: `smoke` could also exercise the projection drain path with a
    correction (second, changed dispatch) — cheap e2e depth.
39. `DOMAIN_LANGUAGE.md`: line-number references rot; consider dropping them
    for file-level references only.
40. FEATURES.md: add the retention purge + PII export entries if missing.
41. CI: add `go tool templ generate` freshness check (diff after generate
    must be empty) — currently only build correctness is proven.
42. CI: art-dupl is warning-only; schedule a periodic dedup pass instead.
43. `search.Rebuild` uses DROP+CREATE inside one tx; consider logging row
    count at rebuild for operational insight.
44. `web/export.go`: Content-Length unknown when streaming; consider
    `Transfer-Encoding` note in docs for API consumers.
45. `internal/ingest` root package has no direct test for `EachLine` blank/
    whitespace semantics (covered indirectly; a 10-line direct test would pin it).
46. Consider `go.mod` `tool` directive audit — templ is pinned; confirm no
    other CLI-only deps sneak in via go run.
47. `AGENTS.md`: the Telnyx "re-verify quarterly" cluster has no due-date
    marker; add last-verified dates per item.
48. README: purge + backup commands are absent from the user-facing command
    list (usage() has them; README may lag — verify next docs pass).
49. `statix` 15 findings in the gate are advisory; batch-review them once.
50. Post-launch follow-up: real-traffic sanity pass on the correction wave
    behavior (first re-poll after the excerpt-cap change).

## g) Questions I cannot figure out myself

Asking via the question tool now (see below).
