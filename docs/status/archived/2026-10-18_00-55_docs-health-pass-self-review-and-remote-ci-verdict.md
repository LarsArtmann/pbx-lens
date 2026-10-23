<!-- docs/status/2026-10-18_00-55_docs-health-pass-self-review-and-remote-ci-verdict.md
     Self-review + status for the second 2026-10-17 docs-health pass (the
     live-set close-out + planning archive), written on explicit demand.
     Companion to the pass report 2026-10-18_00-18 (same sweep). -->

# Status — docs-health pass self-review: what was forgotten, what broke, remote CI verdict (2026-10-18 00:55)

**Scope:** this session only — the "view ALL 2026-0* files / execute
docs-health / archive FULLY done files" dispatch, its execution (247
verdicts, 4 reports + 8 plans archived, living docs rebuilt), and the
honest accounting below. No unrelated research.

**Headline state:** local main 98899f7 + the last two doc edits (daemon
still sweeping), **remote CI GREEN on the repaired tail** (run 39076978166
success, 2m18s — the styles.css repair and the docs pass verified
remotely), tests 14/14, check-rows 4/4 COMPLETE, dprint converged with the
annotation canon, AGENTS.md 219/219.

## a) FULLY DONE

~~1. **The whole corpus sweep** — all non-archived `2026-0*` files read~~ done — reports annotated + archived this pass
before any verdict; 247 inline verdicts across the four 2026-10-17
reports (40/69/65/78 markers), all applied through `annotate.go`
(emit-keys → spec → `--verify` → apply); check-rows ran COMPLETE
BEFORE each `git mv` (the 2026-10-15 gate-order lesson held).
~~2. **The live set is EMPTY**: 4 status reports archived (54 total) and~~ done — live set emptied; 54 archived then, 59 after the 2026-10-23 pass
8 executed plans moved to `docs/planning/archived/` with a manifest —
decisively answering the question the 05-01 sweep staged as §g2
(annotate-in-place vs archive). The three plans with owner-gated or
2027-01 residue (02-24, 05-40, mos-poller-design) stay live on purpose.
~~3. **Ledger re-sync verified against `gh` (not memory):** BuildFlow#45 and~~ done — ledger re-synced; states re-verified with gh on 2026-10-23
#50 — never before state-checked — confirmed CLOSED (2026-10-12),
alongside #55/#56, go-cqrs-lite#62-64, go-appkit#26; the filings doc
and drafts README now carry the 2026-10-17 state check, and the two
2026-10-15 drafts gained their missing ledger rows + FILED banners.
~~4. **HARVEST with no entombment:** the 10-28 §f 19+ queue triaged into six~~ done — rows landed; list rebuilt 2026-10-23
bounded TODO rows, the CSP enforce-mode decision staged as answer-sheet
§11 (+ its BLOCKED row), ten ROADMAP raw ideas, `TestCacheSemanticsHardening`
(flake-flagged for ~14 h with no landing place) folded into the
gate-hardening row.
~~5. **Fix-on-sight class executed:** README mirrors `ingest-cdr -dry-run`;~~ done — README mirrors -dry-run; answer sheet §11 staged
FEATURES flag row; the gate doc records the rejected W20-restyle
alternative (01-45 §f24's claim had silently never landed); answer
sheet §11; `docs/status/README.md` (the 05-01 §f44 live-set index).
~~6. **The inherited CSS-gate break repaired:** `f91f3f4` (landed AFTER the~~ done — repaired via nix run .#update-styles; remote CI green
morning's green verification, daemon-pushed at 10:45) carried a foreign
UNMINIFIED styles.css (+5414 lines) and pushed remote CI to RED.
Repaired via the sanctioned `nix run .#update-styles`; **remote CI
re-verified GREEN** on the repaired tail (run 39076978166). The
styles-freshness CI job did exactly its job on the red SHA.
~~7. **Living docs rebuilt to fit:** TODO_LIST (80 lines: 13 TODO + 17~~ done — TODO_LIST rebuilt 2026-10-23 (32 open rows)
blocked, every row evidence-cited), ROADMAP process/ops block, README,
CHANGELOG (3 new Unreleased entries incl. the styles fix), AGENTS.md
219/219 with the planning-archive pointer.

## b) PARTIALLY DONE

~~1. **The full-gate receipt** — load-blocked all session (1-min 92–112 vs~~ done at `b4a9970`
policy < 15); `go build` + 14/14 tests + dprint + check-rows stand in
per the docs-pass precedent. The literal 70/70 clean-checkout receipt
remains the first TODO_LIST Unblocked row.
2. **dprint ↔ archived-docs relationship** — formatted only the touched **→ ROADMAP (dprint policy for archived docs)**
files; the dprint policy for `docs/status/archived/` is now a staged
ROADMAP question instead of a recurring reflow accident, but UNDECIDED.
3. **Remote sync** — the daemon pushes everything (verified green), but **→ ROADMAP (branch protection idea)**
branch protection / HEAD-green-only remains a raw idea; mid-flight red
SHAs (bafd463, f91f3f4) stay permanently in history.
~~4. **The `.githooks` activation** — script + pre-commit exist and are~~ done — hooksPath set 2026-10-18 and documented in CONTRIBUTING.md
verified; activation is the owner's one `git config core.hooksPath`
line (routed row).

## c) NOT STARTED (all routed, each with a landing place)

1. The six new TODO rows (gate hardening incl. the CI nix lane + statix **→ TODO_LIST (all rows landed or live there)**
   tripwire + config-verify; upstream filings queue; probe polish; release
   rehearsals; lint-policy intent; hooks/budget wiring; AGENTS extraction).
2. Owner answer sheet §1-§11 — eleven staged questions, zero answers. **→ docs/decisions/2026-10-17_owner-answer-sheet.md (zero answers)**
3. The 2027-01 Telnyx window + fold-in watch (checklist + ledger ready). **→ TODO_LIST (2027-01 window row)**
   ~~4. The planned-status-report HTML-vs-md convention note (this report~~ done — reports are .md; ROADMAP records the standing policy
   follows the standing .md default; nothing to do).

## d) TOTALLY FUCKED UP (own goals, all caught before landing)

~~1. **I repeated the exact spec-construction failure class the 05-01~~ **NOT-DO — process note only; pipeline is mechanical now.**
report paid six failed applies to teach.** Nine failed applies across
three files: hand-typed substrings that straddled line wraps, §g items
keyed `1@` when they were items 2/3, and a heredoc that wrote literal
`\t` instead of TAB separators. The tool's atomicity + `--verify`
caught every one before any write — but I had emit-keys output in hand
and still typed keys by hand. The pipeline (emit-keys → mechanical
spec → verify → apply) must be ONE scripted step, never manual.
~~2. **AGENTS.md budget blown mid-pass** (223 > 219) — wrote the~~ **NOT-DO — process note only; budget green at 219/219.**
archived-count edit longhand, caught only by running the budget check
afterward. The check exists; I should have run it as part of the edit,
not after.
~~3. **Used `curl`** in a verification chain — banned command, blocked by~~ **NOT-DO — process note only.**
the security layer. Sloppy; the rule is written in my own instructions.
~~4. **Misread the completeness gate** — declared `grep -L` a failure on~~ **NOT-DO — process note only; per-file counts are the norm.**
its exit code instead of reading its (empty) file list; per-file `-c`
re-verification settled it. Exit codes of grep -L are not intuitive;
use the per-file count form first next time.
~~5. **Almost shipped routed table rows unstruck** — my first rows spec~~ **NOT-DO — process note only; routed tables are struck, not arrowed.**
used kind `r` (append, no strike), which would have made the archived
tables PARTIAL under check-rows. Caught by reasoning + a corpus grep
BEFORE applying, but the routed-in-tables canon was in the archived
corpus all along; I should have looked before building the spec.
~~6. **Ran the test suite outside the devShell first** — the CSS-gate~~ **NOT-DO — process note only; the CSS repair is in CHANGELOG 2026-10-17.**
failure sent me down the ambient-shell hypothesis before the real
cause (the foreign styles.css landing). One wasted diagnosis cycle;
the devShell-first habit would have been wrong too (it failed there
as well), but checking `git log -- styles.css` should have been step
one, not step three.

## e) WHAT WE SHOULD IMPROVE

1. **Make spec construction mechanical for real:** a tiny wrapper that **→ crush-config tooling, not this repo**
   takes a file + section and emits a ready specfile skeleton from
   emit-keys would delete failure class (d.1) entirely. Candidate for a
   skill-side tool PR (annotate.go already has all the pieces).
   ~~2. **Check `git log -- <suspect-file>` BEFORE environment hypotheses**~~ **NOT-DO — process note only; applied in the 2026-10-23 pass.**
   when a "pre-existing" test breaks in a tree the daemon commits to —
   the styles.css root cause was one command away.
   ~~3. **Foreign-session pre-flight is still not a habit:** the CSS break WAS~~ done — pre-flight executed 2026-10-23 (status + diff read before edits)
   a concurrent session's edit landing mid-stream; the ROADMAP
   multi-agent coordination norm (git status pre-flight, ship-state
   verification) exists but was not consciously executed this pass.
   ~~4. **Load the skill references, not just the SKILL.md:** the routed-in-~~ **NOT-DO — agent norm, not a repo artifact.**
   tables canon lived in references/resolving-items.md / the corpus; I
   reconstructed it from the corpus instead. Loading the reference
   earlier is cheaper than corpus archaeology.
   ~~5. **The daemon needs a lint-docs lane:** it pushed a red styles.css and~~ done — daemon routes through the pre-commit hook chain; Styles Freshness is the post-hoc guard
   will push any future doc damage silently; the CI styles-freshness job
   catches it post-hoc, but the daemon-side budget/hook wiring (the new
   TODO row) is the real fix.
2. **CI red-window accounting:** between f91f3f4 (10:45 red) and this **→ ROADMAP (session-start drift guard)**
   repair (~00:45), remote main was red for ~14 h on a file nobody was
   watching — the session-start drift guard (gh run list + one package
   test, now a ROADMAP idea) would have caught it in seconds.

## f) Up to 50 things to get done next (impact-ordered; 1-10 are the queue)

~~1. Run the full buildflow gate at low load → bank the 70/70 receipt;~~ done at `b4a9970`
retire the v0.2.0 workaround note (first TODO row).
2. Owner: answer the answer sheet §1-§11 (D1-D5, PII, mobile, coverage, **→ docs/decisions/2026-10-17_owner-answer-sheet.md**
BF#56 bookkeeping, baseline home, CSP flip) — every blocked High row
gates on this.
~~3. Gate-hardening bundle: statix no-op tripwire, CI nix lane, golangci~~ done — gate-hardening bundle landed 2026-10-18 (tripwire, CI nix lane, flake fix)
config verify, TestCacheSemanticsHardening flake triage.
4. Upstream filings queue (verify-before-filing first): CopyButton nonce **→ TODO_LIST (filings queue row)**
gap, Card TitleTag h3 default, statix exit-0 footgun, F009 question.
~~5. Probe/script polish bundle (csp-sweep attribution, print-probe assert,~~ done — T14-T18 bundle landed 2026-10-18
--baseline wrapper, art-dupl baseline, freshness warning, scrubber
check, README merge).
~~6. Release/tooling rehearsals: cut-release.sh full dry-run, ingest-telnyx~~ done — T13/T14 landed 2026-10-18
-dry-run parity, .buildflow.yml statix contract comment.
7. Lint-policy intent: confirm or formalize the 3d93fdf edits. **→ TODO_LIST (lint-policy row)**
~~8. Hooks wiring: activate .githooks + daemon-path budget check + optional~~ done — hooks + daemon routing landed; CI variant routed
CI budget job.
~~9. AGENTS.md extraction pre-plan (recording-proxy details → http-stack).~~ done — T19 extraction landed (AGENTS 219)
10. Host deploys (owner) via the sibling module; then the ContentHash v2 **→ TODO_LIST (deploy + re-ingest rows)**
re-ingest with the -dry-run rehearsal first.
11. 2027-01 Telnyx window: run the quarterly checklist top to bottom; **→ TODO_LIST (2027-01 window row)**
confirm the calendar entry now.
12. T15 MOS poller execution in the same window. **→ TODO_LIST (MOS poller row)**
13. PII enforcement at the read boundary (after the owner picks A/B). **→ TODO_LIST (PII row)**
14. Mobile card-list after the taste call; D3/D4/D5 follow-throughs. **→ TODO_LIST (mobile row)**
15. CSP enforce-mode flip after the owner call (bundled with a deploy). **→ TODO_LIST (CSP row)**
16. dprint policy for archived docs — decide once, encode in dprint.json. **→ ROADMAP (dprint policy)**
17. Branch protection HEAD-green-only (GitHub settings). **→ ROADMAP (branch protection)**
18. vulnix closure scan + nix-checker cleanup at the next flake bump. **→ ROADMAP (vulnix idea)**
19. Off-hours gate timer or a first-session gate rule. **→ ROADMAP (off-hours gate timer)**
20. Session-start drift guard (gh run list + one package test). **→ ROADMAP (drift guard)**
~~21. Skill-side spec-builder wrapper for annotate.go (e.1).~~ **NOT-DO — the tool lives in the docs-health skill, not this repo.**
~~22. Art-dupl verdict baseline file.~~ done — scripts/art-dupl-baseline.txt landed 2026-10-18
~~23. `verify-all --baseline` convenience wrapper.~~ done — verify-all --baseline landed 2026-10-18
~~24. Ingest-telnyx -dry-run parity.~~ done — ingest-telnyx -dry-run landed
~~25. Baseline-freshness warning (v3 re-save as markup-change DoD).~~ done — baseline-freshness warning landed
~~26. Golden-scrubber stamp-format robustness check.~~ done — TestGoldenScrubberCoversFooterStamp landed
~~27. Print-probe `break-inside` computed-style assert.~~ done — print-probe break-inside assert landed
~~28. csp-sweep per-surface console attribution.~~ done — csp-sweep per-surface attribution landed
~~29. scripts/README duplicate codegen paragraphs merge.~~ done — scripts/README paragraphs merged
30. CHANGELOG Unreleased split at v0.3.0. **→ ROADMAP (CHANGELOG split at v0.3.0)**
31. Long-excerpt quarterly eyeball cadence. **→ ROADMAP (quarterly eyeball cadence)**
32. Purge-timer state on /ops (only if ops asks). **→ ROADMAP (purge-timer idea)**
~~33. Roadmap cadence: schedule the next Pareto pass BEFORE new work~~ done — cadence recorded; this pass is the next one
accretes (the pattern that closed both prior queues).
34-50. The remaining standing rows and watch items are already on
TODO_LIST (fold-ins at 2027-01, dependabot weekly confirm,
dependabot-core#16538, >1MiB policy, browser-304 manual check,
sibling backup fold, schema-v2 encoding, FTS ranking, stale-verdict
probe, recording timing asserts, tooling-triage remainder) — not
re-listed here; TODO_LIST is the backlog, this is the ledger.

## g) THREE QUESTIONS ONLY YOU CAN ANSWER

1. **The 3d93fdf lint-policy edits (2026-10-16): deliberate curation or a **→ TODO_LIST (lint-policy row)**
   foreign session's accident?** The commit removed `lll` from the global
   enable set (line-length checks are now conditional-only), relaxed
   golangci's `run.go` to "1.27" against the 1.27.2 module floor, and
   changed tagliatelle case rules. Three sessions have linted green with
   them; if they were deliberate, I will formalize them with a comment in
   `.golangci.yml`; if not, I revert and re-pin. I cannot know your
   intent — the answer decides a TODO row either way.
2. **dprint vs `docs/status/archived/` (and the annotation-heavy docs **→ ROADMAP (dprint policy)**
   generally): exclude the archive from formatting, or accept reflows?**
   Excluding ends the reflow-mangle class permanently (this pass needed
   repair-mangle after one dprint pass); accepting keeps one formatter
   for the whole tree. It is a one-line dprint.json change either way —
   but it encodes a taste call about whether archived prose is history
   (frozen) or text (maintained).
3. **May I file the two templ-components findings upstream now** **→ TODO_LIST (filings queue row)**
   (CopyButton singleton-script nonce gap; Card TitleTag h3 level-skip
   default), verify-before-filing discipline included? Both are
   discovered by our instruments, fresh, and small; but filings are
   public acts in YOUR repos, and the standing rule is that public
   presence moves on your say-so.

---

_Point-in-time snapshot 2026-10-18 00:55 CEST. Evidence: remote CI run
39076978166 success on the repaired tail; local 14/14 packages; check-rows
4/4 COMPLETE; dprint converged; AGENTS.md 219/219; live set = the 00-18
pass report + this self-review; the daemon sweeps the remaining doc edits.
Waiting for instructions._
