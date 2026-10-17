<!-- docs/status/2026-10-12_07-13_SUPERB-plan-execution-status.md
     Status report for the 2026-10-12 SUPERB pareto plan execution wave
     (docs/planning/2026-10-12_03-43). Snapshot at 07:13 CEST: the full
     gate (R1) is still waiting for load < 40 in a background watcher. -->

# Status — SUPERB plan execution (R1–R27), 2026-10-12 07:13 CEST

Baseline: plan committed `fa3d338`; this wave executed it owner-approved
(Berlin, M20.1+M20.2 deliver, polling both surfaces, release go,
`-max-age` forever, annotate-drafts, raw-at-rest doctrine, R24 filing yes).
Tree at `ef0cc28` (daemon sweeps; a SECOND concurrent session is committing
in this repo and the sibling — see §d.7).

## a. Fully done (verified green)

| Task                                  | Evidence                                                                                                                                                                                                           |
| ------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| R2 dashboard timezone → Europe/Berlin | `internal/web/dashboard.go` (`dashboardLocation`, `dayKey`, `startOfDay`); bare dates resolve in-zone (`server.go parseDateParam`); Berlin-midnight boundary test + UTC injectable test green                      |
| R3 PII export honesty                 | `X-PII-Mode: plain\|hash` header on both endpoints (`piiHeaderValue`); coverage matrix + raw-free-text disclosure in FEATURES.md; README disclosure                                                                |
| R6 partial-number search              | `internal/search` digits shadow table (`lens_digits`), 4+ digit runs merge after FTS hits; 5 package tests + web e2e; search hint rewritten (no longer lies)                                                       |
| R7 streaming export                   | `StreamExportCalls` keyset loop (inclusive cursor + seen-set tieback + loud stall error); both endpoints stream per-view, memory at one 5k page; walk/dedup/stall tests                                            |
| R8 M20.1 + M20.2 delivered            | Range picker (`parseRange`, fragment URL carries range), duration histogram (6 inclusive buckets), week-over-week delta; page + query tests                                                                        |
| R9 live auto-refresh                  | `htmx.PolledRegion`: `/frag/dashboard` 30 s, `/frag/staleness` 60 s; htmx 2.0.10 self-hosted at `/static/htmx.min.js` (no CDN); no-JS fallback intact; markup + fragment tests                                     |
| R11 htpasswd SIGHUP reload            | `htpasswdAuth.reload` + `watchSIGHUP`; broken file keeps the running table; `known()` seam tests                                                                                                                   |
| R13 per-IP throttling                 | `perIPThrottler` (FIFO-capped 1024 buckets, shared fallback, X-Forwarded-For deliberately ignored); separation/eviction/fallback tests                                                                             |
| R14 structured logs                   | Poller logs are key-value records; serve passes `slogLog`; poller test capture renders `k=v`                                                                                                                       |
| R15 `/api/ops`                        | `cli.SourceCounters` (latest-pass per-source stats) + authed JSON endpoint, 503 explains itself when no poller; cli + web tests                                                                                    |
| R16 restore drill                     | `internal/backup/restore_drill_test.go`: snapshot → Verify → restore → 2 rows + FTS hits match source; README documents it                                                                                         |
| R17 e2e in CI                         | Named "End-to-end server probe" step in `.github/workflows/ci.yml`                                                                                                                                                 |
| R18 naming closure                    | cdr `shape`→`byColumn`; `CostEnrichment`→`CostOf(rec) (RecordCost, bool)`; `Client.Recording`→`FetchRecording` (interface updated); `ReasonRetention` = permanent non-fix, annotated in the archived naming report |
| R19 DOMAIN refs file-level            | `docs/DOMAIN_LANGUAGE.md`: zero `.go:NN` refs remain                                                                                                                                                               |
| R12 drafts ledger                     | 6 filed drafts carry FILED banners naming issues; go-atomic-write is the one live draft (owner: annotate-and-keep)                                                                                                 |
| R22 DID design note                   | `docs/did-direction-design.md` (mapping rule, plug-point, consequences) + bounded TODO_LIST row                                                                                                                    |
| R24 Dependabot filing                 | **dependabot-core#16538** filed + verified OPEN; evidence = this repo's PR #1 real diff; draft checked in; ledger updated                                                                                          |
| R10 BuildFlow#45 fix                  | `tools/gomod/tool_timeout.go` (`gateTimeoutCode`, classifiers); compileGate re-tags the runner kill; revert path skips it with an honest warning; 3 discrimination tests green; fix commented on the issue         |
| R23 lessons commit                    | crush-config `97f97c4` (sibling-side-first lesson)                                                                                                                                                                 |
| R5 sibling options                    | `pollMaxAge` (null = keep everything, owner) + `recordingTimeout` wired into `modules/telephony/lens.nix` ExecStart; hosts `pbx`/`pbx-prod` list + eval                                                            |
| Vendor hash                           | `vendorHash.nix` → `sha256-Rucg9dLf16858AK48KS5tCvlOu0M05IH8QMGln75vkA=`, verified with a green nix build                                                                                                          |
| Suite                                 | 14/14 test packages green after the last code change (cmd/pbx-lens now tested too)                                                                                                                                 |
| Markdown                              | dprint run over all 19 touched .md files (5 reflowed)                                                                                                                                                              |

## b. Partially done

~~- **R1 clean full gate** — WAITING: the 15-min load refuses to drop below~~ done — run #5 green 75/75 at 11:36 (docs/buildflow-dev-gate.md)
40 (45–90 all hour; a concurrent session shares the machine). A
background watcher polls and will fire
`BUILDFLOW_NO_RESULT_CACHE=1 buildflow --build-mode dev` the moment load
< 40. Every task's own verification (build + focused + full suite) is
green; only the single clean gate sample + its `docs/buildflow-dev-gate.md`
entry (m1.2) are pending.
~~- **R5 sibling completion** — options landed and evals list; `nix flake~~ done — sibling flake check green, pushed e7c0f8b; host re-deploys = the deploy row
  check` in the sibling still fails its pre-commit leg on TWO MORE
unformatted files I did not touch (`tests/file-secrets-host.nix`,
`tests/tls-mode-host.nix`) — formatting whack-a-mole at HEAD (see §d.6).
Hosts are NOT re-deployed (no deploy path run from here).
~~- **R27 polish** — m27.6 (Sparkline aria + week delta), m27.7 (keyset e2e~~ done — m27.6-m27.9 landed (CHANGELOG); residue = the R27.x advisory row
beyond page 1), m27.8 (`purge -json` + test), m27.9 (AGENTS load
doctrine + quarter calendar + ROADMAP recorded decisions) are done;
vulnix re-triage, art-dupl/branching passes, nix-checker cleanup, README
screenshots are NOT (deferred honestly in the plan annotation).
~~- **Timeout-test hardening** — the 1 ms flake is gone (budgets 50 ms /~~ → routed: recording timeout-test row (context-blocking verified)
250 ms sleep, repro'd and fixed under load 106); the injected-deadline
refactor remains as the TODO_LIST row.

## c. Not started

~~- **R4 v0.1.0 release** — owner said YES; plan-gated on R1's green gate;~~ done — tagged 2026-10-12, on origin
tag + GitHub release + install-doc check intentionally not done before
the gate.
~~- **R20** Telnyx live DTO verify (BLOCKED: no API key).~~ → routed: Telnyx live-verify row (TODO_LIST)
~~- **R25** SegmentBar swap (BLOCKED: upstream release).~~ → routed: ROADMAP bump-watch re-eval
~~- **R26** ContentHash v2 (BLOCKED: v2 gate; SCHEDULED).~~ → routed: schema-v2 row (TODO_LIST)
~~- DID-direction implementation (design note only; implementation row in~~ done — shipped 2026-10-12 (-ai-did + DirectionResolver; CHANGELOG)
TODO_LIST).
~~- Nothing pushed: pbx-lens is ahead of origin (daemon commits + this~~ done — v0.1.0 + v0.2.0 tags and main pushed
wave); the sibling and crush-config have unpushed local commits too.

## d. Totally fucked up (honest list)

1. **Keyset export needed three tries**: my first StreamExportCalls draft
   had a wrong cursor direction (would re-fetch whole pages) and a
   `clear(seen)` that wiped tie-dedup across page boundaries — my own
   5-call/3-page test caught both BEFORE landing, but the near-miss shipped
   design was wrong twice. The stall detector also keeps a documented
   false-positive edge (a timestamp group spanning exactly page multiples
   errors instead of completing — physically impossible at nanosecond
   scale, and loud > silent).
2. **Dead assertion in ops_test.go**: `!lastPass.IsZero() == false` could
   never fire (inverted logic); caught it on re-read and fixed. Tests that
   cannot fail are documentation.
3. **A malformed `question` tool call** wasted a round trip on the owner
   batch (stray object without a type field).
4. **Stale LSP diagnostics** flagged `restore_drill_test.go: timePtr` for
   the rest of the session after the fix landed on disk — burned repeated
   verification cycles re-checking a file that `go test` had already
   proven green.
5. **The docs' dprint command lies**: `nix develop -c dprint fmt` fails
   with `exec: dprint: not found` (dprint is not in the mkShellNoCC
   devShell); had to `nix shell nixpkgs#dprint` instead. The documented
   command in AGENTS/summary should be fixed.
6. **Sibling pre-commit gate is a whack-a-mole**: I nixfmt-fixed
   `lens.nix` + `webphone-dynamic-directory.nix`, and the next
   `nix flake check` fails on `tests/file-secrets-host.nix` +
   `tests/tls-mode-host.nix`. The fleet formatting discipline is not
   holding at HEAD; the gate keeps catching what commits keep missing.
7. **Concurrent session interference, both repos**: another active session
   interleaves commits here (ci.yml cqrs-lint casing, cdr parser
   changelog, the 5-file daemon sweeps) and in the sibling (webphone VM
   work, CI posture docs). My sibling formatting "commit" landed as
   "nothing to commit — daemon swept it first". No data loss, but the
   docs-health lesson #8 (coordinate the parallel worktree editor) keeps
   re-earning its place, and tagging v0.1.0 mid-flight has a real race.
8. **I violated my own PIPESTATUS lesson minutes after committing it**:
   piped the sibling `nix flake check` through grep/echo and read a green
   exit while nix had failed. The lesson exists because this exact shape
   keeps firing — including at me.

## e. Improvements (net effect of the wave)

- The export path is strictly better: bounded memory, no 100k silent cap,
  tie-safe pagination, loud stall; the dashboard rides the same stream.
- The product's trust lies are gone: "Calls today" means today in Berlin,
  partial numbers search, hashed exports announce their mode and their
  limits, the staleness banner re-checks itself.
- Ops: per-IP brute-force throttling + hot credential rotation + a real
  counters endpoint + greppable structured logs.
- Fleet: BuildFlow no longer blames dependency bumps for load starvation
  (and does not destroy trees doing it); dependabot-core has a verified
  fresh repro (#16538).
- The PII posture is explicit doctrine (raw-at-rest, export-gated) instead
  of an implicit accident.

## f. Next tasks (ranked, ≤50)

~~1. Run R1's full gate when load < 40; record the green sample in~~ done — run #5 green 75/75 at 11:36; m1.2 entry recorded (buildflow-dev-gate)
`docs/buildflow-dev-gate.md` (m1.2); retire the "pending" wording.
~~2. Fix the documented dprint command (`nix shell nixpkgs#dprint` or add~~ done — dprint joined the devShell (07-56 a4)
dprint to the devShell).
~~3. If gate lint fires on `dashboardLocation`/`durationBuckets`~~ done — lint zero: nolint:gochecknoglobals with reasons (10-33)
(gochecknoglobals): add justified nolint or refactor behind constructors.
~~4. Verify CI green after push: templ regen + golangci v2.14 + tidy-diff~~ done — CI green on the pushed tree (12-23 a5)
with the promoted htmx require.
~~5. R4: cut annotated tag v0.1.0, GitHub release from CHANGELOG, verify~~ done — v0.1.0 released end-to-end (12-23 a5)
`go install`/nix install paths read true.
~~6. Coordinate with the concurrent session BEFORE tagging (release race).~~ done — release executed clean (12:23 session)
~~7. Push pbx-lens main (daemon + wave commits).~~ done — main + tags pushed (v0.1.0/v0.2.0 on origin)
~~8. Sibling: nixfmt the two straggler test files (or repo-wide `nix fmt`)~~ done — sibling nix flake check ALL CHECKS PASSED (12-23 a7)
and get `nix flake check` fully green.
~~9. Sibling: push; host re-deploys (pbx, pbx-prod) with the new module~~ done-in-part — sibling pushed; host re-deploys = the deploy row (TODO_LIST)
options; update sibling docs for `pollMaxAge`/`recordingTimeout`.
~~10. DID-direction implementation: mapper `DirectionResolver` + `-ai-did`~~ done — DID-direction shipped 2026-10-12 (-ai-did; CHANGELOG)
flag + fixtures (design: docs/did-direction-design.md).
~~11. Sibling: `-ai-did` module option when 10 lands.~~ done — aiDids option landed (sibling e7c0f8b)
~~12. Timeout-test hardening: injected deadlines instead of budget+sleep.~~ → routed: recording timeout-test row (TODO_LIST)
~~13. `cmd/pbx-lens/ingest.go` extraction (shared open→gate→drain→rebuild).~~ done — 83fe566 extraction
~~14. R27.1 vulnix re-triage after the next bump.~~ → routed: R27.x advisory row
~~15. R27.2 art-dupl pass (commands.go mixins, Observation/CallView overlap).~~ → routed: R27.x advisory row
~~16. R27.3 branching-flow pass (fold genuinely complex functions only).~~ → routed: R27.x advisory row
~~17. R27.4 nix-checker hash-bookkeeping cleanup.~~ → routed: R27.x advisory row
~~18. R27.5 README screenshots (four surfaces, auth-safe fixture).~~ done — docs/assets convention + README screenshots (09eb780)
~~19. Watch dependabot-core#16538; answer maintainer questions with the PR#1~~ → routed: dependabot-core#16538 watch row
evidence.
~~20. Watch BuildFlow#45; close it referencing the fix commits once CI~~ done — BuildFlow#45 fixed v0.8.0 (issue closable with the fix)
confirms.
~~21. Owner: tick or drop the go-atomic-write draft's review box.~~ → routed: go-atomic-write draft stays owner-gated (docs/drafts/README.md)
~~22. Telnyx quarter calendar: live DTO re-verify window opens 2027-01 (R20~~ → routed: Telnyx quarterly row
when a key exists).
~~23. Keyset stall UX: surface `export_cursor_stalled` in the export UI~~ done — export honesty note visible on the list page (ba6ec8f)
(currently log-only).
~~24. Consider range-picker presets (7/30-day buttons).~~ done — 7/30-day window presets shipped
~~25. Dashboard cost panel (theme-1 leftover).~~ → routed: ROADMAP theme 1 cost panel
~~26. Search affordance: digits-only search hint chip in the UI.~~ done-superseded — R6 hint copy tells the digits truth (no chip needed)
~~27. SSE upgrade path if poll volume ever matters (R9 note).~~ → routed: ROADMAP (R9 SSE note)
~~28. Staleness fragment: carry filters if the banner ever shows filtered.~~ NOT-DO — banner renders on the unfiltered first page only, by design
~~29. Export memory: a budget test pinning `exportBatchSize` behavior.~~ done — TestExportMemoryBudgetStreamsPastOldCap (ba6ec8f)
~~30. FEATURES evidence refs: keep new rows file-level (line refs rot).~~ NOT-DO — line refs retained deliberately; audit passes re-verify them
~~31. AGENTS.md line budget check (≈205/220) after next edit.~~ done — gate-enforced budget; AGENTS.md held at 219
~~32. TODO_LIST: flip the release + sibling rows DONE after 5/9 land.~~ done — rows evolved (now the deploy row)
~~33. CHANGELOG: fold the release under the version heading when tagging.~~ done — CHANGELOG v0.1.0 heading
~~34. Plan file: re-annotate post-gate/release state (ANNOTATE, never~~ done — plans annotated (12-47/13-32/17-31 EXECUTED); this sweep does the rest
rewrite).
~~35. `nix run . -- version` re-verify after the tag.~~ done — verified (v0.1.0 + v0.2.0 receipts)
~~36. Re-tag vendorHash if go.mod moves before the release.~~ done — vendorHash repaired via the sanctioned app (04-45)
~~37. Verify daemon sweep captured the dprint-reflowed files (5 files).~~ done — daemon sweeps verified (07-56 a3)
~~38. Consider `docs/drafts/` README table (state per draft at a glance).~~ done — docs/drafts/README.md table live
~~39. Dependabot config: consider pinning gomod updates until #16538 fixes~~ → routed: dependabot-core#16538 watch row (owner pinning call)
(owner call — the mangling risk is documented).
~~40. Annotate §g answers into TODO_LIST/ROADMAP once decided.~~ done — §g answers resolved by events (R4 + recorded norms)
~~41. BuildFlow: cherry-pick the gate fix into any pinned release branch.~~ done — v0.8.0 released with the fix
~~42. pbx-lens CI: confirm the new e2e step passes on the first push.~~ done — CI green repeatedly since
~~43. Check `go vet`/lint on `internal/web` fragment handlers (fresh code).~~ done — lint zero (10-33)
~~44. Dashboard: extract the repeated `dateValue`/`dayFormat` constants if a~~ NOT-DO — never grew a third usage; lint clean
third usage appears.
~~45. Consider a `charts` sub-package if dashboard.templ grows again.~~ NOT-DO — dashboard.go never grew past the trigger
~~46. `internal/cli` poller: consider counters for enrichment stats too.~~ NOT-DO — no consumer ask; /api/ops covers ingest + cache
~~47. `/api/ops`: add lastPass age warning in the payload when stale.~~ done — last_pass_age_seconds + stale verdict (v0.2.0)
~~48. Backup drill: wire it into CI as a named step like the e2e probe.~~ done — named "Backup restore drill" CI step
~~49. README: document the polling surfaces + `/api/ops` in the serve~~ done — README operational-surfaces block
section.
~~50. Post-release: harvest this report's §f into TODO_LIST (docs-health~~ done — harvested (sixth-ninth passes); archived by this sweep
HARVEST) and archive this report when superseded.

## g. Questions I cannot answer myself

~~1. **The release gate vs the machine**: load has not dropped below 40 all~~ resolved — R4 executed on the standing approval (v0.1.0 tagged)
hour (45–90, concurrent session). Options: (a) keep waiting for the
quiet window (watcher armed), (b) run the gate NOW at over-budget load
and record it as such, (c) postpone the release until the machine is
genuinely quiet. Which do you want for v0.1.0?
~~2. **The concurrent session** (its commits: ci.yml cqrs-lint casing, cdr~~ resolved — concurrency norms recorded (ROADMAP open question)
parser changelog here; webphone VM work in the sibling) — is it yours
and safe to race, or should I hold the tag + pushes until it lands?
~~3. **Sibling formatting**: want a single repo-wide `nix fmt` commit to end~~ done — sibling green (flake check passed; pushed)
the whack-a-mole (touches files another session may be editing), or
per-file fixes only as the check names them?
