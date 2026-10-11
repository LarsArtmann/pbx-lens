# SUPERB session status — Telnyx enrichment (T15/T13) and the TODO sweep

**When:** 2026-10-11, 09:55–11:31 CEST (single session, parallel agent session
active in the same repo)
**Scope:** TODO_LIST.md pass, ranked by impact. High-impact data gaps first,
then tests, ops drafts, docs.
**Truth basis:** every claim below was verified against the working tree this
session; statuses are verified, not assumed.

---

## a) FULLY DONE

1. **T15 — Detail-Records cost fetch into Enrichment.**
   `GET /detail_records` client (`internal/ingest/telnyx/telnyx.go`):
   required `filter[record_type]=call-control`, optional
   `filter[started_at][gte]`, page size capped at 50 (endpoint schema max),
   alias-tolerant `DetailRecord` decode (`id`/`uuid`,
   `call_control_id`/`telnyx_call_control_id`, cost as decimal string OR
   JSON number). `telnyx.MicrosFromCost`/`CostEnrichment` convert currency
   units → domain micros (rounded, sanity-capped at 1M units).
2. **T13 — Conversation-insights fetch into Enrichment.**
   `GET /ai/conversations/{id}/conversations-insights` (path verified
   against the Telnyx OpenAPI spec — the suffix is `conversations-insights`,
   NOT `insights`). `telnyx.InsightSummary`: newest `completed` row wins,
   pending/in_progress/failed never summarize, non-empty results join with
   `\n`.
3. **The enrichment pass itself.** `cli.EnrichTelnyxCalls`
   (`internal/cli/enrich.go`): correlates detail records by
   `call_control_id` (recordings-correlation pattern), dispatches
   `call.enrich` via the new `CallView.Observation()` rebuild. Idempotent by
   construction: unchanged enrichment replays to zero events; already-final
   fields skip their fetch; the detail list is lazy (≤1 request per pass,
   only when a seed still lacks cost). Classification: Considered / Enriched
   / Unchanged / Missing / Rejected. Refactored to `enrichPass` after
   gocognit flagged complexity 34.
4. **Drain-ordering contract.** Enrichment reads the read model, so
   `pollTelnyx` drains mid-pass (ingest → drain → enrich; pass-end drain
   covers enrichment events) and `ingest-telnyx` drains twice. Documented in
   AGENTS.md — this is the trap class that cost ~1h once before
   (StreamMarker/one-shot-drainer gotchas).
5. **Wiring.** `pollTelnyx` (internal/cli/poller.go) and `ingest-telnyx`
   (cmd/pbx-lens/main.go) both run enrichment and log/emit the new stats
   line.
6. **`CallView.Observation()`** (internal/calls/projection.go) with
   `TestViewObservationRoundTripHashesIdentically` — rebuild re-hashes to
   the view's stored `ContentHash`; the only sanctioned view→observation
   path.
7. **`calls.MicrosPerUnit`** — single cost-unit constant; the web display's
   private `microsPerUnit` duplicate was folded into the domain (split-brain
   kill).
8. **Tests (all green):**
   - `internal/ingest/telnyx`: MicrosFromCost table, CostEnrichment,
     InsightSummary selection (newest-completed, join, empty cases),
     DetailRecords query/shape assertions, tolerant decode fixtures
     (documented shape, webrtc-doc shape, numeric cost, missing-optionals),
     DetailRecordByCallControl, insights path + wire decode, empty-ID
     rejection.
   - `internal/cli`: **httptest-fake Telnyx e2e** (`enrich_test.go`,
     `fakeTelnyx`) — full poll pass fetch→ingest→enrich→index-rebuild,
     enrichment lands in view + FTS, second pass replays to zero
     ("1 unchanged"), skip-when-already-enriched fetch counting, Missing
     seed classification. This also closes the "Poller test with an
     httptest-fake Telnyx source" TODO item.
   - `internal/calls`: hash-roundtrip invariant.
   - `internal/web`: **one-flow UI e2e** (`flow_test.go`) — seed → list →
     filter → detail → search → /api/freshness → auth gate (401 unauth /
     200 auth), staleness banner on the unfiltered page.
9. **T11 — verified already implemented, not redone.** The `-ai-dest` flag,
   `ParseWithOptions`, `isAI`, and tests predate this session (evidence in
   `cdr.go:228`, `main.go:103/115`). The TODO row's evidence
   ("cdr.go never sets IsAI") was stale. Closed by TODO_LIST update, zero
   code.
10. **T14 — nightly backup staging drafts.**
    `deploy/systemd/pbx-lens-backup.service` (oneshot, same StateDirectory →
    shared dynamic user can read index.db and stage into
    `/var/lib/pbx-lens/backups/`) + `.timer` (02:15 nightly, RandomizedDelay,
    Persistent) + `backup.enable`/`backup.calendar` options in
    `deploy/nixos/module-draft.nix`. Host backup tool stays owner of
    retention/shipping.
11. **FTS partial-number hint.** `search.templ`: placeholder now shows a
    full `+number` example, hint paragraph states partial numbers never
    match (prefix queries CANNOT fix mid-number substrings — token is the
    whole number), with a locking test assertion.
12. **Docs.** CHANGELOG (Added entries for enrichment, roundtrip, backup
    staging, tests, hint), FEATURES (4 rows 🟡→🟢: cost, insight, backup
    staging, plus the Operations cost/insight rows), TODO_LIST (rewritten:
    done items removed, T23 marked 🟡 with evidence, new live-verify row),
    AGENTS (enrichment endpoint truths + drain contract + roundtrip +
    MicrosPerUnit gotchas; E014 cqrs-lint policy entry).
13. **Verification state at session end:** `go build ./...` clean; **11/11
    packages pass** (`go test ./... -count=1`); `golangci-lint run ./...` →
    **0 issues**; `cqrs-lint .` → **0 ERROR** (E003/E005/E014 documented
    INFO/decisions); `smoke` e2e green; `nix fmt` → 0 changed.

## b) PARTIALLY DONE

1. **T23 retention/PII purge — 🟡 owned by the parallel session.** I
   implemented a reset-based retention (ResetForRetention + cutoff filtering
   across the poller) and then **reverted it fully** when the parallel
   session's `internal/retention/` package + `purge` command surfaced in the
   tree. Remaining open half per TODO_LIST: PII mode
   (`E164`/`ParticipantHash` load-bearing) and ingest-side cutoff so
   re-derivation respects the purge.
2. **Live-API verification of the two new endpoints.** DTOs are
   spec-verified (OpenAPI) but never touched the real API (no key on this
   machine — checked env, credstore paths). AGENTS.md quarterly re-verify
   cluster now lists this; TODO_LIST carries a dedicated row.
3. **templ toolchain skew (parallel session's bump).** go.mod now carries
   `templ v0.3.1070` (commit 40e1a49) while the CLI in the devShell is
   v0.3.1020 (generator prints a version-check warning). My codegen
   regenerated fine, but the pin should be aligned in the flake.
4. **AGENTS UI-trio table.** Still says the templ pin is v0.3.1020; the
   parallel bump makes that row stale. One-line fix, deliberately not
   touched mid-flight (parallel session owns that bump's docs).
5. **Enrichment field coverage on surfaces I didn't verify.** Export
   (JSONL/CSV) serializes whole `CallView` so cost/insight ride along
   automatically, but I added no explicit export test for the new fields;
   dashboard does not yet show spend aggregates.

## c) NOT STARTED (left as found)

1. Sibling handoff: `deploy/nixos/module-draft.nix` →
   `services.pbx-lens` in nix-international-telephony (T7 half 2).
2. SegmentBar swap in `dashboard.templ aiHumanCard` (BLOCKED: unreleased
   upstream, not in templ-components v1.21.0).
3. cqrs-lint C009 bare-`must` upstream verify + filing.
4. D6 upstream filings: sqliteengine time binding, memory-engine field-name
   resolution, treefmt templ go pin (all three verified in-repo with
   repros, none filed).
5. BuildFlow binary rebuild (200ab99 → HEAD) to fix license-check's sandbox
   go floor; then drop the AGENTS license-check note.
6. `nix build` was NOT re-run after my changes (build+tests+lint+smoke were;
   the flake build with templ/tailwind preBuild was not).
7. DOMAIN_LANGUAGE.md not extended this session (Enrichment pre-dates me;
   `TelnyxSeed`/`EnrichStats`/roundtrip terms not glossary'd).

## d) TOTALLY FUCKED UP (what I got wrong, honestly)

1. **Started T23 blind — the worst miss of the session.** The initial git
   status snapshot listed `?? internal/retention/` and I never registered
   it. I designed and built a competing retention approach (reset + cutoff)
   before discovering the parallel session's `purge` command landing
   mid-session. Cost: one full design+implementation cycle reverted, and a
   real split-brain risk had I wired it into main.go. Root cause: I trusted
   the pasted TODO list over the working tree. Rule going forward: in a
   repo with a known auto-commit daemon and parallel sessions, inventory
   `git status` untracked dirs BEFORE task selection.
2. **Baseline discipline gap.** I ran targeted baseline tests (cli, telnyx)
   but not `internal/web`, then discovered a failing dashboard test and had
   to prove it wasn't mine — a worktree detour that first failed to build
   (`*_templ.go` are gitignored; the worktree needed a templ generate). Two
   wasted cycles. Cheap fix: full `go test ./...` baseline before first
   edit, and remember worktrees need codegen.
3. **First-pass lint failures.** gocognit 34 (EnrichTelnyxCalls), wsl×3,
   gci×3, golines×3, QF1002, nonamedreturns — I wrote code that didn't
   match the house gate and burned a fix round. The refactor (enrichPass)
   genuinely improved the code, but I should have decomposed before being
   told.
4. **Small correctness churn in tests:** asserted on percent-encoded query
   strings (failed; switched to parsed `url.Values`), invented a
   nonexistent `FlexTimeAlias` helper (removed; reuse `ft`), missed the
   `net/url` import. All caught by tests within minutes, but avoidable.
5. **Shared-helper signature churn:** `fetchAll` gained a `pageSize` param
   across all call sites. An options struct would have been the lower-churn
   shape for a generic shared by five endpoints.
6. **Trusted stale LSP diagnostics over the real gate.** godoclint/
   nonamedreturns/QF1008 warnings persisted on enrich.go after the fixes
   were verifiably applied; the devshell `golangci-lint run` was the truth
   (0 issues). Lesson locked in: reconcile against the CLI gate, not the
   LSP cache.

## e) WHAT WE SHOULD IMPROVE

1. **Parallel-session protocol:** before any task ≥ medium, `ls` untracked
   dirs + `git log --since` check; claim tasks in TODO_LIST (🟡) before
   starting.
2. **Live-API verification harness:** a `pbx-lens telnyx-probe` (or
   fixture-recording mode) that hits the real API once and stores sanitized
   fixtures, so DTO drift is caught mechanically each quarter instead of by
   memory.
3. **Test-double consolidation:** `fakeTelnyx` (enrich_test.go) and the
   `testServer` handlers (telnyx_test.go) encode overlapping fixtures
   (eggs conversation exists twice). A tiny `internal/ingest/telnyx/telnyxtest`
   package would keep one truth.
4. **fetchAll API shape:** options struct instead of positional
   (limit, pageSize).
5. **Zero-cost enrichment churn:** a $0.00 record means CostMicros stays 0,
   so the lazy detail fetch never marks the seed "done" — one list request
   per pass forever for such calls. Acceptable (1 request), but a
   "cost-checked" marker or tolerance would stop it.
6. **Enrichment observability:** poller logs counts but not durations or
   skip reasons per field; a one-line duration log would surface
   slow Telnyx passes.
7. **Rejected-seed semantics:** conversations that fail mapping get no
   seed (correct), but insights-fetch errors reject the whole seed while a
   partial cost may already have been resolved this pass — next pass
   re-fetches the detail list lazily, so cost is not lost, but this
   ordering deserves a test.
8. **Docs freshness automation:** the templ-pin row in AGENTS went stale
   within hours of the parallel bump — the docs-health VERIFY pass should
   include version-pin rows.
9. **CLI-command e2e gap:** enrichment is tested via the poller path; the
   `ingest-telnyx` command wiring (double drain, output lines) has no
   direct test.

## f) NEXT — up to 50 things to get done (impact-ranked, evidence-based)

**Live truth & T23 completion**
1. Live-verify `DetailRecord`/insights DTOs with a real key; record
   sanitized fixtures (TODO_LIST row).
2. T23: make `E164`/`ParticipantHash` load-bearing (PII mode).
3. T23: ingest-side retention cutoff so re-derivation respects `purge`.
4. T23: retention × serve concurrency — document or lock (purge while serve
   holds the DB).
5. T23: FTS rebuild integration after purge (verify search drops retired
   calls).
6. Quarterly re-verify cadence: add the two new endpoints to the AGENTS
   checklist explicitly (currently prose).

**Upstream filings (verified in-repo, unfiled)**
7. File sqliteengine time-binding issue (D6 repro exists).
8. File memory-engine field-name resolution issue (D6).
9. File treefmt templ go-pin issue (D6, belongs in go-nix-helpers).
10. Verify + file cqrs-lint C009 bare-`must` exemption (len 4 guard).
11. Consider filing E005/E014 package-blindness class upstream (one issue
    covering both, with repro).

**Sibling repo / deploy**
12. Hand `module-draft.nix` to nix-international-telephony as
    `services.pbx-lens` (T7 half 2).
13. Port the backup service+timer drafts into the sibling module; verify the
    same-StateDirectory shared-dynamic-user claim on a real host.
14. Validate draft units with `systemd-analyze verify` + security
    score on a systemd machine.
15. Align templ CLI pin with the v0.3.1070 lib bump (flake).
16. Re-run `nix build` + `nix flake check` after the parallel templ bump;
    repair `vendorHash.nix` via `buildflow -s nix-hash-fix --fix` if
    go.sum moved.
17. Rebuild BuildFlow binary → re-run dev gate → drop AGENTS license-check
    note.

**Product surfaces (now that enrichment data lands)**
18. Dashboard cost stat card (total spend in window) — data now exists.
19. Detail page: show currency properly when absent (verify dash rendering
    has a test).
20. Export tests asserting CostMicros/InsightSummary round-trip.
21. List page: cost column (opt-in, small) — or explicitly decide against.
22. Search: full-number hint parity on the list filter (if numbers are
    filterable there later).
23. `ingest-telnyx -dry-run` (fetch/map/enrich without dispatch) for field
    diagnosis.
24. Insights display: render multi-result joins readably (currently raw
    `\n` join inside one definition detail).

**Robustness & tests**
25. Detail-records multi-page pagination test.
26. Conversation-insights multi-page pagination test.
27. Insights-fetch error path test (seed rejected, others proceed).
28. Detail-list fetch error path test (lazy fetch surfaces per-seed
    rejection).
29. `ingest-telnyx` command-level e2e with the fake server (double-drain
    wiring).
30. Enrichment idempotency across a correction (corrected event carries
    enrichment — has domain tests; add an e2e pass-over-correction).
31. Zero-cost call e2e (records with "0.000") — pin the no-display +
    re-fetch behavior deliberately.
32. `CallView.Observation()` StreamID equality assertion (currently implied
    via Validate).

**Internal quality**
33. Extract `telnyxtest` shared fake server package.
34. `fetchAll` options struct refactor.
35. Enrichment duration logging in pollTelnyx.
36. Structured logging (key-value) for poller passes.
37. art-dupl pass over the new code (warning-only gate).
38. `govulncheck` local run (CI covers; local gate parity).
39. DOMAIN_LANGUAGE.md: glossary TelnyxSeed, EnrichStats, Enrichment
    lifecycle, `conversations-insights` vocabulary.
40. AGENTS: refresh the UI-trio templ pin row post-bump.
41. README: mention enrichment + nightly backup staging in operations
    (sales-page-adjacent, owner's call).

**Hygiene / small**
42. TODO_LIST: add the live-verify row's sibling — insights status enum
    drift watch (upstream could add statuses).
43. Search hint: measure whether users actually hit it (log zero-result
    queries server-side? owner decision, privacy-sensitive).
44. `fakeTelnyx` conversations fixture: add a second conversation to prove
    multi-seed passes (currently single-seed e2e).
45. Poller: cap enrichment seeds per pass (YAGNI guard for a huge backlog
    after long downtime).
46. Consider `WithPrefetch` revisit note for the enrich reader (F026
    sibling).
47. Make `RetentionReport`-style summary structs a house pattern for CLI
    outputs (consistency with EnrichStats/IngestStats).
48. deploy/pbx-lens.service: cross-reference the backup unit in comments
    (discoverability).
49. CHANGELOG: note enrichment never enters ContentHash (v2 stability) —
    one sentence, prevents future fear.
50. Session-process: add "full-suite baseline before first edit" +
    "inventory untracked dirs" to AGENTS gotchas (this report's lessons).

## g) QUESTIONS I CANNOT ANSWER MYSELF

1. **Telnyx key for live verification:** is there a key (or a machine that
   has one) I may use to live-verify the Detail Records + insights DTOs and
   record sanitized fixtures — or should the verify pass run on the deploy
   host? Without it, the alias-tolerance stays a documented bet, not a
   fact.
2. **Parallel session boundary:** is the other session still active on T23
   (`internal/retention/`, `purge`), and do you want me to stay out of that
   package entirely, or take the PII-mode half once it lands?
3. **Zero-cost presentation:** when Telnyx reports a true $0.00 call, should
   the detail page show "0 USD" (requires distinguishing "fetched zero"
   from "not fetched" — a small model change) or is "no cost row" correct?
   This decides whether I add a cost-fetched marker to the model.

---

**State at close:** 11/11 packages green, lint 0 issues, cqrs-lint 0 ERROR,
smoke green, all session work daemon-committed (b351a6f..), working tree
clean except this report.
