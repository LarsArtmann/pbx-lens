<!-- docs/status/2026-11-20_22-20_pareto-execution-wave-status.md — the
     GET-SHIT-DONE execution session's own a–g ledger. Point-in-time:
     TODO_LIST.md stays the living source. -->

# Status — Pareto execution wave (2026-11-20, 22:20 CET)

**Session scope:** execution of
`docs/planning/2026-11-20_10-44_SUPERB-pareto-execution-plan.md` (27
medium tasks T01–T27 + 76 fine tasks F01–F76) end-to-end: publish →
release v0.4.0 → dependency waves → the P1 structural moves → the
hygiene tail. 44 commits pushed on top of the mega-review's `280a11d`;
current HEAD `d81c09d` is **CI-green on all 9 lanes** (run 44803432699),
tree clean, everything pushed.

**Gates at session end (all green):** build ✓ · `go test ./... -count=1`
16/16 ✓ · golangci-lint 0 issues ✓ · cqrs-lint exit 0 ✓ · dedup gate
0 clone groups ✓ · docs counts + links ✓ · AGENTS budget 219/219 ✓ ·
`nix build` ✓ (vendorHash verified twice after bump waves) · CI on HEAD
✓ · v0.4.0 tag CI ✓.

---

## a) FULLY DONE (verified, receipts below)

1. **v0.4.0 RELEASED.** Version const 0.4.0, CHANGELOG cut
   `[v0.4.0] — 2026-11-20`, `scripts/cut-release.sh v0.4.0` exit 0
   (nix build + full dev gate + all four browser probe suites inside the
   runbook), annotated tag pushed, module proxy serves
   `github.com/larsartmann/pbx-lens v0.4.0` (F12 receipt: `go list -m`),
   GitHub Release published with the full CHANGELOG section, tag CI
   success.
2. **T01 publish + reconcile:** the two red lanes on the mega-review
   push fixed — gitleaks installs via its DECLARED module path
   `github.com/zricethezav/gitleaks/v8` (the gitleaks-org path fails
   with a version-constraints conflict, proven on run 44713670130), the
   viewkey nilerr got its ledger row (F37). All 9 lanes green on
   `280a11d`.
3. **T03 the receipt-grade gates:** BuildFlow SOLO dev gate exit 0
   (poll-gate armed at 15m=26.48), full `go test -race ./...` green
   after fixing `assertConditionalHEAD` — its comment promised a
   ride-out retry for the minute-truncated stamp tick but the loop
   re-sent the stale tag; a miss now re-mints from a fresh GET.
4. **T05 templ-components v1.27.2 retirement act** (target upgraded
   from the plan's v1.27.1 — v1.27.2 verified by `git tag --contains`
   style commit-ancestry checks to carry #48/#51/#63 AND the Toast
   duplicate-`</script>` fix AND the JSON-LD escape): `!items-stretch`×3
   deleted, app.css row-focus CSS + `.tc-tag-*` mirror deleted,
   `transcriptMatchLog` rewritten onto the library Scrollback (matched
   turns only keep anchors — the goldens pin unanchored misses),
   `pbx-lens-nav` id pin retired (content-hashed menu id;
   `TestNavIDIsDeterministic` now asserts render-to-render equality),
   theme/custom CSS layers synced to the v1.27.2 starters, the NEW
   accent token axis (ADR-0044) bridged to the teal signal palette,
   class corpus + 9 goldens re-baselined, verify-all all-pass after the
   keyboard-walk probe moved to the v1.27.2 row contract
   (`data-tc-row-href`; the library dropped invalid `role="link"`).
5. **T06 cqrs wave:** system v4.19.0, metaengine v4.26.0,
   metaengine/sqliteengine v4.9.1, catalog v4.15.0 (sqlite held at
   1.61.1); full suite green = the used-surface contract; vendorHash
   re-pinned.
6. **T07 vulnerability posture:** go-jose v4.1.4 → v4.1.6 clears all
   three Dependabot alerts (CVE-2026-116585/-116586/-116587; indirect
   via go-oidc, minimum-version pinned in our graph);
   GO-2026-6859 (net/http MaxBytesReader race, fixed in go1.27.4)
   recorded as **HOLD** in
   `docs/decisions/2026-11-20_go-2026-6859-posture.md` — nixpkgs still
   ships 1.27.3, the flake override rejected on hermeticity doctrine,
   CI's setup-go lane already runs 1.27.4, re-check is one eval
   (`nix eval nixpkgs#go_1_27.version`).
7. **T08 honest login page:** the register section follows the operator
   seat (boot-time `ReadModel().Count() >= MaxUsers` — the drain is
   synchronous, proven by the reopen leg of
   `TestRegistrationSectionFollowsTheSeat`), favicon accent teal,
   ThemeFromRequest deliberately NOT adopted (the app themes via the
   library's localStorage toggle, not a cookie — recorded in the
   CHANGELOG).
8. **T09 PC-1:** `CallView` now embeds `Observation` + `Enrichment`
   whole; `ViewFrom` stays as the one canonical fold constructor; the
   hand-mirrored `Observation()`/`enrichmentOf` deleted;
   `TestCallViewJSONIdentityAcrossTheEmbed` pins the 18-key set and the
   hash invariant (the enrich pass addresses commands through the
   embedded snapshot).
9. **T10 the web→ingest edge is dead:** the recording proxy speaks
   `calls.RecordingMedia` through its `RecordingFetcher`;
   `telnyx.Client.FetchRecordingMedia` adapts the wire DTO;
   `TestWebProdFilesNeverImportIngest` (go/parser guard) fails the
   build if the edge reopens.
10. **T16 identity extraction (P1 move one):** `userauth.go` + 5 test
    files → `internal/identity`; the ceremony walkers shared through
    `internal/identity/identitytest` (the dedup gate caught the
    duplication — one copy now serves both suites, thin delegates keep
    call sites short); `Options.Identity *identity.Identity`;
    IdentityOps rides with identity; docs cross-refs updated
    (DOMAIN_LANGUAGE, AGENTS, http-stack, cqrs-lint-policy, TODO_LIST);
    the nav logout-form markup assertions moved to the web package where
    the render lives (`TestNavRendersTheLogoutFormWhenAuthed`).
11. **T19 deepening #1 executed:** `cli.PollerHealth` — the typed
    pass-health snapshot (counters + honest age/stale verdict, age −1 =
    never ran) crosses the ops seam; `SnapshotAny` deleted; the serve
    fixture and the ops-page fixtures build the typed value; /api/ops
    source keys now uniformly lowercase (SourceStat carries json tags);
    glossary term added; the serve verb's 22 flags moved to a typed
    `serveFlags` set (funlen seam + T18 prep).
12. **T12 infra solos:** httputil v1.9.0 + go-appkit/systemd v0.2.0,
    ETag/no-cache middleware test lanes green; server_timing already at
    latest (v1.0.3).
13. **T13 scanner pins:** CI govulncheck pinned to v1.9.0 (the
    devShell's pin — @latest was the only unpinned CI tool), the
    nix-installer `github_token` → `github-token` annotation fixed;
    local gitleaks v8.30.1 scan of the tree: **no leaks** (nothing to
    baseline).
14. **T14 hygiene:** coverage row carries the honestly measured **71.2%**
    (was 73.0 — the review wave's new paths + identitytest/oidctest
    helper packages pulled it down; floor 70 still met), ROADMAP
    harvested (PII layer, analytics consumers, owner-gated ASR variant),
    executed rows flipped (release/FM#11/deepening), naming same-day
    note in the CHANGELOG.
15. **T15 report QA:** `scripts/render-report.js` landed (NixOS chromium
    via the probe's discovery; the exit-127 chrome-headless-shell trap
    documented in its header — no libnss3); all six 2026-11-20 reports
    rendered; vision pass (band-sliced, measured) returned
    ALL-SIX-PASS with one real defect — the naming-review's wide tables
    overflowed the margin and squeezed the last column to ~48–70px —
    fixed with `overflow-wrap: anywhere` + a last-column `min-width`,
    re-rendered and re-verified by direct inspection. PNGs stay
    machine-local (regenerable; `docs/{reviews,architecture-understanding}/*.png`
    ignored, `docs/assets` screenshots still tracked).
16. **T22 filings staged:** four drafts with module-graph evidence
    chains (`docs/drafts/2026-11-20_dependency-hygiene-filings.md`):
    fsnotify (a-h/templ), pkg/errors (watermill), charm-v1→charm.land v2
    (go-appkit — OWNER ACTION), erraudit enforce-* refresh; ledger
    appended.
17. **T24 identity backup:** runbook section (ExportUsersToJSON as the
    logical export; file-level backup stays the recovery path; the CLI
    verb recorded as follow-up) + the 24h session-TTL line re-affirmed.
18. **T26 decision pack:**
    `docs/decisions/2026-11-20_owner-decision-pack.md` — four calls with
    option sets and → ANSWER slots (PII boundary, CSP enforce flip,
    catalog push, divergence dossier).
19. **T27 watch lane:** five scheduled items recorded in TODO_LIST
    (cqrs v5 census, fleet index triggers, 2027-01 Telnyx window,
    legacy-keyed rebuild slot, ops OIDC-error surfacing).
20. **CHANGELOG batch two** under `[Unreleased]` (identity boundary,
    honest login, typed seam, PC-1 compose, edge kill, scanner pins,
    posture) — nothing invisible to the next release.

## b) PARTIALLY DONE

1. **T17 query split** — decision ratified + full inventory measured
   (queries.go 712, dashboard.go 563, quality/segments/anomaly/coverage/
   freshness ≈ 2.7k LOC + tests). The move itself NOT executed: it needs
   interface surgery (parseRange and friends are HTTP-layer; the honest
   split exports view models and leaves request parsing in web) — a
   fresh-session opener, not a tail task.
2. **T18 serve composition move** — `serveFlags` extraction done (the
   seam exists), the composition itself (serve.go 590 + ops.go 284 out
   of cmd) not moved.
3. **T20** — PC-3 closed as **no-op by design** (upstream `id.StreamID`
   already IS the validated branded type; `RecordKey`/`LegacyRecordKey`/
   `LookupViewKey` are the canonical seams — a parallel `calls.StreamKey`
   would duplicate, not strengthen). PC-2 (closed lifecycle union over
   the decider's parallel booleans) deferred deliberately: a decider
   rewrite mid-session violates the move-only guardrail spirit.
4. **T21 ScanPage prototype** — deferred to an exploration session
   (spike-branch work, not maintenance).
5. **T23 bump riders** — templCliPackage (go-nix-helpers bump) and
   art-dupl suppressed-bucket surface (its pin bump) recorded as WATCH:
   the triggers are external module releases.
6. **T25 coverage pilot** — measured 71.2% ≥ floor 70; the pilot's
   marginal value is the owner's chase call; no unprompted test-volume
   growth.
7. **T26** — pack written, the four ANSWER slots empty (owner).
8. **Vision QA tooling** — glm-4.6v dispatch failed (max_tokens param
   illegal), synthetic provider exhausted quota mid-verification; the
   final round was verified by direct image inspection (view tool on
   cropped bands) instead of a model. Works, but the model chain needs
   a healthy fallback.

## c) NOT STARTED

1. Actually FILING the four upstream filings (drafts staged only; each
   needs parent-main re-verification at file time).
2. T17/T18 execution (see b).
3. PC-2 implementation.
4. ScanPage spike.
5. `pbx-lens identity-export` CLI verb (runbook-only today).
6. The 2027-01 Telnyx re-verify window (recordings MP3-only,
   enrichment spec-verified) + the MOS poller execution slot.
7. Legacy-keyed rebuild EXECUTION (runbook ready since 2026-11-04;
   needs a deploy window).
8. CSP enforce flip, PII boundary implementation, catalog push,
   divergence dossier — all gated on the decision pack's answers.
9. TLS reverse proxy for the lens (identity posture doctrine says
   TLS-first; no host config exists yet — the passkey ceremony cannot
   run over LAN http today).
10. Fleet index refresh after today's four bump waves (WATCH trigger).

## d) TOTALLY FUCKED UP (honest)

1. **Three consecutive red CI runs (19:02–19:08)** on the
   `Check go mod tidy` lane — I pushed the bump waves and hygiene batch
   without running CI's EXACT tidy check locally (my gates: build, test,
   lint, cqrs-lint, dedup, docs — tidy-diff was missing). The red→green
   flip by `d81c09d` (no targeted go.sum change in between) is NOT fully
   explained; watch for recurrence. This is the session's biggest miss.
2. **A stray build binary** (`cmd/pbx-lens/pbx-lens`) was swept by the
   daemon into the identity-move commit and PUSHED (caught on the next
   status check, removed, ignore rule added).
3. **Six 1–2MB report PNGs** were daemon-swept and pushed (~8MB repo
   weight), then removed; my first `.gitignore` addition was TOO BROAD
   (`docs/**/*.png` would have silently ignored future legit
   `docs/assets` screenshots) — caught by a tracked-files audit and
   narrowed in a follow-up commit.
4. **The release cost three attempts.** Attempt 1 died on the dedup gate
   (bare-bash probe lane resolved a host `art-dupl version dev`);
   attempt 2 died the SAME death because my "fix" routed the lane
   through the devShell where the dev binary shadows the pin — the
   RIGHT fix (the gate's own fix line: install the CI-pinned binary
   into `~/go/bin`, which host PATH resolves first) only landed after
   two wasted ~10-minute runbook runs. Verify tool resolution BEFORE
   re-running a long runbook.
5. **Inverted edit:** the app.css `.tc-tag-*` deletion went in BACKWARDS
   on the first try, duplicating the block (fixed immediately, but it
   shipped in the tree for one edit cycle).
6. **Mangled splice:** the identity_test.go helper-extraction surgery
   via a mid-file Python splice broke the file's syntax; restored from
   git and redone with drop-and-append edits. Also surfaced a real
   design constraint the hard way: identitytest cannot import identity
   while identity's in-package tests import identitytest — slimmed the
   helper package to walkers-only (plain http.Handler, no identity
   import).
7. **transcriptMatchLines anchored EVERY line** on first rewrite — the
   contract is matched-turns-only anchors (goldens caught it
   immediately).
8. **T08 test registered without `display_name`** — the POST 400'd and
   the test failed confusingly until I reused `loginSession` (which
   asserts 201).
9. **Two vision-model dispatch failures** (glm-4.6v max_tokens illegal;
   synthetic quota exhausted) — the fallback chain ended at direct
   inspection, which worked, but the round-trips cost real time.
10. **Commit fragmentation:** several receipt-grade commits landed thin
    ("1 file changed") because the daemon swept the bulk into
    auto-commits seconds earlier. Content always landed, but receipt
    quality suffered where I didn't commit first or squash-while-
    unpushed consistently.

## e) WHAT WE SHOULD IMPROVE

1. **Add the tidy check to the local gate:** `go mod tidy && git diff
   --exit-code go.mod go.sum` (or the cp/diff shape CI uses) into the
   pre-push battery — CI runs it; locally we didn't. This is the single
   highest-value process fix from the session.
2. **Refuse artifacts at commit time:** a guard (BuildFlow step or
   hook) against tracked binaries and multi-MB generated images would
   have caught the binary + the PNGs before push.
3. **Verify tool resolution before re-running long runbooks:** one
   `which art-dupl && art-dupl version` would have saved two release
   attempts.
4. **Check the dedup gate BEFORE pushing duplicated test helpers** — the
   gate caught it, but post-commit; a pre-commit mental check for
   "moved files with copied helpers" would move the catch left.
5. **Vision fallback chain:** document it (glm-4.6v → synthetic → direct
   view-tool inspection) — the view tool DOES render local images; the
   handoff's "model can't view images" lesson over-corrected.
6. **Squash-while-unpushed discipline:** when a receipt commit matters,
   either commit before the daemon sweeps or squash the span before
   push — pick one and always do it.
7. **`.gitignore` changes need a same-commit `git check-ignore` audit**
   of existing tracked files in the touched glob.
8. **Runbook scripts should self-verify their tool preconditions**
   (cut-release.sh could check `art-dupl version | grep <pin>` before
   the 10-minute gate instead of dying at minute 12) — the gate's
   refusal message helps, but a preflight would help more.
9. **T17/T18-class moves deserve a fresh session with a written
   interface-surgery plan** (which functions stay HTTP, which move) —
   the ratified decision doc + this session's inventory make that
   session cheap to start.
10. **CI-queue latency awareness:** a 16-minute queue+run means "watch
    CI" is a real time cost — batch pushes at natural milestones (done
    mostly right today; the 19:02–19:08 stutter pushed four times in
    six minutes).

## f) UP TO 50 THINGS TO DO NEXT (ranked, grouped)

**Structural (the ratified P1 remainder):**
1. T17: internal/query split — write the interface-surgery plan
   (HTTP-layer stays / read-model moves), then execute.
2. T18: serve composition out of cmd into internal/serve (seam exists:
   serveFlags).
3. Dedupe the 8 test-assembly sites via the new serve constructor (T18
   fine-task F51).
4. PC-2: closed lifecycle union in the decider (CallState booleans →
   sum type; guards become type-impossible).
5. ScanPage + compound-cursor spike on a branch (retire ~95 lines if
   green).
6. Post-move: re-run the structural assessment — measure the churn %
   delta and the package count (proves FM#11's payoff).

**Release/deps cadence:**
7. Cut v0.4.1 (the post-release wave: identity boundary, PC-1, honest
   login, typed seam, bumps) when you want it shipped — CHANGELOG
   `[Unreleased]` is already written.
8. Watch `nix eval nixpkgs#go_1_27.version` ≥ 1.27.4 → flake input +
   go.mod floor in one change + govulncheck zero (T07 close-out).
9. Dependabot PR triage: the minor-and-patch group PR (#…94631) failed
   CI on the same tidy lane — rebase it onto the now-green main or
   dismiss it.
10. art-dupl pin bump rider → surface `--show-suppressed` in the gate
    script + restore the AGENTS line (T23a).
11. go-nix-helpers bump rider → templCliPackage adoption, mkForce
    deletion, devShell templ removal (T23b).
12. go-etag root v0.7.0 stranded at submodules v0.6.1 (#6 filed) — fold
    in on its next bump.
13. cqrs v5 first tag → the 3-line StreamRef→StreamKey census behind a
    flag note (T27/F72).

**Owner decisions waiting (each unblocks a small tail):**
14. Decision pack Q1: PII boundary — raw default vs hash-export default
    vs full hash mode.
15. Decision pack Q2: CSP enforce flip — now / after quiet reports /
    per-page.
16. Decision pack Q3: catalog push — local-only / docs site / telephony
    hub repo.
17. Decision pack Q4: divergence dossier — fold / structured table /
    automate in the refresh classifier.
18. Coverage: accept 70 (at 71.2%) or chase 80 (then pick the surface —
    cmd wiring pilot is the cheapest).
19. Session-TTL: re-affirm 24h or set a HandlerConfig window (one field).
20. TLS-first deploy: pick the proxy (caddy vs nginx) and the window —
    it unlocks passkey login on the LAN + the Secure cookie.
21. FM#11 follow-through ratification: are the post-P1 package sizes
    acceptable, or does the read-side split get a second phase?

**Identity/ops polish:**
22. `pbx-lens identity-export` CLI verb wrapping ExportUsersToJSON
    (runbook section exists; the verb is the convenience).
23. /api/ops: surface the oidclogin drained-handoff error code (F76 —
    the typed error exists, the surfacing is missing).
24. Legacy-keyed rebuild execution in the next deploy window (runbook +
    dry-run tests ready).
25. Identity logical backup: decide whether the JSON export rides the
    nightly backup job beside the VACUUM INTO snapshots.
26. Ops never-ran golden (TODO row 207: `EmptyText: "never"` pin —
    20min).
27. Report render QA for the NEXT report batch (script exists — use it
    at authoring time, not as a later QA pass).

**Filing/outreach (after re-verifying parent mains):**
28. File the fsnotify ask on a-h/templ (watch mode dependency).
29. File the pkg/errors ask on watermill (mechanical stdlib replace).
30. Execute the charm-v1 → charm.land/log/v2 move on go-appkit + systemd
    (owner action), then bump it here.
31. Re-verify the erraudit enforce-* gap against current main; file or
    close the 2026-10-27 draft.

**Docs hygiene:**
32. AGENTS.md: fold the identitytest pattern + the render script into
    the Commands/Structure lines (budget 219/219 — extend, don't add).
33. docs/templ-components-adoption.md: add the ADR-0044 accent-axis
    note to the theme-bridge row (the app.css now bridges TWO axes).
34. docs/http-stack.md: note the poller-health typed seam (Snapshot vs
    the /api/ops wire).
35. Harvest the 2026-11-20 session reports (this one + the mega-review)
    into TODO_LIST rows per the docs-health ritual (this file's §f is
    the seed).
36. Roadmap: the pixel-diff baseline strategy call is still open —
    close it (commit baselines vs /tmp-only) with the screenshot corpus
    as evidence.

**Testing/verification debt:**
37. `go test -race` after the NEXT bump wave (today's ran pre-T05; the
    wave's own code only got the plain suite + CI).
38. A coverage-identity test for the ops wire keys (the lowercase
    SourceStat tags changed /api/ops output — pin it so a revert can't
    silently recase keys).
39. Screenshot-probe baselines: today's UI changed twice (bump + accent
    bridge) — eyeball /tmp/lens-shots against expectations once, then
    re-baseline.
40. cmd/pbx-lens coverage pilot IF the owner says chase (two wiring
    surfaces, ~30min).
41. Add the tidy-check to BuildFlow's go steps (process fix #1 made
    durable).
42. Suite-time budget: the full suite is ~20s locally but CI runs it
    twice (Build & Test + race when added) — measure and note the CI
    ceiling before adding more lanes.

**Small cleanups:**
43. `internal/identity` bench file has 0 test funcs (identity_bench_
    test.go) — verify it still compiles into the suite or move its
    benchmarks (it passed vet; confirm content is wanted).
44. `internal/web/identity_flow_test.go` still carries
    `identityOnlyHandler` + delegates — re-check whether flow/ops tests
    could call identitytest directly now that the alias layer exists.
45. CHANGELOG: the `[Unreleased]` section for v0.4.1 is written —
    line-review it once before the next cut.
46. GitHub: close/refresh the three now-fixed Dependabot alerts (they
    re-scan async; confirm the go-jose alerts actually closed).
47. Remove `/tmp/lens-qa` + `/tmp/report-jpg` scratch (machine-local
    hygiene).
48. Consider a `make receipts`-style script that runs the whole local
    gate battery in one command (the AGENTS Commands block lists them
    individually today).
49. Fleet index refresh check (T27/F73) after today's bumps.
50. Re-read docs/concurrency.md's squash-while-unpushed rule and pick
    ONE consistent commit ritual for the next session (improvement #6
    made durable).

## g) THREE QUESTIONS I CANNOT ANSWER MYSELF

1. **The decision pack's four calls** (PII boundary default · CSP
   enforce flip timing · catalog push destination · divergence dossier
   form) — `docs/decisions/2026-11-20_owner-decision-pack.md` has the
   option sets with `→ ANSWER:` slots; each answer flips a small
   executable tail.
2. **Coverage:** accept the 70 floor (measured 71.2%, honest figure) or
   order the chase to 80 — and if chase, is the cmd/pbx-lens wiring
   pilot the surface you want, or should ingest/CLI paths take the
   volume?
3. **Deploy window + TLS:** when is the next production deploy (it gates
   the legacy-keyed rebuild slot), and do you want the TLS reverse proxy
   (caddy or nginx on the telephony host) in that same window — the
   passkey ceremony currently cannot run over LAN http, so the operator
   login story is blocked on this choice.

---

*Session receipt: 44 commits `280a11d..d81c09d`, pushed; v0.4.0 tagged
and published; all local + CI gates green at HEAD; three red CI runs
(19:02–19:08, tidy lane) resolved by HEAD's state and flagged in §d.1.*
