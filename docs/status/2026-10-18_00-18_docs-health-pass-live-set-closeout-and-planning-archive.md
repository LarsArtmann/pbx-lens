<!-- docs/status/2026-10-18_00-18_docs-health-pass-live-set-closeout-and-planning-archive.md
     Point-in-time snapshot for the second 2026-10-17 docs-health pass
     (dispatch: "view ALL 2026-0* files, execute the docs-health skill,
     archive FULLY done and UPDATED (inline strikethrough) .md files").
     Work window: 2026-10-17 late evening → 2026-10-18 00:18 CEST. -->

# Status — docs-health pass: live-set close-out, planning archive, harvest (2026-10-18 00:18)

**Scope:** the whole live `2026-0*` corpus re-sweep after the morning's
Pareto close-out: all non-archived 2026-0* files read (4 status reports, 11
plans, 6 dated drafts, 3 decisions, followons, process-prep; the 4 HTML
snapshots and `status/archived/` stay closed history per the skill), every
verdict verified against code / CHANGELOG / `gh` before it landed.

## a) FULLY DONE

1. **ANNOTATE: 247 verdicts across the four 2026-10-17 reports** (40 + 64 +
   65 + 78), applied via `annotate.go status-items`/`rows`/`repair` with
   emit-keys + `--verify`; check-rows COMPLETE on all four BEFORE any move
   (the 2026-10-15 gate-order lesson held).
2. **ARCHIVE: the four reports → `docs/status/archived/`** — the live set
   is empty for the first time; completeness gate clean; per-file manifest
   rows added; `docs/status/README.md` (new) indexes live-vs-archived
   (05-01 §f44 executed).
3. **PLANNING ARCHIVE: 8 executed plans → `docs/planning/archived/`** with
   a manifest — this RESOLVES the question the 05-01 sweep staged as §g2
   (annotate-in-place vs move; the dispatch directive decided: archive
   fully-done files). 05-07 got its EXECUTED header first (08:02 + 10:28
   legs as execution records). 02-24 / 05-40 / mos-poller stay live on
   owner-gated or 2027-01 residue. Living-doc citations updated
   (AGENTS.md, TODO_LIST, http-stack.md); CHANGELOG historical entries
   keep their original paths (append-only wins).
4. **Ledger re-sync (VERIFY leg):** `docs/upstream-filings.md` gained the
   2026-10-17 gh-verified state check (BuildFlow#45/#50 — checked live for
   the first time — plus #55/#56, go-cqrs-lite#62-64, go-appkit#26 CLOSED;
   the rest open) and per-entry CLOSED tags; the drafts README states were
   re-synced (7 stale rows) and the two 2026-10-15 drafts gained their
   missing ledger rows + FILED banners (#55 statuses corrected).
5. **HARVEST: the 10-28 §f 19+ queue triaged** into six new bounded
   TODO_LIST rows (gate-hardening bundle incl. the never-routed
   `TestCacheSemanticsHardening` CI flake, upstream-filings queue,
   probe/script polish, release/tooling rehearsals, lint-policy intent,
   hooks+budget wiring, AGENTS extraction pre-plan) + the CSP enforce-mode
   BLOCKED row + ten ROADMAP raw ideas — nothing entombed in the report.
6. **Fix-on-sight:** README now mirrors `ingest-cdr -dry-run`; the gate
   doc records the rejected W20-restyle alternative (01-45 §f24's claim
   was false — the note had never landed); answer sheet §11 (CSP) staged.
7. **Living docs:** TODO_LIST preamble refreshed (86 rows→80 lines, 13
   TODO + 17 blocked rows, all evidence-cited); ROADMAP theme-2 gained the
   process/ops raw-ideas block; AGENTS.md 219/219 (archived counts +
   planning-archive pointer); CHANGELOG Unreleased entry appended.

## b) PARTIALLY DONE

1. **Full-gate receipt** — still load-gated (1-min 92 > 15 at pass start);
   `go build ./...` + `go test ./...` stand in per the docs-pass precedent
   (05-01 §b3). The row stays first in TODO_LIST Unblocked.
2. **dprint scope** — living docs + the 4 newly annotated archived files
   only; the other 50 archived reports are deliberately left to the
   ROADMAP dprint-policy question rather than reflowed again.

## c) NOT STARTED (routed, with landing places)

1. The six new TODO rows above (bounded; owner-unblocked except where the
   row says owner).
2. Owner answer sheet §1-§11 — all eleven questions staged in ONE file.
3. The 2027-01 Telnyx window + fold-in watch (checklist + ledger ready).

## d) TOTALLY FUCKED UP (caught; none landed)

1. **Hand-built spec keys failed 9 times across three files** — wrapped
   substrings, wrong item numbers (§g2/§g3 keyed as 1), and lines where
   spaces replaced the TAB separator. All caught by `--verify`/unmatched
   keys before any write (the tool's atomicity did its job); the fix was
   always the same: emit-keys/line-map first, spec second.
2. **AGENTS.md budget blown mid-pass** (223 > 219) — the archived-count
   edit was written longhand. Caught by `check-agents-budget.sh`
   immediately; compressed back to one line (219/219).
3. **`grep -L` completeness misread** — nearly declared the gate failed on
   an exit-code artifact instead of reading the (empty) file list.
   Per-file `-c` re-verification settled it.

## e) WHAT WE SHOULD IMPROVE

1. Spec construction: the ONLY reliable path is emit-keys → specfile with
   real TABs → `--verify` → apply. Every failure this pass was a shortcut
   around that pipeline.
2. The `any:` bullet-list verdicts (08-02 §c) work but bypass the numbered
   matcher — prefer reports that number their queues.
3. Postscript claims need the same verification as body items: 01-45's
   "Task 26 added to (f)" was false (the row never landed) and the flake
   triage it promised had silently no landing place for ~14 hours.

## f) What next (all routed — this list is the ledger, not the backlog)

TODO_LIST is the backlog (13 TODO + 17 blocked rows). The next session
starts from: the gate receipt at low load (first Unblocked row), the six
new bounded rows, and the owner answer sheet. No unharvested reports
remain anywhere in the repo.

---

_Point-in-time snapshot 2026-10-18 00:18 CEST. Evidence: check-rows 4/4
COMPLETE pre-move, `grep -c '~~'` = 40/69/65/78 on the archived set,
219/219 budget green, daemon-swept tree (git status clean of foreign
entries). Living backlog: TODO_LIST.md; this report is the single live
harvest source._
