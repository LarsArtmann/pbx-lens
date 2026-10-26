<!-- Status report 2026-10-26 16:00 — session 5 (continued from the
     2026-10-25 19-09 report). Lane: fix the CI red, wire the gate, earn
     the FULL receipt, execute the owner-answered release. Concurrent
     lane: the 15-09 full-code-review session (its CDR/Telnyx/vendorHash
     fixes share this history and are credited where they land).
     §a ten receipts, §d four fuckups, §f 32 ranked, §g three owner
     questions. -->

# v0.3.0 shipped — and three gates earned their red (2026-10-25 18:25 → 2026-10-26 16:00)

**Input:** the 19-09 report's next-list (CI red ×3, gate unwired, receipt
owed, owner gates open). **Outcome:** the red fixed and its gate moved
left, a second gate bug found by its own receipt and fixed with a seeded
red, the daemon norm owner-ratified, the FULL receipt landed honest, and
**v0.3.0 is tagged, pushed, and served by the module proxy**.

## a) Fully done

1. **The CI red is fixed at root** (`afae51e`): `Query`/`QueryTurns` lost
   their duplicated prologue to `normalizedQuery`; `firstMatchedTurn` now
   scans via `slices.IndexFunc` over the shared match views. Verified with
   the CI pin (93432fd): pristine tree reproduces CI's exact group, fixed
   tree reports 0. CI run 40221643859 green.
2. **The duplication gate moved left** (`315587e`): new
   `scripts/check-dedup-gate.sh` runs CI's exact command locally, REFUSES
   a non-pin binary, self-tests red (seeded 12-statement clone) and green
   (clean scratch tree); wired into `.githooks/pre-commit` (every commit,
   daemon included) and CI's Guard Self-Tests job ("Four gates" became
   five); scripts/README.md row; pin-parity gotcha recorded in
   docs/art-dupl-baseline.md. CI run 40222247523 green, all eight jobs.
3. **The stale-binary trap, documented the hard way**: the devShell's
   `art-dupl` resolved to a stale user-local dev build that missed a real
   10-line clone AND flagged a 1-token comment-vs-struct "clone" the pin
   does not see — both directions wrong. `~/go/bin/art-dupl` is now the
   pin; the owning doc carries the incident.
4. **Browser lane green**: `verify-all.sh` passed all four probe suites
   (42 PNGs) — standalone before the receipt and again inside the release
   runbook.
5. **poll-gate armed over-policy; fixed with its production fixture as
   the seeded red** (`5db1853`): `ready()` compared the whole
   "1m 5m 15m" triple as one STRING (lexicographic "39.25 …" < "40" is
   true), so a quiet 1m laundered a hot 15m — production proof: armed at
   15m=65.13 under a 15m<40 policy, receipt triple exposed it. Only the
   15m field arms now; the self-test carries the exact case. CI green.
6. **The daemon norm is owner-ratified** (`f57cebe`, gate §g3 = squash):
   docs/concurrency.md leaves DRAFT — the fetch-first/soft-reset/recommit
   repair is doctrine, push-immediately shrinks the window, pushed daemon
   commits stay; rule 2 gains the void-over-policy-receipt case, rule 5
   the stale-tool-binary audit. Executed live three times this session
   (afae51e, 315587e, 0df9fe6), each pushed within a minute.
7. **The BuildFlow#55 corruption was repaired** (`0df9fe6`, mine):
   the failed run's forbidden `nix-hash-fix` blanked the pinned cqrs-lint
   source hash to all-A's and the daemon committed it; restored the
   original hash for the unchanged rev, `nix build` verified. The
   companion corruption (cqrs-lint's own vendorHash overwritten by the
   sweep) was fixed by the concurrent session (`7ff3173`); both verified
   by building the root package AND the devShell.
8. **The third FULL receipt is honest and covers the features**
   (`1b5208a`): verified triple 1m=48.05 5m=40.35 15m=29.02 (policy < 40),
   tree frozen at `3cdc5b2` (START-HEAD == END-HEAD), 60 success / 0
   failed, exit 0, run `20261026-101959-8b3301e0`, wall 2m15s. The two
   void runs and their lessons are recorded beside it in
   docs/buildflow-dev-gate.md.
9. **P3 EXECUTED — v0.3.0 is out** (gate §g2 = v0.3.0, answered via the
   question tool): CHANGELOG Unreleased renamed (`3e97485`), then
   `cut-release.sh v0.3.0` cleared quiesce, CHANGELOG sweep, nix build,
   the dev gate (48/66 advisory), all four browser suites, remote-CI-on-
   HEAD, tag + push, and the module-proxy check (`go list -m` serves
   v0.3.0). Plan header updated with the receipts (`891d3f6`).
10. **§g1 retired without asking**: PR #3 is already CLOSED and modernc
    sqlite v1.61.0 sits in go.mod with every gate green — the close-plus-
    hand-roll option executed itself before the question was worth asking.

## b) Partially done

1. **The GitHub Release object**: the notes are written, complete, and
   committed at docs/drafts/2026-10-26_v0.3.0-release-notes-shipped.md —
   but `gh release create` died with 401: the gh token went invalid
   mid-session (run watches worked at 14:09; the token is now dead per
   `gh auth status`). Needs `gh auth login`, then one command (in the
   file's header comment).
2. **CHANGELOG vs release notes for the 10-26 gate work**: the CDR and
   Telnyx fixes carried their own CHANGELOG entries; the poll-gate and
   dedup-gate fixes live in the release notes and docs but were not given
   their own CHANGELOG rows — confirm or append.
3. **The 19-09 report's §f 1–5 are now done** but not yet annotated
   inline in that report (only the plan header carries the state).
4. **P15/P16 remain session-4 partials** (pixel baselines rode P1; ETag
   hook counters + retention SLO fields untouched).

## c) Not started

1. **P12** — backlog harvest (15-39/16-58/17-12 §f → TODO_LIST/ROADMAP +
   inline annotations).
2. **P11** — answer-rate analytics (billsec-0 × hangup-cause).
3. **P13** — sibling folds (CDR call_direction template, lens.nix backup
   wording, webphone deep link; check the sibling window first).
4. **P14** — lint residue (gofumpt/v2.14 parity, unknown-`//nolint` grep).
5. **P16 remainder** — ETag `OnETagGenerated`/`On304` counters and
   retention SLO fields on `/api/ops`.
6. **P17** — two decision notes.
7. **P18** — drills (FTS tokenizer table test; VACUUM INTO restore).
8. **P19** — upstream watchlists.
9. **P4 remainder** — dependabot github-actions-only grouping.
10. Note: the concurrent 15-09 session is executing a 7-lane full-code-
    review plan that overlaps P14 territory — coordinate before starting.

## d) Totally fucked up

1. **I trusted the wrong binary for an hour.** The devShell's stale
   art-dupl produced a bogus clone pair and I did a wasted extraction
   round (positional literal, extra comment) chasing its verdict before
   installing the CI pin — which showed the extraction was already
   sufficient. The IndexFunc rewrite survived as a genuine improvement,
   but the detour was mine: gate work should begin with
   `art-dupl version | grep 93432fd`, not end with it.
2. **My repair verification had a blind spot.** After the first hash
   corruption I "verified" with `nix build` — the root package, which
   never touches the cqrs-lint FOD. The second corruption survived my
   green check and killed the next FULL run. Root + devShell (all nix
   targets) is the minimum verification for any hash touch.
3. **Two FULL runs died before the receipt landed.** The first armed
   against its own policy (the string-compare bug — the receipt triple
   exposed it; I nearly accepted interleaved output as an honest arm).
   The second died on the doubled hash corruption. ~2 hours of gate wall
   time is the bill; both deaths produced real fixes, which is the only
   honest consolation.
4. **The release body was drafted to /tmp** and the tmp reaper ate it
   overnight; the `cat >>` then silently recreated a stub, caught only at
   copy time. Durable artifacts go in the repo, full stop.

## e) What we should improve

1. **FOD-adjacent repairs get verified against every nix target** (root
   package + devShell + checks) — one command each, and the cqrs-lint
   class of failure can never hide again.
2. **Gate binaries at the CI pin are step zero** of any gate work; a
   version check belongs in the ritual, and in the script (done for
   dedup; poll-gate needs no pin but its self-test now discriminates).
3. **The receipt triple is read against the policy before accepting an
   arm** — the exit code proves nothing; the printed, verified triple is
   the evidence.
4. **Self-test fixtures must discriminate the actual bug** — poll-gate's
   old fixtures passed under the string-compare bug; a seeded case that
   cannot fail under the bug is not a test of the bug.
5. **Durable artifacts live in the repo from birth** (docs/drafts/), never
   /tmp, for anything that must outlive the hour.
6. **cut-release.sh should reuse poll-gate.sh** for its load wait (its
   inline awk reads fields separately and is safe today, but the hand-
   rolled retry loop I used is the smell).
7. **AGENTS.md at 219/219 is full**: the art-dupl PATH-pin gotcha deserves
   an AGENTS line and currently lives only in the owning docs — needs a
   prune-first commit.
8. **The daemon + concurrent sessions made receipts expensive** (three
   attempts): rule 2 documents voidness; a session-scoped tree claim
   during receipt runs would prevent it.

## f) Next (ranked)

1. Owner: `gh auth login`, then `gh release create v0.3.0 --title "v0.3.0
   — the analytics surface, transcript truth, and gate honesty"
   --notes-file docs/drafts/2026-10-26_v0.3.0-release-notes-shipped.md`
   (strip the comment header first).
2. P12: harvest the three reports' §f lists into TODO_LIST/ROADMAP.
3. Annotate 19-09 §f1–5 done with receipts (afae51e, 315587e, lane green,
   third receipt, 5cad5fc).
4. Decide + document the rapid `.fail` seed policy (keep-in-git as
   regression pins vs gitignore; the daemon will sweep future seeds
   either way).
5. Confirm/append CHANGELOG rows for the 10-26 gate fixes (poll-gate,
   dedup gate) inside the v0.3.0 section.
6. Owner decision: exclude `nix-hash-fix` from buildflow runs (145/180
   failures, BuildFlow#55 corrupted production hashes twice this week).
7. Wire check-dedup-gate.sh into verify-all.sh's head (the one wiring
   place deliberately skipped — decide and do).
8. P4 remainder: dependabot github-actions-only grouping.
9. P11: answer-rate (billsec-0 × hangup-cause definition, honest
   denominators row, QualityRollup extension, dashboard section).
10. P16: ETag hook counters (pinned v0.6.1 hooks) into /api/ops.
11. P16: retention SLO fields (oldest unretired view, purge lag) on
    /api/ops.
12. P13: sibling folds (check the sibling window first).
13. P14: gofumpt/v2.14 parity + unknown-`//nolint` grep.
14. P17: two decision notes.
15. P18: FTS tokenizer table test.
16. P18: VACUUM INTO restore drill re-run on the current schema.
17. P19: upstream watchlists refresh (go-cqrs-lite v5 budget; versions.json
    lies — verify with `git tag --contains`).
18. AGENTS.md prune-first commit, then add the art-dupl PATH-pin gotcha.
19. Rebuild the local BuildFlow binary (built 58115f7, repo at 24dff81 —
    the freshness warning fires every run).
20. `sqlite3 ~/.cache/buildflow/cache.db VACUUM` (0.18 GB, 87% free
    pages; the state DB at 0.63 GB deserves a look too).
21. Tag-triggered CI: add `v*` tags trigger to ci.yml (the release tag
    itself never ran CI; main did).
22. go-licenses: "no license found" for the larsartmann modules
    (go-appkit, go-cqrs-lite modules) — add LICENSEs upstream or suppress,
    17 error lines per FULL run.
23. file-size residue: split server.go (715), dashboard.go (551),
    search.go (570), telnyx_test.go (836) by domain.
24. jsonv2 watchlist: the 39 jsonv1-boundary findings are the inventory
    for the eventual GOEXPERIMENT flip — order them by risk.
25. branching-flow triage: projection.go (18 fields), dashboard.go
    (24 fields) — shrink or document.
26. cqrs-lint policy pass: 19 findings vs docs/cqrs-lint-policy.md (the
    CBOR time.Time timezone warning is real and new).
27. govulncheck's 13 stdlib findings: map to the toolchain-bump window
    (quarterly re-verify cadence, next window 2027-01).
28. vulnix's 22 advisories: pinned vips/bash/binutils set — nixpkgs bump
    window.
29. TODO_LIST refresh against what v0.3.0 shipped (31 open rows at plan
    input).
30. ROADMAP theme 0 update: v0.3.0 is out; what is the next north-star
    slice?
31. docs/http-stack.md bench receipts: re-run both benchmarks on a quiet
    box (today's regression verdicts were load variance).
32. Harvest the concurrent session's 15-09 §f 50-ranked after it closes —
    merge with this list, dedupe, and feed the next Pareto plan.

## g) Owner questions (three, not self-answerable)

1. **gh token**: refresh it (`gh auth login`) so the v0.3.0 GitHub
   Release notes can be published from the prepared drafts file — or is
   notes-in-drafts the intended final state for this release?
2. **nix-hash-fix**: it has now corrupted real hashes twice (BuildFlow#55,
   production, this week). Exclude it from buildflow runs repo-wide
   (`--exclude`/skip_steps), or keep it armed under the
   verify-every-FOD discipline?
3. **Lane split**: the concurrent 15-09 session overlaps the plan tail
   (P12 harvest, P14 lint residue). Should this lane take
   P11/P13/P16–P19 and leave P12/P14 to it — or does one lane drain the
   whole tail while the other stands down?
