# Status: T2 receipt banked, release runbook rehearsed, and the lint-parity doctrine discovered

**Session window:** 2026-10-18 04:45–06:55 CEST · **Head:** `e8fea85` (pushed, CI run
39097483432 success) · **Tree:** clean · **Test command:** `GOEXPERIMENT=jsonv2 go
test ./...` green (untouched packages; no Go code changed this session)

This session executed the dispatched T2/T13 close-out: the literal clean-checkout
full-gate receipt, the cut-release.sh full-body rehearsal with every quiesce guard
proven to fire, and — as an unplanned centerpiece — the root cause of the repo's
gate-green/CI-red lint divergence (§e4 from the 04-25 report), solved with run
evidence and banked as a receipt doctrine. Everything is committed and remote CI
is green.

## a) FULLY DONE

~~1. **Session state re-established and the stale report commit shipped**: `36fd121`~~ done at `36fd121`
(the 04-25 status report, docs-only) was still unpushed at session start —
dprint-verified clean, pushed, CI green (run 39091624366). No drift found.
~~2. **§e4 SOLVED — the gate-green/CI-red lint divergence explained by run~~ done at `b4a9970`
evidence** (commit `b4a9970`, CI green): the 04-25 report's either/or ("dev
mode skips those linters or the gate ran a stale pass") was NEITHER. The
ca31221-era gate run `20261018-003727-d08c3565` shows `golangci-lint success
   1.3s` in its step table AND carries the exact CI trio (`gocyclo ×1,
   ineffassign ×1, modernize ×1`) in its findings section: **buildflow reports
golangci findings as non-fatal advisories, while CI exits non-zero on any
finding.** Verified NOT version skew (devShell and CI both v2.14.0) and NOT
invocation skew (no-cache dry-runs of dev AND full both plan the identical
`golangci-lint run --allow-parallel-runners ./...` against the repo config).
Banked as doctrine in docs/buildflow-dev-gate.md: **a receipt is
CI-equivalent only when the run's findings section shows ZERO golangci-lint
entries.**
~~3. **T2 — the literal full-gate receipt exists** (F5–F6): two FULL-mode gate runs~~ done — receipt banked 06:17 (57/0/0, load 14.41/35.00)
on the clean checkout (`b4a9970`, tree frozen, daemon quiet):

- Strict run: `20261018-041735-4709f40d`, **57 success / 0 failed / 0 skipped
  (+5 via config)**, exit 0, wall 1m0.8s, launched at load **14.41/35.00** —
  strictly inside the 1-min<15 AND 15-min<40 policy after an 80-minute
  60s-interval poll.
- First run: `20261018-041204-bc0e3b6d`, also 57/0/0 exit 0, wall 2m18.7s cold
  nix, at 1-min 17.24 (2.24 over threshold at launch — poll had verified
  12.94/39.50 seconds earlier; zero starvation casualties, so it stands as
  corroboration, not the receipt).
- test-race ✔ (27.7s), test-coverage ✔ (26.0s), nix-build ✔, nix-flake-check
  ✔, license-check ✔, and **go-generate/govalid green IN-GATE** — the literal
  step count is 57, not the 74 of the 2026-10-12 receipt (the pipeline
  compiles 118 tools → 85 DAG nodes; the executed set drifts with the fleet;
  banked per-run literally). Receipt paragraph in docs/buildflow-dev-gate.md.
  ~~4. **F7 — the v0.2.0 BuildFlow#56 workaround note closed**: the gate doc's~~ done — gate doc note rewritten to CLOSED
  "Gate tools trail module toolchain floors" bullet rewritten from
  "Until it lands: verify individually" to CLOSED (BF#56 closed 2026-10-17,
  binary `ef32fa0` reports go1.27.2, and the T2 receipt above ran govalid
  green in-gate). The lesson is kept (future floor bumps trail until the fleet
  binary rebuilds); the fallback doctrine is retired.
  ~~5. **T13 — cut-release.sh rehearsed end-to-end, all guards proven** (F48–F50,~~ done — rehearsed 06:24; receipts in scripts/README
  F52):
- **Body run 1 (CI-red rail)**: stubbed `gh` seeding
  `{"conclusion":"failure"}` → the full body ran green (quiesce ✔, CHANGELOG
  sweep ✔, nix build ✔, full dev gate ✔, all four probe suites ✔ with 42
  PNGs) and the remote-CI check REFUSED with the tag-red lesson message.
  Exit 1, no tag, no push. Stub in /tmp only — nothing in-repo seeded.
- **Body run 2 (abort-before-push)**: pre-created scratch tag `v0.0.0` → the
  entire body ran green with the REAL gh (the remote-CI check PASSED on
  `b4a9970`) → `fatal: tag 'v0.0.0' already exists` → FAIL **before** the
  push. Scratch tag deleted; `git ls-remote` confirms nothing leaked.
- **Negative guard proofs (cheap, pre-window)**: dirty tree (non-ignored
  scratch file), unpushed HEAD (`36fd121` vs origin), and over-policy load
  (fired live at 34.80/61.08). A discovered nuance is documented: gitignored
  names (`*.tmp`) are correctly invisible to the quiesce check — ignored
  files never enter flake sources.
- **F51 disposition**: the `go list -m` sleep-10 → retry loop stays
  UNIMPLEMENTED — the tag guard makes the proxy tail unreachable in a
  rehearsal, no real release has shown the lag, and the FAIL message already
  instructs the retry (YAGNI). Recorded in the scripts/README receipt (F52).
  ~~6. **Gate-surfaced advisories fixed on sight**: FEATURES.md dprint table padding~~ done — fixed on sight (5164305, e9d2749)
  (1-line whitespace fix); scripts/verify-ops-verdict.js oxfmt drift (daemon
  committed the fix as `5164305` — formatting-only, reviewed); the GENERATED
  `styles.css` excluded in `.buildflow.yml` next to `*.min.js` after
  prettier-format flagged it (a foreign-formatter rewrite would re-red the CSS
  gate — the 2026-10-15 incident class; scoped `buildflow -s prettier-format`
  re-run clean: "All matched files use Prettier code style!").
  ~~7. **Living docs rebuilt around the receipts**: CHANGELOG 2026-10-18 section~~ done — CHANGELOG 2026-10-18 + TODO_LIST (rebuilt 2026-10-23)
  gained the receipt + rehearsal bullets; TODO_LIST lost its last unblocked row
  and the rehearsals-residue row (Unblocked section now explicitly empty —
  every open row is owner/externally gated); the 04-25 report annotated inline
  (§b1, §b2, §f1, §f3, §f4 → done with pointers); live-set index updated.
  ~~8. **Final push verified**: three daemon-carried commits (`5164305`, `e9d2749`,~~ done — remote CI green on 8a04b4e (2026-10-23)
  `e8fea85`) + `b4a9970` pushed; `gh run watch 39097483432 --exit-status` →
  success.

## b) PARTIALLY DONE

~~1. **The §e4 deliverable is prose, not a literal matrix table**: the gate doc~~ **NOT-DO — prose carries the verdict + evidence; a 3x3 table adds no decision.**
carries the verdict + the invocation/version evidence + the zero-findings
receipt rule, but the "matrix" the 04-25 queue asked for (dev vs full vs CI
as a table) is folded into prose. What remains: a 3-row × 3-col table if the
literal artifact matters (S); blocker: none — cosmetic preference only.
2. **The runbook's proxy tail (`go list -m`) remains practice-unproven**: by **→ TODO_LIST (release proxy-tail watch row)**
design the rehearsal aborts at the tag guard, so the module-proxy check has
never executed against a real release. What works: the check's code path is
read-reviewed and its failure message instructs the retry. What remains: the
first real release exercises it; if proxy lag flakes, implement the bounded
retry then. Blocker: no release pending. Effort: S (watcher note only).
~~3. **Receipt-grade commit messages were lost to the daemon sweep** (see §d1):~~ **NOT-DO — content intact; the same-call commit norm was adopted 2026-10-23.**
the T2/T13 banking content landed in `e8fea85` ("chore: auto-commit 6 changed
file(s)") instead of the drafted detailed message. Content intact and CI
green; history readability degraded. What remains: a norm decision (§e1/§g-adjacent).
~~4. **§f queue items 5+ of the 04-25 report**: untouched this session (by~~ done — harvested into TODO_LIST 2026-10-23
dispatch — T2/T13 were the mandate). The carried queue is restated in §f
below with this session's additions.

## c) NOT STARTED

All owner- or window-gated; deliberately untouched this session. Each is
already a TODO_LIST row with evidence pointers — statuses here as of 06:55:

1. **Owner answer sheet §1–§11** (T1): staging doc ready, zero answers recorded. **→ docs/decisions/2026-10-17_owner-answer-sheet.md**
   Unblocks six High/Medium rows (date vocab, controls, export note, AI tone,
   mobile, CSP, coverage, baseline, BF#56 bookkeeping).
2. **Upstream filings** (T7–T10): 3 drafts staged + dupe-searched (CopyButton **→ TODO_LIST (filings queue row)**
   empty-nonce, Card h3 TitleTag, statix exit-0); the FILED step is an
   owner-go-ahead public act. This session adds a 4th candidate (§e5).
3. **T12 — lint-policy intent (daemon commit 3d93fdf)**: unexamined (owner **→ TODO_LIST (lint-policy row)**
   question); gates T5 (config-verify residue) and the T11 CI budget job.
4. **T24 — production re-ingest rehearsal**: needs the prod Master.csv path **→ TODO_LIST (production re-ingest row)**
   (owner); the `-dry-run` machinery is proven ready by its Telnyx twin.
5. **T15 — MOS poller execution**: design done; needs the 2027-01 Telnyx API **→ TODO_LIST (MOS poller row)**
   window.
6. **Host deploys + CSP enforce flip**: both bundle with the next host deploy **→ TODO_LIST (deploy + CSP rows)**
   (owner path; the `aiDids` module option is newly available).
7. **PII enforcement**: one-pager + recommendation ready; owner boundary call. **→ TODO_LIST (PII row)**
8. Everything else in the Low-impact TODO_LIST band (coverage chase, baseline **→ TODO_LIST (coverage, baseline, mobile, buffer, fold-in rows)**
   home, mobile cards, buffer policy, sibling fold-ins, dependabot watches,
   2027-01 live-verify, ContentHash v2 encoding) — no movement, all gated.

## d) TOTALLY FUCKED UP

Nothing broken — no test, build, or CI regression exists on `e8fea85`. The
honest list is process damage, all self-inflicted and corrected mid-session:

~~1. **The daemon ate two receipt-grade commit messages.** My drafted detailed~~ **NOT-DO — process note only; the same-call norm is in AGENTS.md + CONTRIBUTING.md.**
commit for the T2/T13 banking never landed: between `git add` and `git
   commit` the daemon swept the six files into `e8fea85` ("chore: auto-commit 6
changed file(s) (heuristic)"), and the on-sight fixes had already gone out as
`5164305`/`e9d2749`. Severity: history readability only — the content is
complete, pushed, and CI-green, and the receipts live in the docs, not just
the messages. Root cause: the daemon commits within seconds and my banking
batch spanned multiple tool calls. Mitigation this session: none (accepted
the daemon commits rather than rewriting pushed history). Permanent fix
proposal in §e1.
~~2. **My first dirty-guard "proof" was a false pass I nearly banked.** I seeded~~ **NOT-DO — process note only; the guard-message rule is in AGENTS.md.**
the quiesce check with `dirty-guard-proof.tmp` — but `*.tmp` is gitignored,
so the guard never saw it and the script failed on the NEXT guard instead.
I initially had two proof outputs that "looked right" (exit 1, FAIL line)
while proving the WRONG rail. Caught by reading which message fired; re-proven
with a non-ignored `.seed` name. Severity: none in the end (the final proof
set is sound), but the failure mode — a negative-path test that silently
tests the wrong thing — is exactly the class that produces fake receipts.
~~3. **A pipeline exit-code trap cost a verdict reversal.** I read `grep ... |~~ **NOT-DO — process note only.**
   head -5; echo $?` as "grep found golangci findings" (exit 0) in the T2
receipt run — but `$?` was `head`'s exit, not grep's. One `cat -v` re-check
showed the only matches were step-table rows: zero findings, receipt valid.
Two tool calls wasted; for a moment the session's headline receipt was
misread in the wrong direction (harmless direction, but still).
~~4. **The strict-window re-run cost ~7 minutes because of my own launch~~ **NOT-DO — process note only.**
overhead.** The poll opened at 12.94/39.50; by the time the gate's nix
develop started, 1-min load had crept to 17.24. The receipt discipline
forced a second run in a verified-strict window. Correct outcome, wasted
window: the poll loop should exec the gate itself on exit (§e2).
~~5. **Two wasted calls on stub-gh creation** (`printf %q` unsupported in the~~ **NOT-DO — process note only.**
shell harness; then the write tool's empty-file mtime rejection). Trivial,
listed only because §d demands honesty about every stumble.

## e) WHAT WE SHOULD IMPROVE

~~1. **Beat the daemon at receipt moments.** Pattern: any commit whose MESSAGE~~ done — adopted: AGENTS.md commit line + CONTRIBUTING.md push-to-push section
matters (receipts, reversions, doctrine) must be created in the SAME tool
call as the final edit — `git add … && git commit …` immediately, no
verification gap. Impact: today two detailed messages were lost; every
future receipt session hits this. Suggested fix: adopt the same-call rule
(record it in AGENTS.md's Git Workflow note or the lessons file), and when
the daemon wins anyway, `git commit --amend` on the UNPUSHED daemon commit
to restore the message (never after push).
2. **Make the load-poll launch the gate.** The 04-25 session and this one both **→ TODO_LIST (poll-exec wrapper row)**
paid a poll→launch gap that re-dirtied the load window. Fix: end the poll
loop with `exec nix develop -c env BUILDFLOW_NO_RESULT_CACHE=1 buildflow
   --build-mode full --max-time 15m` so the START LOAD the receipt records is
the load the loop itself verified. Effort: S; candidate for
scripts/ or the fleet's buildflow wrapper.
~~3. **Negative-path proofs must assert the guard's MESSAGE, not just its exit.**~~ done — guard-message rule adopted in AGENTS.md
Today's false-pass (d2) was visible only because I read the FAIL line.
Rule: a guard proof is valid only if the failure message names the seeded
cause. Fold into the same lessons note as §e1.
~~4. **Pipeline exit codes: never read `$?` after a pipe to `head`.** Use~~ **NOT-DO — cross-project lesson; belongs in crush-config, not this repo.**
`grep -c` (count), or capture `PIPESTATUS[0]`, or run grep alone. Cross-project
lesson — candidate for crush-config `references/lessons.md` (by commit there,
not in-session).
5. **New upstream-filing candidate: buildflow's advisory/fatal divergence.** **→ TODO_LIST (buildflow filing-decision row)**
The gate treats golangci findings as advisory while CI fails on them — the
exact gap that let ca31221 land gate-green/CI-red. A repo-side doctrine
exists now (zero-findings receipts), but the root fix is upstream: a
fail-on-findings knob or documented severity policy. Owner go-ahead needed
(filing doctrine); a draft would go through verify-before-filing like the
other four.
~~6. **Tooling-triage disposition text should track the doctor count.** The gate~~ done — gate doc refreshed 2026-10-23 (65 passed / 3 warnings / 30 failed)
doc's receipt says "29 tools" (2026-10-18 re-verify); today's runs report
**38 ok / 0 warn / 0 fail** — same disposition classes, drifted count. A
one-line refresh keeps the receipt honest (S).
7. **prettier-format runs via `nix run nixpkgs#prettier` fallback** (binary not **→ TODO_LIST (prettier devShell row)**
in the devShell) — 2.7s per run and a warning line every gate. Adding
prettier to the devShell would silence it and pin the version. S, fleet-config
adjacency.
~~8. **The oxfmt advisory is order-flaky**: the full gate flagged~~ **NOT-DO — did not recur through 2026-10-23; add a gate-doc line only if it does.**
verify-ops-verdict.js unformatted; the scoped re-run (and the tree) were
clean, and the backgrounded `--fix` no-op'd. One gate run, one contradicting
scoped run = an advisory class worth a line in the gate doc if it recurs.

## f) Top things we should get done next (impact-ordered; routing noted)

Items 1–3 are new or re-scoped by this session; 4+ carry the 04-25 queue
forward. HARVEST routing: ★ = already a TODO_LIST row (keep), ◆ = new (needs a
row or ROADMAP), ✚ = owner-gated, no row change.

1. ◆✚ **Decide the buildflow advisory-divergence filing** (§e5): draft a **→ TODO_LIST (buildflow filing-decision row)**
   fail-on-findings knob request upstream or hold with the repo doctrine.
   Critical for lint trust. S. (Blocked on §g1.)
   ~~2. ◆ **Adopt the same-call commit rule + amend-on-unpushed-daemon-commit norm**~~ done — adopted 2026-10-23 (AGENTS.md commit line + CONTRIBUTING.md)
   (§e1) and the guard-message assertion rule (§e3); write them into AGENTS.md
   Git Workflow (2–3 lines, budget is 218/219 — may need a swap, not an add).
   High. S.
2. ◆ **Poll-exec wrapper for load-gated gate runs** (§e2): the poll loop execs **→ TODO_LIST (poll-exec wrapper row)**
   the strict-window gate, printing START LOAD from its own last check. High.
   S.
3. ✚ **Owner answers §1–§11** (T1): unblocks the six High/Medium rows. **→ docs/decisions/2026-10-17_owner-answer-sheet.md**
   Critical. 30min.
4. ✚ **FILE the four staged drafts** (CopyButton nonce → templ-components, **→ TODO_LIST (filings queue row)**
   Card h3 → templ-components, statix exit-0 → molybdenumsoftware/statix;
   optional testkit comment per the retraction scope). High. 30min.
5. ✚ **T12 lint-policy intent (3d93fdf)**: confirm or revert; gates T5 + T11. **→ TODO_LIST (lint-policy row)**
   High. 15min.
6. ★✚ **T24 prod re-ingest rehearsal**: needs the Master.csv path; then the **→ TODO_LIST (production re-ingest row)**
   one-time ContentHash v2 re-ingest. Critical (prod honesty). 30min + 60min.
7. ✚ **Stamp-churn doctrine call** (carried §g3): client-side footer stamp / **→ TODO_LIST (stamp-churn row)**
   RelativeTime chips would make the 304 economy real (today ~every poll
   re-downloads). High, product-level. S decision + M implementation.
8. ★✚ **Host deploys with current module options** (`aiDids` new): then bundle **→ TODO_LIST (deploy row)**
   the CSP enforce flip. High. 30min.
9. ★ **T5 residue once T12 lands**: wire `golangci-lint config verify` on the **→ TODO_LIST (gate-hardening residue row)**
   chosen surface; prove it rejects the historical gohumanize config. Medium.
   30min.
10. ★ **T11 residue once T12 lands**: CI AGENTS-budget job + 220-line seed **→ TODO_LIST (hooks residue row)**
    proof. Low. 15min.
    ~~12. ◆ **Refresh the tooling-triage count** (29 → 38) in the gate doc. Low. S.~~ done — gate doc refreshed 2026-10-23
11. ◆ **Watch the first real release's `go list -m` tail**; implement the **→ TODO_LIST (release proxy-tail watch row)**
    bounded retry only if proxy lag actually flakes (F51 watcher). Low. S.
12. ★✚ **PII enforcement boundary** (read-side recommended): owner decision **→ TODO_LIST (PII row)**
    then M implementation. Medium.
13. ★✚ **Coverage floor**: chase toward 80% or accept 70 (measured 73.0%) — **→ TODO_LIST (coverage row)**
    owner call. Low. 60min.
14. ★✚ **Pixel-baseline durable home** (v2 corpus diffs clean at **→ TODO_LIST (pixel-baseline row)**
    ~/lens-baselines/v2): owner picks machine-local vs committed. Low. 15min.
15. ★ **Fold-ins on upstream land**: go-appkit#26 (ready), go-etag#2 (open), **→ TODO_LIST (fold-ins row)**
    templ-components #48/#49/#50/#51/#55 (bump watchlist runs the re-checks).
    Low. 30min each cycle.
16. ★✚ **Mobile card-list for the calls table**: owner taste call first **→ TODO_LIST (mobile row)**
    (options staged in the answer sheet). Low. L.
17. ★✚ **>1MiB marathon transcripts**: buffer policy decision (accept **→ TODO_LIST (>1MiB policy row)**
    untagged overflow logging vs raise MaxBufferSize). Low. 15min.
18. ★ **Confirm dependabot proposes the newly-direct deps** (httputil, **→ TODO_LIST (dependabot row)**
    server_timing, go-etag/server) on the next weekly cycle. Low. 10min.
19. ★✚ **dependabot-core#16538** watch + maintainer questions; then the gomod **→ TODO_LIST (#16538 row)**
    pinning call. Low. 10min.
20. ★✚ **Sibling repo fold**: backup-rotation opinion into lens.nix timer **→ TODO_LIST (sibling backup row)**
    descriptions. Low. 15min.
21. ★ **2027-01 Telnyx window**: run docs/telnyx-quarterly-checklist.md top to **→ TODO_LIST (2027-01 window row)**
    bottom (Detail Records DTOs, MP3 Range); calendar confirm is step one.
    Low. 30min.
22. ★ **ContentHash v2 length-prefixed encoding**: scheduled at schema v2; **→ TODO_LIST (schema-v2 row)**
    do not pull forward. Low. 60min.
    ~~25. ◆ **If §d1 recurs: amend norm for unpushed daemon commits** — fold into~~ done — folded into item 2 (norm adopted 2026-10-23)
    item 2 rather than a separate row. Low. S.
23. ◆ **prettier into devShell** (§e7) to kill the nix-fallback warning. **→ TODO_LIST (prettier devShell row)**
    Low. S. (fleet-config adjacency — may belong in crush-config, not here.)
    ~~27. ◆ **Gate-doc line for the oxfmt transient advisory** (§e8) only if it~~ **NOT-DO — no recurrence through 2026-10-23.**
    recurs. Low. S.

(Stop at 27 — everything else in the 04-25 §f 50-item tail is either done,
done-adjacent via this session's receipts, or duplicated by the TODO_LIST rows
above. Padding to 50 would invent work.)

## g) Top 3 questions I can NOT figure out myself

1. **Filings — file or hold, and does the buildflow divergence join the queue?** **→ TODO_LIST (filings queue row)**
   Four candidates now: the three staged drafts (CopyButton empty-nonce, Card
   h3 TitleTag, statix exit-0-on-config-error) plus this session's discovery
   (§e5: buildflow classifies golangci findings as advisory while CI fails on
   them — request a fail-on-findings knob or a documented severity policy?).
   I cannot decide what goes public in your name. A yes/no per draft is enough;
   the drafts and dupe-searches are ready.
2. **The stamp-churn doctrine call (re-asked from 04-25 §g3, now with receipt **→ TODO_LIST (stamp-churn row)**
   context):** with 1-second body granularity, the dashboard's 30s htmx polls
   almost never hit their 304s — the ETag changes every second, so the whole
   conditional-GET economy (the v0.2.0 centerpiece) is largely theoretical
   today. Rendering the "Updated HH:MM:SS" stamp and RelativeTime chips
   client-side would make bodies second-stable and 304s real. This is a product
   tradeoff (JS on the critical path, CSP nonce work) I should not decide
   unilaterally. Build it, or accept the churn?
3. **3d93fdf lint-policy intent (re-asked; still gating two rows):** the daemon **→ TODO_LIST (lint-policy row)**
   commit touched `lll` exclusion scope, `run.go` "1.27" vs the 1.27.2 floor,
   and tagliatelle rule changes. Deliberate policy (I'll comment the why
   in-file and wire the gated residues), or accidental (I'll revert and pin the
   policy note)? T5 (config-verify surface) and T11 (CI budget job) cannot
   start without this answer.

---

_Point-in-time snapshot per the status-report skill. §f is the HARVEST input;
the ★ items already live in TODO_LIST.md, the ◆ items are new (items 2–3 are
small enough to fold into AGENTS.md/scripts directly on a go-ahead), and the
✚ gates are unchanged. Await instructions._
