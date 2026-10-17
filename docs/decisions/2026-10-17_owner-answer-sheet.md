# Owner Answer Sheet — one pass, seven answers (P0-T3 staging)

- **Created:** 2026-10-17 (Pareto plan P0-T3). consolidates every open owner
  question into ONE answerable sheet: D1-D5 (UI direction), the PII
  enforcement boundary, and the mobile card-list taste call.
- **How to answer:** write the choice ID (or free-form) after **→ ANSWER:**
  on each question. On arrival, outcomes are recorded into TODO_LIST +
  AGENTS.md per the plan-F7 contract and the unblocked work starts.
- **Full context per question:** D1-D5 in
  `2026-10-15_ui-owner-decision-pack.md`; PII in
  `2026-10-15_pii-enforcement-one-pager.md` (both in this directory); mobile
  in the 02-24 SUPERB plan M17 (F56-F58).

---

## 1. D1 — Date/number vocabulary → ANSWER:

A keep US · B EU format (09.10.2025 22:31) · C hybrid (EU dates + relative text).
**Recommendation: B** — the UI already thinks in Europe/Berlin. Gates a
one-decision all-surfaces date change (RelativeTime, hits, detail, tooltips).

## 2. D2 — Dashboard control rows → ANSWER:

A keep two rows (presets / range form stacked) · B merge into one denser row.
**Recommendation: A** — the controls have different scopes; one-templ change either way.

## 3. D3 — Volume chart in quiet weeks → ANSWER:

A spike-only reading is fine · B file the BarChart baseline/axis request upstream ·
C local baseline rule under the bars.
**Recommendation: B** (C only if the operator actually watches the chart in quiet stretches).

## 4. D4 — Export honesty note → ANSWER:

A keep visible · B collapse (always-open-by-default CollapsibleSection).
**Recommendation: A for now** — the note is short; collapsing saves less than it risks.

## 5. D5 — AI-share card tone → ANSWER:

A keep the purple accent (distinct AI identity) · B switch to teal/info (full signal-palette uniformity).
**Recommendation: A** — deliberate differentiator, already on the version-bump watchlist.

## 6. PII enforcement boundary → ANSWER:

A enforce at INGEST (storage normalized to E164 + hashes; ContentHash v2 re-ingest wave) ·
B enforce at READ (storage stays raw; `calls.E164` becomes the single rendering path +
a palette-gate-style test pinning it).
**Recommendation: B** — zero migration, source-truth doctrine intact, 90% of A's
value for 10% of its cost. Details: the one-pager.

## 7. Mobile calls list (375px) → ANSWER:

A keep the table, allow horizontal scroll (zero new markup) ·
B card-list below the breakpoint (the M17 sketch: who/when/first line stacked) ·
C hybrid (cards < 640px, table above).
_Taste call — no assistant recommendation; B/C follow the 02-24 plan sketch,
A is the zero-risk default._

---

**Answered on (date):** ____________ · **Recorded into TODO_LIST + AGENTS by:** ____________

## 8. CI coverage floor → ANSWER: 70 (implemented by default)

Floor raised 50→70 in `.github/workflows/ci.yml` on 2026-10-17 (measured
total 73.0% after the cmd wiring tests). Chase-to-80 stays open as the
TODO_LIST owner row; say "chase" to reopen the work.

## 9. BuildFlow#56 binary rebuild → superseded upstream

BuildFlow#56 (gate tools trail the module go floor) CLOSED 2026-10-17 —
the fix landed. The open question shrinks to bookkeeping: does the LOCAL
buildflow binary still want the manual `nix build . && nix run
.#reinstall` pass? → ANSWER:

## 10. Pixel-baseline home → ANSWER:

Baseline v2 saved 2026-10-17 at `/home/lars/lens-baselines/v2`
(machine-local, post-h2-change, diffs clean 0 alerts / 41 PNGs). Options:
A stay machine-local (re-save per machine) ·
B commit the corpus under `docs/assets/baselines/` (reviewable diffs,
repo weight ~10-20MB) ·
C artifact-store per CI run.
**Recommendation: A** for a single-host internal tool; revisit if a
second operator machine starts touching the UI.

## 11. CSP enforce-mode flip → ANSWER:

The report-only sweep is enforce-clean in the probe browser (zero
violations across 6 surfaces after the nonce fixes). Enforce-mode breaks
pages silently for real users if any surface was missed — only you know
the operator risk tolerance for that. Options:
A stay report-only until real-browser usage exists on the deployed hosts ·
B flip to enforce in v0.3.0 after one more clean sweep (the flip bundles
with the next host deploy either way).
**Recommendation: A** for a private internal tool; the ROADMAP
collector-endpoint idea would surface any real violations first.
(Staged from the 10-28 close-out report's §g Q3, 2026-10-17.)
