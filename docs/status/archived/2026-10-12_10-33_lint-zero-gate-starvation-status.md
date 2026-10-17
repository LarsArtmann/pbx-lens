# Status — Lint-to-Zero, Gate Starvation, Release Stalled on the Gate

2026-10-12 10:33 CEST · `main` daemon-swept through `939f194` · machine now
GENUINELY quiet (load 9.4/5.3/4.3 — first time all day); gate run #3 is in
flight in the background at this quiet window (started 10:32 with
BUILDFLOW_NO_RESULT_CACHE=1).

Continues `2026-10-12_07-56_gate-watch-lint-audit-status.md`. This report
covers 07:56–10:33 only.

## a. Fully done

1. **Full-repo lint driven to ZERO (the release blocker cleared).** All 29
   golangci findings fixed across five packages:
   - `internal/web`: `StreamExportCalls` split into `streamPageOptions` /
     `dedupeKey` / `advanceCursor` (gocognit 30 → ~9); justified
     `//nolint:gochecknoglobals` with reasons on `dashboardLocation`
     (injectable zone), `durationBuckets` (declarative table), and
     `exportBatchSize` (test-shrinkable seam); `weekDelta` magic numbers →
     `weekDays`/`percentScale` consts; all `golines` wraps taken verbatim
     from the `golangci-lint fmt --diff` oracle; bodyclose closes in the
     header test; every wsl blank line; three nolintlint removals; headers
     canonicalized `X-PII-Mode` → `X-Pii-Mode` (identical on the wire — Go
     canonicalizes; the old spelling was never what clients saw); gosec G705
     re-suppressed as a same-line directive (the above-line form was
     reported unused while gosec does fire there — same-line is unambiguous).
   - `internal/search`: `Rebuild` DDL extracted to `resetSearchTables`
     (cyclop 14 → ~8) with prepares + defers kept in `Rebuild` so
     sqlclosecheck sees the Close pairs; Query doc-comment dedent; two var
     alignments; digits-merge wsl blank.
   - `internal/cli`: `maps.Copy` replaces the snapshot loop (modernize);
     the four-`mergeStats` chain wrapped; poll-test log uses
     `fmt.Fprintf(&kv, …)` (staticcheck QF1012).
   - `internal/ingest/cdr`: wsl blank lines in the concurrent session's new
     bare-quote test.
   - `cmd/pbx-lens` (earlier in session): errcheck ×3, golines ×2, gci,
     cyclop via `writeDryRunList`.
2. **Verification at ground truth, not LSP cache:** `golangci-lint run ./...`
   → **0 issues**; `golangci-lint fmt --diff ./...` → **empty** (canonical
   formatting everywhere); `go test ./...` → **14/14 packages green**; go vet
   green on all touched packages.
3. **ingest lifecycle extraction landed (§f#13):** `openIndexSession(db)`
   (bounded ctx + composition root + ordered teardown) + `finalizeIngest`
   (drain → rebuild) now carry the shared head/tail of all four ingest verbs
   and purge; telnyx keeps its pre-enrichment drain explicit with its
   contract comment intact.
4. **`unused-dsn` artifact eliminated at the root:** the purge test now
   passes `""` (the honest memory-mode value — rebuild is a documented
   no-op), the 28 KiB stray SQLite file is untracked + trashed, and it can
   never regenerate from a test run.
5. **README `/api/ops` documented (§f#41)** — auth-gated counters endpoint,
   503-with-explanation semantics.
6. **CHANGELOG defect fixed:** duplicated `### Added` heading deduped.
7. **Six new war stories in `docs/gotchas.md`:** lint-green ≠ test-green;
   stale golangci-LSP cache; errcheck's `_test.go`-only scope; test DSN
   literals are real file paths; `golangci-lint fmt --diff` as the formatter
   oracle; `rg -r` means replace.
8. **dprint joined the devShell** (earlier this session, carried in
   `flake.nix`): the documented `nix develop -c dprint fmt` now works.

## b. Partially done

~~1. **R1 gate — TWO manual runs FAILED on spawn starvation, run #3 in~~ done — run #5 green 75/75 at 11:36
   flight.** Run #1 (15-min load ~54): `go-mod-update` killed after 13.3 s
   and `lychee` after 2.4 s — `[transient:tool.timeout] … no output at all
   (hung at or right after spawn)`, i.e. load starvation, zero code signal.
   Run #2 (15-min ~47): same class, `test-coverage` killed after 1.1 s plus
   one more step. **Run #3 started 10:32 at load 4–9** — the first genuinely
   quiet window of the day; result pending. The old watcher (12E) was killed
   in favor of manual runs.
~~2. **The htmx vendored-asset mystery — damage controlled, culprit NOT~~ done — culprit = oxfmt/oxlint provider gap; bytes restored + global exclude (12-23 a2)
   found.** A gate run's step un-minified the vendored `htmx.min.js`
   (1 line → 3,416). I restored the pinned bytes once, but run #2 reformatted
   it AGAIN and a daemon sweep has now COMMITTED the reformatted version
   (`939f194`). BuildFlow's JS profile explicitly default-excludes
   `*.min.js`, so the culprit is some other step — identified but not named
   yet. The server serves the file byte-for-byte, so the 3,416-line version
   works functionally, but vendored assets must stay byte-pinned.
~~3. **R4 release — fully staged, not executed:** release flow designed~~ done — v0.1.0 released 12:23
   (rename `[Unreleased]` → `[v0.1.0] — 2026-10-12` in CHANGELOG, push main,
   annotated tag, `gh release create` from the changelog, `nix run . --
   version` — the binary's hardcoded `version = "0.1.0"` will match the
   tag). Blocked ONLY on a green gate.

## c. Not started

~~- **R4 execution** (gate-gated; staging done).~~ done — v0.1.0 released 12:23
~~- **§f#10 DID-direction implementation** — deliberately NOT started: doing a~~ done — shipped 2026-10-12 (-ai-did; CHANGELOG)
  feature between gate-green and tag would invalidate the gate.
~~- **Sibling repo** (flake check rerun, fmt strategy, pushes) — owner §g~~ done — sibling green, pushed e7c0f8b
  answers still pending.
~~- **CI verification post-push**, m1.2 gate-sample recording, gochecknoglobals~~ done — CI green; m1.2 recorded; nolint rationales live in code
  audit notes into docs (the nolint rationales live in code now).

## d. Totally fucked up (honest)

1. **I fired an accidental gate run while gathering data for THIS report.**
   `rg -A2 "failed:" … buildflow …` — I pasted the RERUN command instead of
   reading job 199's saved output. It moved to background (19C) mid-report.
   Mitigation: at load 4–9 it is exactly the run we wanted, but that is luck,
   not judgment.
2. **`rg -r` misuse, SECOND time this session** (`-rln` parsed as
   `-r ln`, silently replacing matches in output with "ln"). The first one
   is even documented in gotchas NOW. Paid twice for the same flag.
3. **The gate reformatted a vendored asset TWICE and I only caught the first
   one** — I restored `htmx.min.js` after run #1, did NOT re-check after run
   #2, and the daemon committed the reformatted 3,416-line version. A
   vendored file's integrity must be re-verified after every gate run until
   the culprit step is fixed (and a checksum pin considered).
4. **Gate-vs-load doctrine vs the owner's pre-approval tension:** the release
   is owner-approved "YES once gate green", but the gate has burned ~25
   minutes across two starvation-failed runs. I chose option (b) (run at
   over-budget load, record honestly) without an explicit owner answer to
   §g Q1 — defensible, but it cost two red runs I could have avoided by
   waiting ~2 more hours for the machine to actually quiet down.
5. **Repair steps mutate the tree silently.** Run #1's repair list
   (go-mod-update / nix-flake-update / nix-fmt / templ-fmt "ran (not
   measured)") changed files beyond the failures; I audited only via
   `git status` and caught htmx by luck of its size change. A gate that
   repairs should be followed by a deliberate diff review every time.

## e. What we should improve

1. **After ANY buildflow run, diff-review the tree before anything else** —
   repair/format steps write freely; a `git status` + targeted `git diff`
   ritual belongs in the gate loop (and a checksum pin for vendored assets).
2. **Find and fix the htmx-culprit step** — the JS profile's `*.min.js`
   exclude does not cover whichever tool reformatted it; either extend that
   tool's excludes in BuildFlow or pin vendored bytes with a checksum gate.
3. **Spawn-starvation is a systemic gate failure mode:** 3 of 3 gate attempts
   today lost steps to `tool.timeout … hung at or right after spawn` under
   load ≥ ~45 (1-min). Consider a pre-flight load check inside the gate
   itself (fail fast with "machine busy" instead of starved child tools) or
   per-step spawn retries with backoff — worth filing upstream (BuildFlow).
4. **rg flag hygiene:** ban myself from `-r` outside deliberate replaces;
   the gotcha exists, the second burn shows a rule, not a note, was needed.
5. **Never launch verification commands by re-pasting:** keep a saved
   "inspect" command set separate from "execute" ones.

## f. Next tasks (ranked)

~~1. Read gate run #3's result (job 19C, started 10:32 at load 4–9).~~ done-superseded — run #5 green (11:36)
~~2. If green → R4: CHANGELOG `[Unreleased]` → `[v0.1.0] — 2026-10-12`, dprint,~~ done — v0.1.0 released 12:23
   commit, push main, annotated tag, `gh release create` from changelog,
   `nix run . -- version`, confirm the tag matches the binary, watch CI.
~~3. If run #3 starved again → rerun at load ≤ ~10 (it is 4–9 now; this IS the~~ moot — run #3 superseded by run #5
   window) and re-verify `htmx.min.js` byte-integrity after.
~~4. Restore vendored `htmx.min.js` (3,415-line reformatted version is~~ done — bytes restored + `.buildflow.yml` global exclude (12-23 a2)
   committed at `939f194`) and prevent recurrence: identify the culprit step
   (`buildflow -s <fmt> --verbose` bisection), extend its excludes, add a
   checksum pin for `internal/web/static/`.
~~5. Record the m1.2 gate sample in `docs/buildflow-dev-gate.md` with the~~ done — run #5 entry in buildflow-dev-gate.md
   honest load context of whichever run goes green; retire "pending" wording.
~~6. File the spawn-starvation failure mode upstream (BuildFlow issue: tools~~ done — BuildFlow#50 filed; #45 fixed v0.8.0
   killed at spawn under load; propose pre-flight load check / spawn retry).
~~7. Coordinate the tag with the concurrent session before pushing (§g#2).~~ done — released clean
~~8. §f#10 DID-direction implementation (post-release):~~ done — DID-direction shipped
   `DirectionResolver` mapper + `-ai-did` flag + fixtures, then the sibling
   `-ai-did` module option.
~~9. Sibling: repo-wide `nix fmt` vs per-file (§g#3), `nix flake check` green~~ done — sibling green/pushed (deploy row remains)
   (PIPESTATUS trap), host re-deploys with the new module options, push.
~~10. Watch CI on push: templ regen, golangci v2.14, tidy-diff with the htmx~~ done — CI green
    require, e2e probe step, backup-drill-as-step (§f#48).
~~11. Annotate §g answers into TODO_LIST/ROADMAP once decided.~~ done — resolved by events (R4)
~~12. TODO_LIST: flip release + sibling rows DONE after they land; harvest~~ done — sixth-pass harvest
    this report's §f (docs-health HARVEST).
~~13. `nix build` re-verify after the CHANGELOG/README edits (doc-only, but~~ done — nix build green on the doc-only tree
    the gate hash changes).
~~14. §f#12 residue: audit remaining `time.Sleep` in tests for the~~ → routed: recording timeout-test row
    cancellation-driven pattern (WAV test is the template).
~~15. gochecknoglobals audit note: move the three nolint rationales into~~ NOT-DO — rationales live as code comments at the nolint sites
    `docs/` (cqrs-lint-policy-adjacent) so they survive refactors.
~~16. Keyset stall UX: surface `export_cursor_stalled` in the export UI.~~ done — export honesty note (ba6ec8f)
~~17. Range-picker presets (7/30-day buttons).~~ done — 7/30-day presets
~~18. Dashboard cost panel (theme-1 leftover).~~ → routed: ROADMAP theme 1 cost panel
~~19. Search hint chip for digits-only search in the UI.~~ done-superseded — R6 hint copy
~~20. SSE upgrade path if poll volume ever matters.~~ → routed: ROADMAP (R9 SSE note)
~~21. Staleness fragment filter carry.~~ NOT-DO — banner unfiltered-only by design
~~22. Export memory budget test for `exportBatchSize`.~~ done — TestExportMemoryBudgetStreamsPastOldCap
~~23. FEATURES evidence refs: keep rows file-level.~~ NOT-DO — line refs retained deliberately
~~24. CHANGELOG: fold the release under the version heading (done inside R4).~~ done — v0.1.0 heading
~~25. Plan file re-annotation post-gate/release (ANNOTATE, never rewrite).~~ done — plans annotated; this sweep finishes the set
~~26. `nix run . -- version` re-verify after the tag (inside R4).~~ done — receipts verified
~~27. Re-tag vendorHash if go.mod moves (go-mod-update failed twice today —~~ done — vendorHash repairs via the app (04-45)
    check `git status` for reverted go.mod state).
~~28. Consider `docs/drafts/` README table.~~ done — drafts README live
~~29. Dependabot gomod pinning decision (owner call, #16538 context).~~ → routed: dependabot-core#16538 watch row
~~30. Watch dependabot-core#16538; answer maintainer questions.~~ → routed: dependabot-core#16538 watch row
~~31. Watch BuildFlow#45; close referencing fix commits once CI confirms.~~ done — v0.8.0 released with the fix
~~32. Owner: tick or drop the go-atomic-write draft review box.~~ → routed: go-atomic-write draft (docs/drafts/README.md)
~~33. Telnyx live DTO re-verify window opens 2027-01 (needs API key).~~ → routed: Telnyx quarterly row
~~34. BuildFlow: cherry-pick the gate fix into pinned release branches.~~ done — v0.8.0
~~35. `/api/ops`: lastPass age warning when stale.~~ done — last_pass_age_seconds (v0.2.0)
~~36. Backup drill as a named CI step.~~ done — named CI step
~~37. Verify the two status reports + README/CHANGELOG edits all landed in~~ done — files committed (0ae3b15/a47ba32 confirmed)
    daemon sweeps (`0ae3b15`/`a47ba32` look like them — confirm).
~~38. Post-release: harvest §f into TODO_LIST, archive superseded reports.~~ done — harvest passes done; archived by this sweep
~~39. Retire the "concurrent session" unknowns: confirm its buildflow bursts~~ done — concurrency norms recorded (ROADMAP)
    are done before tagging (load history says yes as of 10:33).
~~40. R27.x residue: art-dupl pass, branching-flow pass, nix-checker cleanup,~~ → routed: R27.x advisory row
    README screenshots, vulnix re-triage (unchanged priorities).

## g. Questions I cannot answer myself

~~1. **Gate #3 is running right now (started 10:32, load 4–9).** If it comes~~ resolved — v0.1.0 released on the standing approval
   back green, do I proceed straight into the full R4 sequence (CHANGELOG
   rename → push main → annotated tag → GitHub release), or do you want to
   eyeball the tree first? The concurrent session looks done (load 4–9), but
   only you know if more bursts are scheduled.
~~2. **The reformatted `htmx.min.js` is now committed** (`939f194`, un-minified~~ resolved — bytes restored pre-tag + the exclude rule held
   3,416 lines, functionally identical, served byte-for-byte). Restore the
   pinned upstream bytes before tagging v0.1.0, or ship v0.1.0 with the
   pretty-printed asset and fix the pin in a follow-up?
~~3. **Gate policy going forward:** keep the "15-min load < 40" doctrine (it~~ done — load policy recorded (1-min <15 AND 15-min <40; buildflow-dev-gate)
   never once fired today), switch to "1-min < 15" (would have passed 3 of 3
   attempts), or run unconditionally and retry starvation failures once?
   I can implement any of the three — the choice changes when v0.1.0 ships.
