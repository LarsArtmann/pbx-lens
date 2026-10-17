# Status — Gate Watch, Lint Audit, Session Hardening

2026-10-12 07:56 CEST · branch `main` at `ef0cc28` (+ uncommitted: flake.nix dprint,
AGENTS.md, docs/buildflow-dev-gate.md, recording_test.go) · watcher job 12E armed,
15-min load 56 (was 69–107), still above the 40 gate threshold.

This report covers the CURRENT session only (07:20–07:56), continuing
`2026-10-12_07-13_SUPERB-plan-execution-status.md` (that file sat untracked — now
being `git add`-ed so the daemon sweeps it).

## a. Fully done (this session)

1. **LSP diagnostic triage (ground truth).** All 4 stale warnings disproven with
   compiler-level checks: `go vet` + `go test` green on `internal/backup` and
   `cmd/pbx-lens`. The `timePtr` ghost in `restore_drill_test.go` and the three
   `ops_test.go` findings (nlreturn/gci/predeclared) reference lines that do not
   exist in the files on disk — stale golangci-LSP cache, not real findings.
2. **§f#31 — AGENTS.md line budget:** 210/220. OK.
3. **§f#37 — daemon sweep verified:** `ef0cc28` = exactly the 5 dprint-reflowed
   files (FEATURES, TODO_LIST, DOMAIN_LANGUAGE, treefmt-nix-545 comment, plan).
4. **§f#2 — dprint devShell fix (root cause).** `dprint` added to the
   `mkShellNoCC` devShell in `flake.nix`; verified `nix develop -c dprint
   --version` → 0.60.1 (the env drv built, so the flake evaluates clean). Docs
   updated: `docs/buildflow-dev-gate.md` annotation + AGENTS.md (Commands block
   and stack-decision tool list).
5. **§f#12 — timeout-test hardening.** `TestRecordingUpstreamTimeoutBoundsWavHeaders`
   no longer sleeps 250 ms against a 50 ms budget: the fake audio endpoint now
   blocks on `r.Context().Done()` and the injected `UpstreamTimeout` fires.
   Same 502 contract, no wall-clock race, pass time 0.07 s (was ~0.3 s).
6. **§f#43 superset — REAL lint audit of `cmd/pbx-lens` (pre-gate blocker
   discovery).** Ran golangci-lint v2.14 scoped (ground truth, not LSP): **6 real
   findings in fresh code that WOULD fail the release gate**:
   - `ops.go:211/214/243` errcheck ×3 — unchecked `fmt.Fprintf(out, …)` in
     `runPurge` (the `_test.go` errcheck exclusion does not cover non-test
     files; a lost report write would exit 0 silently),
   - `ops.go:170` golines (runPurge call > 120 cols),
   - `serve.go:149` golines (`web.Register(…)` > 120 cols),
   - `ops.go:235` gci (file-format diff, exact reflow to be read via `fmt --diff`).
     Fix lands immediately after this report (mechanical, before any gate/release).

## b. Partially done

~~1. **§f#13 — ingest.go extraction.** Design settled: `openIngestSession(db)`~~ done — 83fe566 extraction (10-33 a3)
(timeout ctx → OpenSystem → combined cleanup, mirroring the existing
4-value return style) + `finalizeIngest(ctx, sys, dsn)` (drain → rebuild).
All four verbs (cdr/telnyx/webhook/agent) share the head; cdr/webhook/agent
share the tail verbatim; telnyx keeps its pre-enrichment drain explicit.
NOT yet applied.
~~2. **R1 gate.** Watcher 12E alive, zero fires; load 69→56 and trending down but~~ done — run #5 green 75/75 at 11:36

> 40 all session. The concurrent session ran its own buildflow (07:21–07:25).
> ~~3. **The 6 lint findings** — discovered + fix designed (error-checked Fprintf~~ done — fixed same day; lint zero (10-33 a1)
> that returns the error; wrapped long calls), not yet applied.
> ~~4. **Session reporting hygiene** — previous report was untracked (the daemon~~ done — files committed; daemon swept
> does not sweep untracked files); both status files are being added to git now.

## c. Not started (this session)

~~- **R4 release** (tag v0.1.0, `gh release`, push) — blocked on R1 green.~~ done — v0.1.0 released 12:23
~~- **m1.2 gate sample** into `docs/buildflow-dev-gate.md` — needs a green gate.~~ done — run #5 entry in docs/buildflow-dev-gate.md
~~- **§f#10 DID-direction implementation** (design doc exists, no code).~~ done — shipped 2026-10-12 (-ai-did; CHANGELOG)
~~- **Sibling repo work** — `nix flake check` rerun, straggler nixfmt, pushes~~ done — sibling flake check green, pushed e7c0f8b
(waiting on owner §g answers).
~~- **gochecknoglobals check** on `dashboardLocation`/`durationBuckets` — scoped~~ done — rationales in code; lint zero (10-33)
lint of `internal/web` not yet run this session; the gate or a scoped run
will answer it.

## d. Totally fucked up (honest)

1. **Release-blocking lint debt sat in fresh code for a day.** The SUPERB waves
   wrote `runPurge`'s unchecked `fmt.Fprintf` calls and two >120-col lines, and
   deferred lint verification to "watch the gate" — the gate never ran (load),
   so the findings were invisible. Only this session's scoped ground-truth lint
   caught them. A push + tag yesterday would have gone straight to red CI.
2. **I nearly trusted "go test green ⇒ lint clean."** The previous session's
   handoff said to ignore the LSP warnings; I verified with vet/test (correct)
   but initially extended that trust to lint. What forced the truth out was the
   LSP emitting FRESH findings for `ops.go` post-restart — the scoped
   golangci-lint run confirmed 6/6. Test-green is not lint-green; only the
   linter is the linter.
3. **Tooling slip:** used `rg -rn` where `-r` means _replace_; one read-only
   output came back mangled ("dprint" → "n"). Caught immediately, redone
   correctly. No harm, but sloppy flag hygiene.
4. **Report latency:** the owner demanded this status report "RIGHT NOW" in the
   previous message; I only gathered state before the follow-up instruction
   arrived. It lands one turn late.

## e. What we should improve

1. **Hard rule: scoped `golangci-lint run <touched pkgs>` before ending ANY
   wave.** "Watch the gate" is not verification when the gate cannot run.
2. **Never dismiss LSP lint findings on summary say-so** — re-lint the package
   (or verify the finding's cited line against disk) before acting either way;
   the golangci LSP cache goes stale across sessions.
3. **errcheck policy note:** non-test `fmt.Fprintf(out io.Writer, …)` in report
   writers must propagate the error — a failed write to stdout/buffer must not
   exit 0 with the report lost.
4. **Prefer cancellation-driven deadline tests over sleep budgets** (the WAV
   test is now load-immune and 4× faster) — audit remaining `time.Sleep` in
   tests for the same pattern.
5. **The daemon never sweeps untracked files** — new files must be `git add`-ed
   explicitly or they are one `git clean` away from loss.

## f. Next tasks (ranked)

~~1. Fix the 6 lint findings (errcheck ×3 returning write errors; wrap the two~~ done — 10-33 a1 (29 findings fixed)

> 120-col lines; read + apply the gci/golines formatter diff for ops.go).
> ~~2. Scoped lint `cmd/pbx-lens` → 0 issues; then scoped `internal/web` (answers~~ done — lint zero (10-33 a2)
> the gochecknoglobals watch item).
> ~~3. Apply the ingest session extraction (§f#13) with build+test+lint.~~ done — 83fe566
> ~~4. Full `go build ./...` + `go test ./...` sweep.~~ done — suite green
> ~~5. R1: full gate when load < 40 (watcher 12E armed); record the m1.2 sample;~~ done — run #5 (11:36)
> retire "pending" wording in buildflow-dev-gate.md.
> ~~6. R4: annotated tag v0.1.0, GitHub release from CHANGELOG, verify install~~ done — v0.1.0 released
> paths, push main+tag (owner pre-approved "YES once gate green").
> ~~7. Coordinate with the concurrent session before tagging (release race, §g#2).~~ done — release clean (12:23)
> ~~8. Verify CI green on push (templ regen, lint v2.14, tidy-diff with htmx~~ done — CI green on push
> require, e2e step).
> ~~9. §f#10 DID-direction implementation (`DirectionResolver` + `-ai-did` +~~ done — DID-direction shipped
> fixtures) per docs/did-direction-design.md; sibling option after.
> ~~10. Sibling: repo-wide `nix fmt` vs per-file (§g#3), `nix flake check` green~~ done — sibling green/pushed (deploy row remains)
> (remember `${PIPESTATUS[0]}`), then push when owner approves.
> ~~11. Audit remaining `time.Sleep` in tests for cancellation-driven rewrites~~ → routed: recording timeout-test row
> (improvement #4).
> ~~12. R27.1 vulnix re-triage after next bump.~~ → routed: R27.x advisory row
> ~~13. R27.2 art-dupl pass (commands.go mixins, Observation/CallView overlap).~~ → routed: R27.x advisory row
> ~~14. R27.3 branching-flow pass (fold genuinely complex functions only).~~ → routed: R27.x advisory row
> ~~15. R27.4 nix-checker hash-bookkeeping cleanup.~~ → routed: R27.x advisory row
> ~~16. R27.5 README screenshots (four surfaces, auth-safe fixture).~~ done — docs/assets convention (09eb780)
> ~~17. Watch dependabot-core#16538; answer maintainer questions with PR#1 evidence.~~ → routed: dependabot-core#16538 watch row
> ~~18. Watch BuildFlow#45; close referencing the fix commits once CI confirms.~~ done — BuildFlow#45 fixed v0.8.0
> ~~19. Owner: tick or drop the go-atomic-write draft review box.~~ → routed: go-atomic-write draft (docs/drafts/README.md)
> ~~20. Telnyx quarter calendar: live DTO re-verify window opens 2027-01 (needs key).~~ → routed: Telnyx quarterly row
> ~~21. Keyset stall UX: surface `export_cursor_stalled` in the export UI.~~ done — export honesty note (ba6ec8f)
> ~~22. Range-picker presets (7/30-day buttons).~~ done — 7/30-day presets
> ~~23. Dashboard cost panel (theme-1 leftover).~~ → routed: ROADMAP theme 1 cost panel
> ~~24. Search affordance: digits-only hint chip in the UI.~~ done-superseded — R6 hint copy tells the digits truth
> ~~25. SSE upgrade path if poll volume ever matters.~~ → routed: ROADMAP (R9 SSE note)
> ~~26. Staleness fragment: carry filters if the banner ever shows filtered.~~ NOT-DO — banner renders unfiltered-first-page only, by design
> ~~27. Export memory: budget test pinning `exportBatchSize` behavior.~~ done — TestExportMemoryBudgetStreamsPastOldCap (ba6ec8f)
> ~~28. FEATURES evidence refs: keep rows file-level (line refs rot).~~ NOT-DO — line refs retained deliberately; audits re-verify them
> ~~29. TODO_LIST: flip release + sibling rows DONE after 6/10 land.~~ done — rows evolved into the deploy row
> ~~30. CHANGELOG: fold the release under the version heading when tagging.~~ done — CHANGELOG v0.1.0 heading
> ~~31. Plan file: re-annotate post-gate/release state (ANNOTATE, never rewrite).~~ done — plans annotated; this sweep finishes the set
> ~~32. `nix run . -- version` re-verify after the tag.~~ done — receipts (v0.1.0/v0.2.0)
> ~~33. Re-tag vendorHash if go.mod moves before the release.~~ done — repaired via the sanctioned app (04-45)
> ~~34. Consider `docs/drafts/` README table (state per draft at a glance).~~ done — drafts README live
> ~~35. Dependabot config: consider pinning gomod updates until #16538 (owner call).~~ → routed: dependabot-core#16538 watch row
> ~~36. Annotate §g answers into TODO_LIST/ROADMAP once decided.~~ done — resolved by events
> ~~37. BuildFlow: cherry-pick the gate fix into any pinned release branch.~~ done — v0.8.0 released with the fix
> ~~38. pbx-lens CI: confirm the e2e step passes on first push.~~ done — CI green repeatedly
> ~~39. `/api/ops`: lastPass age warning in the payload when stale.~~ done — last_pass_age_seconds (v0.2.0)
> ~~40. Backup drill: wire into CI as a named step like the e2e probe.~~ done — named "Backup restore drill" CI step
> ~~41. README: document polling surfaces + `/api/ops` in the serve section.~~ done — README operational-surfaces block
> ~~42. Post-release: harvest this report's §f into TODO_LIST (docs-health HARVEST).~~ done — harvested; archived by this sweep

## g. Questions I cannot answer myself

~~1. **Release gate vs machine load (carried over):** load has stayed 45–107 for~~ resolved — v0.1.0 released on the standing approval
1.5+ h. For v0.1.0: (a) keep waiting for the quiet window (watcher armed),
(b) run the full gate NOW at over-budget load and record it as such, or
(c) postpone the release until the machine is genuinely quiet?
~~2. **Concurrent session (carried over):** its buildflow is running right now.~~ resolved — concurrency norms recorded (ROADMAP open question)
Is it yours and safe to race a tag + push, or should I hold until it lands?
~~3. **Sibling formatting (carried over):** one repo-wide `nix fmt` commit in~~ done — sibling green (per-file fixes held; flake check passed)
nix-international-telephony (touches files another session may edit), or
per-file fixes only as `nix flake check` names them?
