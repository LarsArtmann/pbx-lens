<!-- docs/status/2026-10-12_12-23_release-did-sibling-gate-green-status.md —
     session report: gate closed green, v0.1.0 released, DID-direction
     shipped, sibling aiDids option landed. Written 2026-10-12 12:23 CEST. -->

# Status 2026-10-12 12:23 — release v0.1.0 + DID-direction + sibling option

Resumed from the 10:33 handoff (gate run #3 unknown). This session: read
run #3 (failed, NEW cause), root-caused the htmx un-minifier, fixed a REAL
test race the gate exposed, closed the gate green twice, executed R4
(v0.1.0 released), shipped the DID-direction slice, landed the sibling
module option, and harvested the 10:33 report into TODO_LIST (sixth pass).

## a. Fully done

1. **R1 gate: CLOSED GREEN — twice.** Run #3 (job 19C) had failed with a
   new deterministic cause: the flake `checks.pre-commit` (templ fmt
   fail-on-change rejected dashboard.templ's unsorted imports) + oxlint 1
   finding. Run #4 exposed a REAL data race (`TestSqliteEndToEndAndReplay`)
   and a nix-build kill (self-inflicted, see §d). Run #5 (11:35): **75
   success, 0 failed** in 55.3s at load 6.5/8.6/13.4. Run #6 (12:00) on
   post-release main: **75/0 again**, 55.7s. Both runs' only findings are
   the documented advisories.
2. **htmx.min.js mystery SOLVED, pinned, defended.** Culprit: BuildFlow's
   oxfmt provider triggers on plain `**/*.js` (`webFilePatterns`) with NO
   `*.min.js` exclude — that exclude lives in the JS _profile_ defaults
   (`js_profile.go:32`), which the provider never consumes; oxlint's
   `jsFilePatterns` has the same gap. Fix: pinned bytes restored from
   `bd51d29` + global `exclude: ["*.min.js"]` in `.buildflow.yml`
   (`MatchExcludePattern` globs basenames, so the nested path matches).
   Verified scoped (`buildflow -s oxfmt`/`-s oxlint` → filesScanned: 0) and
   the file survived two full green gates.
3. **templ fmt / pre-commit coupling fixed.** `templ fmt` sorts Go imports
   inside `.templ`; dashboard.templ had `forms` before `feedback`. Fixed;
   `nix build .#checks.x86_64-linux.pre-commit` standalone green (exit 0).
4. **REAL test race fixed (paid for by run #4).** `startProjections` fired
   `Start(hostCtx)` bare; `defer sys2.Close()` runs BEFORE
   `t.Cleanup(stop)`, so `Host.Stop()` wrote host state while `Start` read
   it — only detectable under package-wide shuffle + load (not
   reproducible with `-run` alone, 3 attempts). Fix: `startProjections`
   returns `stopAndWait`, called before EVERY `System.Close`; verified
   with the gate's exact shuffle seed ×2 + 5 plain race repeats.
5. **R4 RELEASED: v0.1.0.** Followed the go-release skill: go.mod hygiene
   (no replaces/pseudo-versions), CI diagnosis first (the 04:54/05:00 red
   runs were the pre-lint-zero tree — only the Lint job failed, listing
   exactly the 29 since-fixed findings), pushed `2868910`, **watched CI go
   green (all 5 jobs) before tagging**, annotated tag v0.1.0 on the
   CI-verified commit, pushed, `gh release create` (Pre-release per v0.x
   doctrine; notes voice-checked: 0 FAIL 0 WARN), `nix run . -- version` →
   `pbx-lens 0.1.0`. Live: releases/tag/v0.1.0.
6. **DID-direction shipped (R22 post-release slice).**
   `telnyx.MapOptions.ResolveDirection` + `DIDDirectionResolver` (checks
   both parties; DIDs do not originate calls) + `ParseAIDIDs`;
   `TelnyxOptions.AIDIDs` + `PollConfig.AIDIDs` wiring; `-ai-did` on both
   `ingest-telnyx` and `serve`; fixture trio + parser test
   (table-driven). Scoped lint/fmt clean, CI GREEN on the push
   (38232649559), final gate green.
7. **Sibling module option landed.**
   `services.telephony.pbx-lens.aiDids` (nullOr nonEmptyListOf
   nonEmptyStr) wired as `-ai-did <comma-joined>`; `nix fmt` 0-changed;
   **`nix flake check` ALL CHECKS PASSED** (hosts evaluate); pushed
   (`e7c0f8b`). Discovered the old TODO row was stale: `-max-age` and
   `-recording-timeout` were already exposed AND wired — resolved against
   code, replaced by the deploy row.
8. **TODO_LIST sixth-pass harvest** of the 10:33 report §f: v0.1.0 + DID
   rows removed (CHANGELOG carries them), stale sibling row resolved, new
   rows added (backup-drill CI step as the one Unblocked Medium; seven
   bounded Low rows; two Blocked externals), sixth-pass note written.
9. **Docs memory:** 4 gate-doc entries (oxfmt min.js, templ-fmt coupling,
   tree-freeze rule, race teardown pattern) + run #5 green sample +
   evidence-based **gate load policy** (1-min < 15 AND 15-min < 40, tree
   frozen during runs); 5 gotchas; AGENTS.md DID note (213/220 lines);
   CHANGELOG `[Unreleased]` DID entry.

## b. Partially done

~~1. **Sibling CI on `e7c0f8b`: in_progress** at write time (run~~ done — sibling CI green; e7c0f8b pushed
   38233052109, ~27 min — longer than its usual lane; earlier sibling
   pushes also still in flight). Local `nix flake check` is green; the CI
   verdict is the open tail.
~~2. **§g answers (10:33 report): effectively executed, not ratified.** All~~ done — ratified as the standing norm (buildflow-dev-gate load policy + ROADMAP machine-hygiene)
   three questions resolved by action (R4 proceeded on the standing
   approval; htmx restored pre-tag; load policy recorded from evidence) —
   but you have not explicitly ratified the policy wording.
~~3. **Harvest routing done, source reports untouched.** The 10:33/07:56~~ done — this 2026-10-17 sweep annotates + archives them
   reports are not yet annotated/archived (rowed: §f.38) — HARVEST by
   design never edits the source.
~~4. **Host deploys:** the `aiDids` option exists but no host consumes it —~~ → routed: host deploy row (TODO_LIST)
   blocked on the deploy path (owner).

## c. Not started

~~- The seven new Low rows (gochecknoglobals rationale docs, keyset-stall~~ done — all seven dispositioned in §f below (landed, superseded, or NOT-DO)
  UX, range presets + digits hint chip, staleness filter carry, export
  memory-budget test, `/api/ops` staleness warning, drafts README).
~~- Backup drill as a named CI step (the one Unblocked Medium).~~ done — named ci.yml step live
~~- Upstream filings: BuildFlow spawn-starvation failure mode; NEW from this~~ done — BuildFlow#50 filed; the oxfmt min.js gap NOT filed (repo-side exclude shipped; §f4)
  session — the oxfmt/oxlint `*.min.js` provider gap.
~~- Injected-deadline refactor of the recording timeout tests; ingest.go~~ → routed: timeout-test row; the ingest extraction landed (83fe566)
  (269-line) extraction — both pre-existing rows, untouched today.

## d. Totally fucked up (honest)

1. **I edited docs MID-GATE (run #4).** The mid-run edit re-dirtied the
   source; nix re-evaluated the `-dirty` derivations and started a SECOND
   go-modules + full build beside the running tests; nix-build was killed
   8s in. Cost: one full gate run + a wrong "OOM" conclusion risk. Now a
   recorded rule (tree frozen for the duration); I still broke my own
   discipline the same session I was enforcing R1.
2. **Sloppy multiedits — three near-misses, one real hit.** One multiedit
   silently deleted the `cmd, err :=` line in app_test.go (caught on
   re-view, fixed); one edit REPLACED the templ-fmt gotcha entry instead
   of appending (memory loss — caught and restored); one edit was rejected
   for a stale read and I briefly believed I had clobbered the 00:45
   history entry (no damage). Pattern: hand-typed old_string against
   files I had only seen via bash, not the view tool.
3. **The race survived three prior green runs.** `TestSqliteEndToEndAndReplay`
   has shipped unsynchronized Start/Close since it landed; "14/14 green"
   was weaker evidence than it sounded — the gate's shuffle+load was the
   only thing that ever exposed it. Lesson recorded: the shuffle seed in
   the step log is the reproduction tool.
4. **The htmx damage shipped in two gate runs and one daemon commit
   (`939f194`) before I root-caused it.** Restored before the tag, so
   v0.1.0 carries the pinned bytes — but I initially accepted "BuildFlow's
   JS profile excludes *.min.js" as exculpatory without checking that the
   _provider_ layer never consumes those defaults.

## e. What we should improve

1. Gate discipline as muscle memory: docs before/after, never during; the
   recorded rule needs to be followed the first time, not after paying.
2. Edit hygiene: view tool before every edit (bash `sed`/`cat` reads do
   not register), prefer whole-symbol replaces for code, append-by-anchor
   with the anchor AND its successor line for docs.
3. "Green" claims should name the detector: 14/14 green did not include
   shuffle+race at load; say what ran, not just what passed.
4. Provider-level excludes in BuildFlow are a systemic gap (profile
   defaults vs provider triggers) — worth one upstream issue covering both
   oxfmt and oxlint, not two.
5. TODO_LIST Low section is growing (12+ rows) — next AUDIT pass should
   re-rank by impact or the list becomes a junk drawer.
6. telnyx_test.go crossed 836 lines (file-size-check advisory grew from
   749 after my tests) — split it when the next wave touches it.

## f. Next tasks (ranked; ledger = TODO_LIST sixth pass)

~~1. Watch the sibling CI verdict on `e7c0f8b` (in flight).~~ done — sibling CI green on e7c0f8b
~~2. Backup drill as a named CI step (Unblocked Medium).~~ done — named ci.yml step live
~~3. Owner: deploy hosts with `aiDids` (needs DID values + deploy path).~~ → routed: host deploy row (TODO_LIST)
~~4. File BuildFlow upstream: oxfmt/oxlint `*.min.js` provider gap (with the~~ NOT-DO — never filed; the `.buildflow.yml` global exclude shipped instead (buildflow-dev-gate oxfmt bullet)
   `.buildflow.yml` global-exclude workaround).
~~5. File BuildFlow upstream: spawn-starvation failure mode (runs #1/#2~~ done — BuildFlow#50 filed (a061fbb)
   evidence).
~~6. ANNOTATE + archive the two 2026-10-12 status reports (§f.38).~~ done — this 2026-10-17 sweep
~~7. Re-annotate the SUPERB plan post-release (§f.25).~~ done — plans annotated (12-47/13-32/17-31 EXECUTED)
~~8. gochecknoglobals rationale docs (§f.15).~~ NOT-DO — rationales live at the nolint sites (visible in lint context)
~~9. Keyset-stall UX in the export UI (§f.16).~~ done — export honesty note (ba6ec8f)
~~10. Range-picker presets + digits-only search hint chip (§f.17/19).~~ done — presets shipped; the chip superseded by the R6 hint copy
~~11. Staleness fragment filter carry (§f.21).~~ NOT-DO — banner unfiltered-only by design
~~12. Export memory-budget test for `exportBatchSize` (§f.22).~~ done — TestExportMemoryBudgetStreamsPastOldCap
~~13. `/api/ops` lastPass age warning (§f.35).~~ done — last_pass_age_seconds (v0.2.0)
~~14. `docs/drafts/` README table (§f.28).~~ done — docs/drafts/README.md live
~~15. Injected-deadline refactor, recording timeout tests (pre-existing row).~~ → routed: recording timeout-test row
~~16. `cmd/pbx-lens/ingest.go` extraction (pre-existing row).~~ done — 83fe566
~~17. Split `telnyx_test.go` (836 lines; grew this session).~~ NOT-DO — accept-and-document (835 lines, table-driven; R27.x triage)
~~18. Prod re-ingest ContentHash v2 (Blocked, owner).~~ → routed: prod re-ingest row
~~19. Watch dependabot-core#16538; answer maintainer (Blocked).~~ → routed: dependabot-core#16538 watch row
~~20. R27.x advisory residue pass (Blocked).~~ → routed: R27.x advisory row
~~21. Telnyx live DTO re-verify (window opens 2027-01).~~ → routed: Telnyx quarterly row
~~22. SegmentBar swap when templ-components publishes it.~~ → routed: ROADMAP SegmentBar re-eval
~~23. ContentHash length-prefix encoding at schema v2.~~ → routed: schema-v2 ContentHash row
~~24. Owner: tick or drop the go-atomic-write draft review.~~ → routed: go-atomic-write draft (docs/drafts/README.md)
~~25. Check whether other fleet repos' tests start hosts bare (mirror the~~ → routed: crush-config (stopAndWait pattern documented in buildflow-dev-gate)
    stopAndWait pattern fix).
~~26. Ratify/adjust the recorded gate load policy (docs/buildflow-dev-gate.md).~~ done — load policy is the standing norm
~~27. Consider a Nix-side checksum pin for vendored static assets (stronger~~ NOT-DO — global exclude proven across green gates; no recurrence
    than the global exclude — integrity, not just skip).
~~28. Update docs/buildflow-dev-gate.md run-ledger when run #6's numbers are~~ done — run #5 cited in the ledger
    next quoted (both #5 and #6 are green; cite either, not "pending").
~~29. Audit remaining `time.Sleep` in tests for the cancellation pattern~~ → routed: recording timeout-test row
    (§f.14 residue; WAV test is the template).
~~30. Close the session loop: verify the daemon swept every doc edit from~~ done — tree clean; daemon swept
    this session (currently clean at write time).

## g. Questions I cannot answer myself

~~1. **Host deploy:** which AI DID numbers should `services.telephony.pbx-lens.aiDids`~~ → routed: host deploy row
   carry on the live host(s), and do you want to drive the deploy now, or
   should the next deploy window pick it up? (The option is landed and
   flake-check green; I have no prod values or deploy path.)
~~2. **Release presentation:** v0.1.0 is marked Pre-release on GitHub per~~ resolved — superseded by v0.2.0 (a full, latest release)
   the go-release v0.x doctrine — flip it to a full Latest release, or
   keep the pre-release label until v0.2?
~~3. **Upstream filing order:** file the two BuildFlow issues (oxfmt/oxlint~~ done — BuildFlow#50/#55/#56 filed over the following days
   min.js provider gap + spawn-starvation) now via the usual
   verify-before-filing flow, or batch them into your next BuildFlow
   session where you can review in context?
