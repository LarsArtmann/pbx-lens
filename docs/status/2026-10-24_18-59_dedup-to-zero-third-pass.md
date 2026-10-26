# Status Report — 2026-10-24 18:59 CEST — Deduplicate-to-Zero, Third Pass

**Scope:** This session only (user directive: no unrelated research). The session had one
task: `art-dupl --sort total-tokens -t 10 --timing --html` reported 4 detected clone
groups (3 shown, 1 suppressed); instruction was "deduplicate to zero".

**Session lens vs repo gate (read this first):** the user's report ran `-t 10` with no
`--min-lines`; the enforced CI gate is `art-dupl check -t 4 --min-lines 2` against a
zero-group baseline. The gate deliberately ignores 1-line signature matches (shape
coincidence, not duplication). "Zero" here means **zero harmful clones**; it does NOT
mean zero report lines at the t-10 lens.

---

## a) FULLY DONE

| #  | Item                                                                                                                                                                                                                                                                                                     | Evidence                                                                                        |
| -- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------- |
| ~~1~~ done (docs-health pass 2026-10-26) | ~~Skill loaded before acting (`deduplicate-code`); user's exact threshold (`-t 10`) honored~~ | ~~session start~~ |
| ~~2~~ done (docs-health pass 2026-10-26) | ~~Suppressed bucket audited — found the hidden 4th group: a **27-line verbatim harness block** copied across both poll-pass tests in `internal/cli/poller_test.go` (invisible to the default report AND to the CI gate, because test files are auto-excluded from `check`)~~ | ~~`art-dupl --show-suppressed` output~~ |
| ~~3~~ done (docs-health pass 2026-10-26) | ~~All 4 groups read body-by-body and judged: 3 production groups = 1-line signature-shape matches with different documented business rules → **accept**; test group = real clone → **extract**~~ | ~~calls.templ:392/410, detail.templ:443/458, dashboard.templ:624/671, poller_test.go:30-57/77-104~~ |
| ~~4~~ done (docs-health pass 2026-10-26) | ~~Extraction: `openPollHarness` (+ `webhookHangupLine`) in `internal/cli/poller_test.go` — memory system + search index + poll-1 webhook fixture in one helper; both pass tests now one-line harness calls; fixture bytes **identical** to the old inline literal (verified by construction + green tests)~~ | ~~poller_test.go:226-266~~ |
| ~~5~~ done (docs-health pass 2026-10-26) | ~~Third inline fixture literal (counters test, `ops-1`) consolidated onto the existing `webhookPollLine` helper — byte-identical output, no drift risk~~ | ~~poller_test.go:238-241~~ |
| ~~6~~ done (docs-health pass 2026-10-26) | ~~**Lying comment fixed**: `webhookPollLine` claimed to render "one initiated+hangup webhook pair" but rendered only the initiated line; now documents reality and the pairing~~ | ~~poller_test.go:222-225~~ |
| ~~7~~ done (docs-health pass 2026-10-26) | ~~Verification: `go test ./...` green (15 packages); `golangci-lint run ./internal/cli/...` clean (after fixing its 2 findings); `art-dupl -t 10` now reports only the 3 accepted groups; **CI gate `check -t 4 --min-lines 2` still 0 groups, exit 0**~~ | ~~session run output~~ |
| ~~8~~ done (docs-health pass 2026-10-26) | ~~Docs updated: `docs/art-dupl-baseline.md` — test-scaffolding bullet now names both extracted full-stack repetitions (`serveTestStack`, `openPollHarness`) and explains WHY the poller clone was gate-invisible; third-pass history entry added~~ | ~~docs/art-dupl-baseline.md~~ |
| ~~9~~ done (docs-health pass 2026-10-26) | ~~Formatting verified: `dprint check` clean on both touched files (via `nix develop`; dprint is not on host PATH)~~ | ~~session run~~ |
| ~~10~~ done (docs-health pass 2026-10-26) | ~~Change-set committed by the auto-daemon (e.g. `dbdfaf8`, 2 files) — harness contract honored, no manual commit~~ | ~~git log~~ |

## b) PARTIALLY DONE

| # | Item                                            | State                                                                                                                                                                                                                                                                                     |
| - | ----------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| ~~1~~ done — confirmed — zero-harmful IS the contract (the gate, the baseline doc, the AGENTS bullet) | ~~"Deduplicate to zero"~~ | ~~Zero **harmful** achieved. The t-10 report still shows 3 accepted signature groups by design. Zero _visible_ would require renaming parameters or merging genuinely different duration formatters — judged worse code, not done. This interpretation is not user-confirmed (question g1).~~ |
| ~~2~~ done — closed — later FULL gates ran green over the change-set (2026-10-25/26 receipts) | ~~Quality-gate coverage of the session change-set~~ | ~~Scoped: golangci on `./internal/cli/...` + full test suite. NOT run: `buildflow --build-mode dev` (the full local gate incl. all formatters) and full-repo `golangci-lint run ./...` (repo rule: before pushing — no push happened).~~ |
| ~~3~~ done — closed — the v1.24.0 sweep ran the deeper suppressed audit; the corpus re-derived at the 2026-10-26 pin | ~~Duplication sweep depth~~ | ~~Audited only at the user's t-10 lens. The skill's deeper variant (`art-dupl --type-aware -t 4 --show-suppressed`) was not run; whether it surfaces more suppressed items is **unknown**. The gate's multi-line contract is green, so any extra finds would be scaffolding-class.~~ |

## c) NOT STARTED (deliberate waits, not omissions)

| # | Item                                                                                                        | Why waiting                                                                                 |
| - | ----------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------- |
| ~~1~~ done — later harvests plus this pass | ~~`docs-health` **HARVEST** of section (f) into `TODO_LIST.md`~~ | ~~User instruction: report, then wait. Skill says harvest when the session continues.~~ |
| ~~2~~ done — moot — the daemon carried it; the harness rule was honored | ~~Manual receipt-grade commit~~ | ~~Harness rule: never commit unasked; auto-daemon already picked the files up.~~ |
| ~~3~~ done — decided — test-only; the baseline doc and the CHANGELOG dedup entries carry the history | ~~CHANGELOG entry decision~~ | ~~Change is test-only + doc; likely no entry, but not decided (item f4).~~ |
| ~~4~~ done — superseded — the systemscenario wave replaced the fixture landscape (2026-10-25) | ~~Fixture-consolidation sweep beyond `poller_test.go` (other test packages may carry verbatim fixture blocks)~~ | ~~Out of this session's report scope; candidate item f7.~~ |
| ~~5~~ done — event-driven per the baseline doc; ran with the v1.24.0 sweep | ~~Deeper `--type-aware -t 4` suppressed audit~~ | ~~Event-driven per baseline doc (pin bumps / suspicious greens); none triggered. Question g2.~~ |

## d) TOTALLY FUCKED UP

**Nothing at that severity this session.** Near-misses, honestly:

1. ~~First `golangci-lint` run **failed** (golines: over-long line in the counters test; wsl_v5: missing whitespace in the harness) — caught and fixed in the immediately following step; final lint clean.~~ **NOT-DO — process note; caught and fixed in the next step.**
2. ~~The lying `webhookPollLine` comment **pre-dated this session** and shipped contradicting its own code; found and fixed only because the dedup pass touched the file.~~ **NOT-DO — process note; found and fixed by this pass.**
3. ~~`dprint` missing from host PATH — markdown-formatting verification was initially skipped and only recovered via `nix develop`. Process friction worth remembering: **formatting tools live in the devShell**.~~ **NOT-DO — process note; the devShell rule is recorded.**

## e) WHAT WE SHOULD IMPROVE

1. ~~**Run the full local gate once per session** that touches any tracked file — scoped lint+test missed nothing today, but `buildflow --build-mode dev` is the contract and costs one command.~~ done — adopted — the FULL-gate receipt is the contract (poll-gate era)
2. ~~**State the lens delta up front.** The user's report (t-10, no min-lines) and the enforced gate (t-4, min-lines 2) measure different things; I explained this mid-flight — it belongs in the first message.~~ done — adopted — lens deltas are stated up front in later reports
3. ~~**Verify docs with the repo formatter in the same verification step as Go lint**, not as an afterthought (dprint miss above).~~ done — adopted — formatter receipts sit in the verification battery
4. ~~**Accepted-groups rationale for the t-10 view lives only in prose.** The gate enforces zero groups it can see; the t-10-visible accepts are doc-only. Baseline doc now at least enumerates them — keep that list fresh when signatures drift.~~ done — held — the baseline doc enumerates the accepts; kept fresh through the v1.24.0 sweep
5. ~~**Keep suppressed audits event-driven** (pin bumps, suspicious greens) to avoid ritual creep — today's audit was triggered by a user report and paid off; automating it into every session would not have.~~ done — adopted — event-driven stands (AGENTS CI bullet)

## f) Up to 50 things to get done next

Brainstorm, not commitment list. Provenance marked: **[S]** session-derived, **[B]** standing backlog known from repo docs, **[R]** roadmap fuel. Extra items beyond the default 25 need `docs-health` HARVEST routing rigor (most are ROADMAP fuel).

**Immediate follow-ups (this change-set):**
~~1. [S] Harvest this report's (f) into `TODO_LIST.md` (docs-health HARVEST).~~ h ec89cce
~~2. [S] Run `buildflow --build-mode dev` on the session change-set before next push.~~ v gate run completed with a starvation verdict (nix-build deadline at loadavg 50-66, concurrent session); all five casualty classes individually verified green per docs/buildflow-dev-gate.md
~~3. [S] Full `golangci-lint run ./...` before next push (repo rule).~~ h ec89cce
4. [S] Decide CHANGELOG: entry for the harness extraction, or explicitly skip (test-only).
5. [S] Answer g1 (zero-harmful vs zero-visible) and record the verdict in the baseline doc.
6. [S] Answer g2 (deep suppressed audit now vs event-driven) and record.

**Dedup program continuation:**
7. [S] Sweep other test packages for verbatim fixture blocks like the poller one (candidates unverified).
8. [B] On next `internal/web` test changes, verify all full-stack tests actually use `serveTestStack`.
9. [B] Keep the zero-group gate contract: new multi-line clone → extract, never baseline-accept without review.
10. [B] After any art-dupl pin bump: regenerate baseline + suppressed audit in the same change (pin parity).
~~11. [S] Baseline doc: make explicit that `check` auto-excludes test files (why the poller clone was invisible) — currently implied, not stated.~~ v explicit since the third pass ("invisible to the gate because test files are auto-excluded" in docs/art-dupl-baseline.md)
12. [B] If a third function ever shares one of the accepted signatures, re-litigate that accept (drift trigger).
13. [B] Keep `--min-lines 2` calibration; any real 1-line clone class → recalibrate with proof both directions.
14. [B] Quarterly full suppressed-bucket review as cheap hygiene (calendar, not ritual).
~~15. [S] Consider noting the third pass in the AGENTS.md art-dupl bullet (currently says "second pass" only) — minor, optional.~~ v done 2026-10-25 (AGENTS.md art-dupl bullet now says "second pass ... third pass same day")
16. [B] When ingest mappers next change, re-check `internal/ingest` for clone pressure (historically clean; watch only).

**Docs health loop:**
17. [S] ANNOTATE this report (non-destructively) as its items land.
18. [S] Ensure harvested items land in `TODO_LIST.md` with status, not in the timestamped file alone.
19. [B] FEATURES.md unaffected this session (no feature change) — verify on next feature work.
20. [S] Keep `docs/art-dupl-baseline.md` and the AGENTS.md art-dupl bullet in sync on gate mechanics.
21. [B] Archived reports stay CLOSED — never harvest or re-open (standing).
22. [B] Living backlog = `TODO_LIST.md`; this report is a snapshot and will go stale.

**Standing backlog (from repo docs, unverified this session):**
23. [B] Telnyx live-API quarterly re-verification (recordings verified 2026-10-12; window 2027-01, m27.9).
24. [B] Telnyx enrichment path: spec-verified only — live-verify when the window opens.
25. [B] templ-components version-bump watchlist (gen-tc-classes → map re-audit → hand-rolled re-check → filings re-read → golden `-update`, in order) on next bump.
26. [B] Guard the dependabot lane: templ bumps are manual-only (a templ PR mangles this go.mod).
27. [B] golangci-lint pin vs go language-level parity on next bump (v2.14 = go1.27.1 currently).
28. [B] After any app.css change: rebuild ONLY via `nix run .#update-styles` (foreign build reds the CSS gate).
29. [B] go.mod/go.sum changes: `nix run .#update-vendor-hash` on a CLEAN tree; never nix-hash-fix.
30. [B] E003 second aggregate: hold-flat — revisit only on its recorded triggers.
31. [B] `retention.Policy.Admits` gates serve-poller `-max-age` — re-verify interaction on retention changes.
32. [B] auth.go invariant: htpasswd timing pad hashes at the LOADED entries' bcrypt cost — preserve when touching.
33. [B] Per-client-address throttling (bounded FIFO) — revisit only if abuse patterns change.
34. [B] Exports/recording stream unbounded (`WriteTimeout: NoTimeout`) — revisit posture if exposure widens.
35. [B] htmx cadences (dashboard 30s, staleness 60s) — tune only on staleness-UX evidence.
36. [B] Consult docs/gotchas.md before any search/FTS5/projection work (standing rule).
37. [B] New projections keep BARE record keys via `calls.LookupView` (standing invariant).
38. [B] server_timing: no middleware constructor upstream — manual wrap stays until a release ships one.

**Testing depth:**
39. [B] Re-run the poll benchmark (≈82 µs/op 304, ≈101 µs full-200) after any serve-path change.
40. [B] Browser lane `scripts/verify-all.sh` before any UI-visible change (not needed this session — test-only).
41. [B] Pixel-diff baseline refresh discipline on dashboard visual changes (footer stamp ≈0.04% absorbed).
42. [B] Golden `-update` only after the full templ-components bump checklist order.
43. [R] Consider fuzz/property tests for the ingest parsers (cdr/webhook/agent) — candidate, existence unverified.
44. [R] Consider locking the poll fixture wire shape with a byte-exact assertion — likely over-engineering; judge later.
45. [S] Poller idempotence test now shares the harness — extend (third pass, changed-hash bug path) only if the spec grows.
46. [B] Age-cutoff test intentionally writes initiated-only lines — keep the helper-pairing rationale documented.

**Roadmap fuel:**
47. [B] PII policy: `E164` + `ParticipantHash` exist but are NOT enforced on stored fields — the planned enforcement work.
48. [R] Webphone read-only deep link (sibling integration, no data dependency) — north-star item.
49. [B] nix-international-telephony lens module: keep the deploy module in sync with serve config knobs (Version/Metrics/WriteTimeout).
50. [B] Periodic progress review against the SUPERB plan (docs/planning/2026-10-09_20-28_*).

## g) Questions I cannot figure out myself

1. **What does "zero" mean to you?** Zero _harmful_ clones (current state: 3 accepted 1-line signature groups still visible at t-10, documented as shape coincidence) — or zero _visible_ groups at t-10, which would force parameter renames or merging genuinely different formatters (`clockDuration`/`compactDuration`, `mosBars`/`volumeBars`) that I judge would make the code worse?
2. **Deep audit appetite:** run `art-dupl --type-aware -t 4 --show-suppressed .` now (the deeper lens the skill recommends; the gate cannot see test files, so more scaffolding-class finds may surface), or keep suppressed audits event-driven (pin bumps / suspicious greens) as the baseline doc prescribes?
3. **Commit handling:** keep the auto-daemon's heuristic commits for this change-set, or do you want a manual receipt-grade commit (single `git add … && git commit`, amend only unpushed)?

---

## Post-session correction (appended 2026-10-24, same day — non-destructive)

This report's acceptance of `clockDuration`/`compactDuration` as intentional
shape coincidence is **superseded**: the afternoon templ-components deep dive
(docs/research/2026-10-24_templ-components-deep-dive.html, finding F1) showed
`utils/format` at the pin ships both helpers (plus `StringOrDash`/`Percent`).
They were undetected dependency duplication, not shape coincidence; question
(g).1's premise ("merging genuinely different formatters") was wrong — they
were the SAME formatter, hand-rolled twice. All three locals were deleted
when utils/format was adopted (plan T4,
docs/planning/2026-10-24_19-32_SUPERB-data-truth-and-ui-debt-plan.md);
`mosBars`/`volumeBars` and `mosGaugePercent` keep their accepted verdicts.
Lesson: grep the dependency graph before accepting a signature pair.

---

_Point-in-time snapshot; goes stale by design. Harvest section (f) into `TODO_LIST.md` via docs-health when the session continues; annotate this file non-destructively as items land._
