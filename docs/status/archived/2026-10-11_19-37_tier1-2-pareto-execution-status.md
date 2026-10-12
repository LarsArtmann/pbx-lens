<!-- docs/status/2026-10-11_19-37_tier1-2-pareto-execution-status.md
     Status report for the 2026-10-11 ~19:10-19:37 dispatch: execute pareto
     tier 1 (P1+P2) plus tiers 2-3 (P3-P8, P11) from
     docs/planning/2026-10-11_19-05_SUPERB-pareto-todo-plan.md.
     Format: .md per this repo's recorded dispatch-report convention
     (2026-09-19, 2026-10-11_18-57); the status-report skill's HTML default
     yields to it. All numbers re-verified live at 19:37 CEST. -->

# Status — Pareto Tier 1-3 Execution (P1, P2, P3-P8, P11)

Date: 2026-10-11 19:37 CEST · Baseline at dispatch start: plan committed
(4ff5541), tree clean, 117 test functions green, gate's only ERROR =
AGENTS.md 478 > 220 lines.

## a) FULLY DONE

| Item                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                              | Evidence                                                                                                                                                                                                                                                             |
| ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **P1 — AGENTS.md shrink 478 → 176 lines** (≤220): extracted Gotchas, Telnyx live-API cluster, cqrs-lint policy, buildflow gate policy, upstream filings verbatim into 5 new `docs/` pages; AGENTS.md keeps stack/domain/commands + a Deep-context pointer section; fixed the E003 pointer to the new doc path                                                                                                                                                                                                     | `wc -l AGENTS.md` = 176; `buildflow -s go-structure-linter` = 0 errors, 1 documented warning (assets/); `docs/gotchas.md` (127), `docs/telnyx-live-api.md` (72), `docs/cqrs-lint-policy.md` (71), `docs/upstream-filings.md` (17), `docs/buildflow-dev-gate.md` (66) |
| **P2 — retention.Policy.Admits ghost system wired into the poller**: `PollConfig.MaxAge` + `applyAgeCutoff`/`ingestAdmitted` in `internal/cli/poller.go`; over-age rows rejected with `retention:age-cutoff` (new constant in retention); Telnyx enrichment seeds dropped in tandem via admitted StreamKey set; `serve -max-age` flag (0 = keep everything) in main.go                                                                                                                                            | `TestPollOnceAgeCutoffRejectsOverAgeObservations` green; LIVE e2e: serve with `-cdr-file -poll-every 1s -max-age 4h` logged "2 offered, 0 recorded, 2 rejected", `/api/freshness` = `calls_indexed:0`; FEATURES.md + AGENTS.md rows added                            |
| **P3 — docs closure**: CHANGELOG entries (retention gate, four-lens session, usage/purge, typed Hit, padCost, excerpt ceiling, StreamMarker purge, AGENTS.md extraction + 4 owner decisions); FEATURES.md ingest-side age-gate row + serve flags; 18-30 plan report's 10 task badges "Open" → "Done (2026-10-11)"                                                                                                                                                                                                 | grep: 0 `>Open<` left in the report; CHANGELOG under correct Added/Fixed sections                                                                                                                                                                                    |
| **P4 — 4 regression pins**: `padCost` reads loaded cost (incl. cost-12 + fallback subtests); `backup.Verify` REJECTS valid-but-empty sqlite (behavior strengthened: new no-tables schema check in backup.go — an empty db previously PASSED integrity_check); CDR 18-col template + appended MOS; `ingest.EachLine` direct (trims/blanks/64 KiB growth/1 MiB ceiling)                                                                                                                                             | `TestPadCostReadsLoadedEntriesCost`, `TestVerifyRejectsValidButEmptySqlite`, `TestParseFallbackTemplateWithAppendedMOS`, 3×EachLine + buffer-const test — all green                                                                                                  |
| **P5 — web robustness**: freshness mid-write failure now log-and-abort (was http.Error after body started); `Options.Log` wired from serve; `search.MaxLimit = 200` clamp on `Query`; route-table test over all 9 registered routes (catches ServeMux-default 404 rot, not resource 404s)                                                                                                                                                                                                                         | `TestEveryRegisteredRouteAnswers`, `TestFreshnessMidWriteFailureLogsInsteadOfCorrupting` (asserts exactly 1 write attempt) green                                                                                                                                     |
| **P6 — backup durability**: staging file fsynced before the atomic rename; package doc states the fsync+Verify crash post-condition                                                                                                                                                                                                                                                                                                                                                                               | `internal/backup` tests green; `syncFile` helper                                                                                                                                                                                                                     |
| **P7 — naming closures**: twin `excerpt()` twins (telnyx + agent) unified into domain `calls.Excerpt(turns, maxRunes)` with decision documented in godoc (they had begun to drift); `searchResultLimit` duplicate deleted — web passes `search.DefaultLimit`; `exportViews` reads `pii` once                                                                                                                                                                                                                      | telnyx ExcerptMax test knob preserved (`calls.Excerpt(turns, opts.ExcerptMax)`); all 4 affected packages green                                                                                                                                                       |
| **P8 — UI e2e + README truth**: live serve probe of `/`, `/dashboard`, `/search?q=…`, `/api/freshness` against a real seeded sqlite — all 200 with expected content; README fixed: stale "web UI is not built yet" claim removed, purge + backup usage sections added, export API + `?pii=hash` + `-max-age` documented                                                                                                                                                                                           | probe output: 4×ok; README now lists all 9 real subcommands' workflows                                                                                                                                                                                               |
| **P11 — polish batch**: recordings-dir help text now names all 3 stem forms (ref, {uuid}_{dest}.wav, bare uuid); dashboard `dayFormat` constant (4 uses, was 3); webhook map-iteration order-independence note (corrected once — first draft claimed "sorted state", truth is min/max + first-wins folds); `TurnRole` typed enum (`TurnRoleUser`/`TurnRoleAssistant`), hash-compatible (%s and JSON render byte-identical strings), ripple closed through telnyx mapper, agent mapper, detail.templ (regenerated) | `rg` shows no bare "user"/"assistant" role literals left in src; full suite green after templ regen                                                                                                                                                                  |

Suite state at 19:37: **build green, 127 test functions green (was 117; +10
new), gofmt clean, go vet clean, templ regenerated.**

## b) PARTIALLY DONE

| Item                            | Works                                         | Open                                                                                                                                                                                                                             | Effort |
| ------------------------------- | --------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ |
| ~~**P1.6 full gate re-run**~~ | ~~Targeted `go-structure-linter` step: 0 errors~~ | ~~The full 73-step `buildflow --build-mode dev` gate has NOT been re-run this session~~ done — full gate green 23:30 (23:41 report) and twice more on 2026-10-12 | S      |
| ~~**TODO_LIST.md harvest**~~    | ~~FEATURES/CHANGELOG/README updated~~         | ~~TODO_LIST rows for P1-P8 still say TODO; pareto plan file's own status column not annotated~~ done — rows flipped in the 20:26 dispatch; plan annotated by P3 + this pass | S      |
~~| **Tier 3 rest (P12, P16)**      | —                                             | P12 (sibling draft annotation) and P16 (RecordingFetcher) untouched                                                                                                                                                              | S each |~~ done — P12: module-draft.nix header annotated SUPERSEDED; P16: landed (20:26 typed boundary models)
| ~~**AGENTS.md wording precision**~~ | ~~Core rewritten~~                            | ~~The Structure bullet still says "every source mapper caps with it"~~ done — rewritten in the 20:26 docs closure (mappers call `calls.Excerpt`) | S      |

## c) NOT STARTED

| Item                                                                                     | Why                                                                                      | Still wanted?                                                                          |
| ---------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------- |
~~| **P9 — upstream filings** (go-cqrs-lite#62/#63/#64 exist? verify, plus treefmt templ go) | External repos; needs verify-before-filing + github-voice flow and Lars's GitHub session | Yes — drafts exist in docs/drafts/, upstream state tracked in docs/upstream-filings.md |~~ done — filed (11:48): #62/#63/#64, go-nix-helpers#7, treefmt-nix#545; state in docs/upstream-filings.md
~~| **P10 — rebuild BuildFlow binary** (5aaf336 → HEAD)                                      | Single atomic external action; license-check skew note stays valid until done            | Yes                                                                                    |~~ done — binary rebuilt 2026-10-11; license-check green 22:31
~~| **P13 — Telnyx live DTO verify**                                                         | BLOCKED: needs real API key                                                              | Yes, quarterly cluster                                                                 |~~ **→ TODO_LIST blocked row (needs a real API key)**
~~| **P14 — ContentHash v2 encoding**                                                        | BLOCKED: owner SCHEDULED at v2                                                           | Yes                                                                                    |~~ **→ TODO_LIST blocked row (schema v2 gate)**
~~| **P15 — proxy timeout knob**                                                             | BLOCKED: owner DEFERRED until traffic data                                               | Yes                                                                                    |~~ done — landed 2026-10-12 (web.DefaultUpstreamTimeout); production value still deferred
~~| **P17 — SegmentBar swap**                                                                | BLOCKED: unreleased upstream templ-components                                            | Yes                                                                                    |~~ **→ TODO_LIST blocked row (unreleased upstream)**
| **UI e2e probe as a checked-in artifact**                                                | Ran ad-hoc from /tmp, deleted after                                                      | Yes — see (e)                                                                          |
| **cross-project lessons.md entry** ("verify-compile after every edit batch")             | Belongs in crush-config repo by commit                                                   | Yes                                                                                    |

## d) TOTALLY FUCKED UP

Nothing is currently broken — build, 13/13 test packages, gofmt, vet are
green — but the honest list of what went wrong **during this dispatch**:

1. **Three self-inflicted compile-broken intervals.** (a) backup_test.go
   appended without the `strings` import; (b) failingWriter test referenced
   `request` I had not defined; (c) the `dayFormat` multiedit landed its
   comment WITHOUT the constant — undefined for the next build. Root cause:
   appending test code via heredoc batches without compiling between.
   Mitigation applied: final build+test+gofmt+vet pass is green; the fix is
   behavioral, not code (see e1).
2. **The dispatch interrupt arrived mid-ripple.** When the status-report
   demand landed, the TurnRole type change was applied to domain.go,
   telnyx, agent, and detail.templ — with NO build/test run and NO templ
   regeneration. The tree did not compile at interrupt time. I finished
   verification before writing this report rather than publish an
   unverified green claim.
3. **I wrote a factually wrong code comment and almost shipped it.** The
   webhook order-independence note first claimed ContentHash "is built
   from sorted state" — false; the aggregation is min/max timestamps +
   first-value folds. Caught in self-review, corrected to the truth.
4. **First e2e probe expectation was wrong.** I asserted the list page
   would render "John Doe" — the domain has no caller-name field; the CSV
   display name is discarded by design. The probe failure was my model of
   the UI being wrong, not the UI.
5. **P2.1's first draft recomputed the age policy per observation with a
   drifting `time.Now()`** for the Telnyx seed filter — boundary rows could
   flip between the batch cutoff and the per-row check. Caught before
   commit; keys now derive from the single admitted set.
6. **P8's verification is not repeatable.** The live probe was a throwaway
   /tmp program, deleted after the run. The evidence exists (output above)
   but cannot be re-run by the next session.

## e) WHAT WE SHOULD IMPROVE

1. **Compile after every edit batch, no exceptions.** Three broken
   intervals, all the same shape. Fix: after any `cat >>`/multiedit batch,
   immediately `go build ./...` before moving on. Candidate lessons.md
   entry (crush-config, by commit).
2. **Verify claims BEFORE writing them into comments/godoc.** The "sorted
   state" comment lie was written-then-corrected; the order should be
   read-then-write. Same class as the published-number correction in the
   18-57 session — this is the second occurrence, so it is a pattern.
3. **Turn live probes into checked-in artifacts.** A `scripts/ui-probe/`
   Go program (or an e2e test that binds a real listener) would make P8's
   run repeatable and CI-able.
4. **`ingestAdmitted` asymmetry.** Three poller sources use the helper;
   pollTelnyx inlines a variant because it must co-filter seeds. Either
   generalize the helper to return the admitted set (and let all four call
   sites share it) or document why telnyx is special. Today it is
   documented, but it is also the only duplicated shape the session
   created.
5. **The plan-vs-daemon commit interleaving.** The auto-commit daemon
   absorbed this session's work as ~10 "chore:" commits, so history does
   not show P1..P11 as logical units. If tier structure matters in
   history, work must be committed deliberately before the daemon's sweep
   (owner-call: see g1).

## f) Top next tasks (harvest ground for TODO_LIST — 30 items)

| #  | Task                                                                                                                                     | Impact | Effort | Category      |
| -- | ---------------------------------------------------------------------------------------------------------------------------------------- | ------ | ------ | ------------- |
| ~~1~~  | ~~Re-run the full buildflow gate; expect ZERO error findings; retire the "only gate ERROR" wording in TODO_LIST/plan~~ done — full gate green 23:30 (23:41 report) and twice on 2026-10-12 | ~~High~~ | ~~S~~ | ~~Quality~~ |
| ~~2~~  | ~~Mark TODO_LIST rows P1-P8 done + annotate the pareto plan's status column~~ done — rows flipped in the 20:26 dispatch; plan annotated by P3 | ~~Medium~~ | ~~S~~ | ~~Documentation~~ |
| ~~3~~  | ~~Commit this batch as structured commits (subject to g1) before further edits~~ **Won't implement — daemon history accepted as-is.** | ~~Medium~~ | ~~S~~ | ~~Cleanup~~ |
| ~~4~~  | ~~Check in the UI e2e probe as `scripts/ui-probe/main.go` (or a listener-binding e2e test) and wire into CI~~ done — landed as TestLiveServerEndToEnd (20:26) | ~~Medium~~ | ~~S~~ | ~~Quality~~ |
| ~~5~~  | ~~Fix AGENTS.md Structure bullet: "mappers cap via calls.Excerpt" wording~~ done — rewritten in the 20:26 docs closure | ~~Low~~ | ~~S~~ | ~~Documentation~~ |
| ~~6~~  | ~~Add `TurnRole` + `calls.Excerpt` to docs/DOMAIN_LANGUAGE.md~~ done — TurnRole + Age-gate rows landed in docs/DOMAIN_LANGUAGE.md | ~~Low~~ | ~~S~~ | ~~Documentation~~ |
| ~~7~~  | ~~P9.1: verify go-cqrs-lite#62/63/64 state at source; post or update with repro evidence~~ done — state re-verified in docs/upstream-filings.md (2026-10-12) | ~~Medium~~ | ~~M~~ | ~~Quality~~ |
| ~~8~~  | ~~P9.4: treefmt-nix templ formatter goPkg — corroborate #545 or file on go-nix-helpers~~ done — go-nix-helpers#7 + treefmt-nix#545 tracked in docs/upstream-filings.md | ~~Low~~ | ~~M~~ | ~~Quality~~ |
| ~~9~~  | ~~P10: rebuild BuildFlow binary from HEAD; re-run gate; delete the stale license-check note in docs/buildflow-dev-gate.md~~ done — rebuilt 2026-10-11; license-check green | ~~Low~~ | ~~S~~ | ~~Cleanup~~ |
| ~~10~~ | ~~P12: diff deploy/nixos/module-draft.nix vs sibling lens.nix; annotate the draft as a historical design record in its header~~ done — header annotated SUPERSEDED | ~~Low~~ | ~~S~~ | ~~Cleanup~~ |
~~| 11 | P16: `RecordingFetcher` consumer-side interface + swap `RecordingOptions.Telnyx`                                                         | Low    | S      | Feature       |~~ done — landed (20:26 typed boundary models)
| ~~12~~ | ~~Generalize `ingestAdmitted`/seed filtering so all four poller sources share one admission path~~ done — unified as AgeGate/GateIngest (20:26) | ~~Low~~ | ~~S~~ | ~~Cleanup~~ |
| ~~13~~ | ~~Decide + implement whether `ingest-*` one-shot commands gain `-max-age` (currently deliberately ungated; live e2e exposed the asymmetry)~~ done — landed (20:26 AgeGate on all four commands) | ~~Medium~~ | ~~S~~ | ~~Feature~~ |
| 14 | Sibling repo: set the production `-max-age` window in `services.telephony.pbx-lens` (needs g3) **→ TODO_LIST blocked row (owner picks the window)** | High   | S      | Feature       |
| ~~15~~ | ~~backup: fsync the parent directory after rename for full crash durability (the file fsync alone does not guarantee the rename persists)~~ done — landed (CHANGELOG: parent directory flushed after rename) | ~~Medium~~ | ~~S~~ | ~~Bug~~ |
| ~~16~~ | ~~Add a stale=true freshness test path if missing (freshness tests cover the fresh/empty cases — verify the stale branch has a pin)~~ done — freshness tests cover fresh/stale branches | ~~Low~~ | ~~S~~ | ~~Quality~~ |
| ~~17~~ | ~~Auth: add failed-attempt rate limiting (none today; basic auth brute-force is unthrottled)~~ done — landed (20:26 x/time/rate throttle) | ~~Medium~~ | ~~M~~ | ~~Feature~~ |
| 18 | Auth: htpasswd reload on SIGHUP or mtime watch (restart required today) **→ ROADMAP theme 2** | Low    | M      | Feature       |
| 19 | Export: document/verify streaming behavior near the maxExportRows cap **→ ROADMAP theme 1 (export streaming)** | Low    | S      | Documentation |
| 20 | Poller: surface per-source rejection counters on an ops endpoint (today they live only in log lines) **→ ROADMAP theme 2** | Low    | M      | Feature       |
| 21 | Adopt slog structured logging for the poller/serve logs (std log.Printf today) **→ ROADMAP theme 2** | Low    | M      | Quality       |
| ~~22~~ | ~~CI: add a cqrs-lint lane so the findings policy is enforced off-box too~~ done — landed (20:26 CI job) | ~~Low~~ | ~~S~~ | ~~Quality~~ |
| 23 | Re-run art-dupl after `calls.Excerpt` extraction; revisit the 20-findings advisory with fresh data **→ advisory; documented non-fix class** | Low    | S      | Quality       |
| 24 | README: add UI screenshots of the four surfaces **→ ROADMAP theme 2 (public presence)** | Low    | S      | Documentation |
| 25 | Keyset pagination e2e: probe "Older calls" beyond page 1 with >page-size data **→ ROADMAP theme 1** | Low    | S      | Quality       |
| 26 | templ-components: watch for the next published version beyond v1.21.0 (SegmentBar unblock = P17) **→ TODO_LIST blocked row (unreleased upstream)** | Low    | S      | Feature       |
| 27 | Tag the first release (v0.1.0) via go-release once gate is green (CHANGELOG is release-ready) **→ TODO_LIST blocked row (owner release gate)** | Medium | M      | Feature       |
| ~~28~~ | ~~Search page: e2e-verify the FTS full-token hint renders for a mid-number query~~ done — landed (11:31 search hint + locking test) | ~~Low~~ | ~~S~~ | ~~Quality~~ |
| 29 | Record "verify-compile after every batch" in crush-config references/lessons.md (by commit, not in-session) **→ crush-config repo, by commit** | Low    | S      | Documentation |
| 30 | Quarterly Telnyx live re-verify cluster (P13) — schedule the first run once an API key exists **→ TODO_LIST blocked row (needs a real API key)** | Medium | S      | Quality       |

## g) Top questions I can NOT figure out myself

1. ~~**Commit boundaries:** the auto-commit daemon has absorbed tiers 1-2 as~~ **Won't implement — daemon history accepted as-is.**
   ~~~10 heuristic "chore:" commits. Do you want a deliberate structured~~
   ~~commit (or a few per-task commits) for this batch going forward — and~~
   ~~should I squash/reword the chore commits covering P1-P11, or let~~
   ~~history stay as the daemon wrote it?~~
2. **P9 upstream filings:** post the D6 filings live now (verify-before- **→ owner go; drafts verified OPEN (docs/upstream-filings.md)**
   filing → github-voice flow, using your GitHub auth), or keep them as
   drafts in docs/drafts/ until you've reviewed?
3. **Production retention window:** what `-max-age` should the NixOS **→ TODO_LIST blocked row (owner picks the window)**
   module set for the two hosts (e.g. 720h = 30 days? 2160h = 90?), and
   should the `ingest-*` one-shot commands gain the same gate so CLI
   backfills can't silently re-import over-age rows?
