# Status — corrections pass, HARVEST, full battery, and the /api/ops wire fix (2026-11-21, 15:15 CET)

**Scope.** Verified this session: every gate receipt below was run live
(docs counts, doc links, agents-budget, freshness, dedup, cqrs-lint,
golangci, erraudit, tidy, treefmt, verify-all's five browser suites);
both LOC figures re-derived from git at named commits; the spike census
and golden word-diff run in a disposable worktree; the /api/ops fix
verified at unit level, under `-race` (web+serve), by its browser probe,
and by the full lane. NOT verified this session: CI on the newest pushed
head (run `44912647699` was in flight at writing); the spike branch's
own test suite (its green rests on the prior session's receipt); the
14 newly-harvested rows (routed, not executed).

## a) FULLY DONE (verified, receipts)

1. **TODO_LIST consistency repair** — the false "Unblocked table is now
   EMPTY" header claim removed (the T14-receipts row remains), the
   dangling wave-report fragment repaired, the lockout truth folded in;
   FM#11 moved to Unblocked (🔴 TODO) with the measurement lineage below;
   the ScanPage row moved to Blocked (owner merge decision + the verified
   conflict: `queries.go` diverged ~53 lines post-spike); the Type-design
   ScanPage disposition refreshed. dprint-formatted; gates green.
2. **The measurement lineage settled (both numbers honest).** The 13-05
   report's "4398 → 2508 / 7591 → 7022" is correct AT `6332988`
   (post-T17, pre-DI — its §a6 names the commit; re-derived and
   confirmed). At `80349b3` (post-DI) the same tree measures **4398 →
   2529 / 7591 → 7042** because `f92f7a4`'s batch-size DI added ~21
   lines back. The FM#11 row now carries BOTH commits; the original
   "11989 → 9530 across the three moves" stays wrong as published
   (included tests, wrong attribution) — 13-05 §d4 stands.
3. **cqrs-lint-policy.md refreshed** to the fresh census (still 27: 16
   WARNING / 11 INFO / 0 ERROR): F031 relocated to
   `internal/query/queries.go:232` (the walk's limit is now visible in
   its Scan args; only the probe fires), serve C023 pair at
   `internal/serve/serve.go:544/552`, identity C023 coordinates
   202/220/244/275/276/442, a NEW disposition for the `ops.go:252`
   export-staging discard, a 2026-11-21 census note, and the E023 entry's
   package path updated to `internal/identity`.
4. **docs/gotchas.md** — the `representative_test.go` path fixed to its
   post-T17 home + two new war stories (the templ-fmt/treefmt pre-push
   remedy; "OpenIdentity creates its tables wherever it is pointed" with
   the JSON-lockout corollary).
5. **Annotations via the writer** (`annotate.go`, judge-verified): 02-28
   §c2's dangling `211a302` marked superseded → `3f8098a`; 03-40
   §b/§c/§f/§g got all 17 inline verdicts (now archivable at the next
   ritual); the wave report's landed-but-open §f items 1 (T17), 2 (T18),
   3 (F51), 5 (spike) struck; 13-05 §f struck 21 rows done + §g1 routed.
6. **AGENTS.md at 219/219** (budget gate green): `internal/query` +
   `querytest` + the identity-export one-liner, the pre-push
   `templ fmt` + treefmt + `tidy -diff` command, `-race` on the test
   line, the stale `representative.go` path fixed — with five
   compensating trims.
7. **SECURITY.md** — the stale `internal/web/userauth.go` path fixed to
   `internal/identity`, and the export-truth paragraph added (the JSON
   is an audit RECORD with no credentials; importing it closes the seat;
   snapshot-restore is the only migration).
8. **Decision-doc STATUS block** — `2026-11-20_web-package-policy.md`
   now states all three P1 moves landed with the re-verified delta.
9. **Spike verification owed since the handoff — delivered** (worktree
   at `251a4ad`, branch untouched): templ generated, cqrs-lint rc 0 with
   the SAME 27-finding set (only coordinate shifts from the older base —
   no new rule fires), and BOTH re-baselined goldens word-diffed
   PROSE-ONLY (the `lens.web.export_cursor_stalled` code reference
   dropped, matching the retired stall semantics — that prose change is
   the owner's §g2 call, now visible in full).
10. **The full battery — and what it caught.** `go test -race ./...`
    green; `nix build` green with version parity (`pbx-lens 0.4.0`);
    `nix flake check` 4/5 (pre-commit = the documented templ sandbox
    false-positive, CI treefmt lane green on the pushed head);
    erraudit `lint --max` 0; `go mod tidy -diff` clean; golangci 0
    issues (the LSP's gci/golines flags on `cli_messages.go` were
    noise); cqrs-lint 27/rc 0; dedup 0 groups; treefmt check rc 0
    (captured without pipes); **verify-all initially FAILED at
    verify-ops-verdict** — a real shipped regression (item 11).
11. **The /api/ops wire-shape regression — root-caused, fixed, pinned.**
    The 2026-11-20 PollerHealth deepening swapped the hand-built map
    payload for the typed struct, which fell into `composeOpsPayload`'s
    defensive non-map wrap branch and NESTED every poller key under a
    `"poller"` key — silently breaking the flat R15 contract the repo's
    own test comment promises ("the R15 keys stay where they always
    were"). All Go tests stayed green because their assertions were
    level-blind `strings.Contains` leaf checks; the browser probe caught
    it because it parses `body.stale` at the top level. Fixed in
    `ensureOpsPayloadMap` (typed payloads flatten through their json
    tags, per the PollerHealth doctrine), pinned flat by
    `TestOpsEndpointFlattensTypedPollerPayload`, CHANGELOG 2026-11-21 e,
    a gotchas war story, and the probe re-run: `/api/ops -> 200 stale:
    true` → ALL PASS → the full lane ALL PASS (54 PNGs to
    `/tmp/lens-shots`, outside the tree). web + serve re-raced green.
12. **HARVEST** — 13-05 §f routed per docs-health: 14 bounded Low-impact
    rows + 3 existing-row extensions (identity posture decision (d), the
    watchdog's items 5/6, the coverage attribution note), every row
    citing its source item. Pre-flight first said UNSAFE (my lines would
    have flipped the wave report's landed-but-unstruck T17 item to
    DRIFT); the root fix was striking those landed items, then SAFE.
    Freshness: FRESH, 0 unharvested open §f items.
13. **Hygiene** — LSP restarted (the gci/golines noise died; golangci 0
    is the truth), the /tmp probe dirs + scratch files swept with
    `trash`, the claims roster carries this continuation session's row,
    `.docs-gates-preflight.json` gitignored per the tool's own note.

## b) PARTIALLY DONE

1. **Spike (handoff f7)** — verification done (a9); the PUSH still
   awaits the owner's §g3 say-so; the branch's test-suite green rests on
   the prior session's receipt, not re-run here.
2. **FM#11 close-out** — measurement lineage + decision-doc STATUS done;
   the structural-assessment RE-RUN (the actual payoff) has not run.
3. **flake check** — 4/5: the pre-commit templ sandbox false-positive
   (documented 2026-10-28) still reds every full check; CI's treefmt
   lane is truth.
4. **The /api/ops fix** — fully verified locally (unit + race + probe +
   lane); CI on the exact newest pushed head was still in flight at
   writing (run `44912647699`).
5. **13-05 §f** — 21 struck done; 24 remain open or routed (TODO_LIST
   is the backlog now).

## c) NOT STARTED

1. The **14 newly-harvested Low-impact rows** (CLI What/Why/Fix template
   campaign, query-purity import guard, benchmarks re-run, screenshot
   baseline re-render, deploy-draft backup-unit check, atomic-write
   draft fate, webObservation dedupe check, erraudit enforce-posture,
   DOMAIN_LANGUAGE where-used sweep, `DefaultExportBatchSize` const
   block, identity-export ops-card surfacing, analytics denominators
   spot-check, http-stack read-through, export-while-live on a real
   serve) — all bounded, all cited, none started.
2. The **three owner decisions** (§g below + decision (d) migration
   story on the Identity posture row).
3. The rest of the Blocked table — unchanged by design (GO-2026-6859
   posture, T15 MOS window 2027-01, PII boundary, CSP flip, coverage
   call, pixel strategy, ...).
4. An upstream filing for the pre-commit templ sandbox false-positive —
   a candidate only, never drafted.
5. Investigating the docs-gates quirk: `docs-gates annotate docs/status/`
   exits 1 on 4 archived F/spacing findings despite `annotate_max: 247`
   (pre-2026-11-12 backlog; the findings predate this session and my
   reports are clean) — whether the ratchet applies only inside the
   `run` composition is un-investigated.

## d) TOTALLY FUCKED UP (honest)

1. **I accused a correct number of being wrong.** While repairing the
   FM#11 row I wrote "its 2508/7022 correction does not reproduce" —
   without reading the 13-05 report's own provenance line ("measured at
   `6332988`", §a6). My measurement was at a LATER commit (post-DI), so
   both numbers were honest at their own trees and my refutation was the
   false claim. It sat in the routing doc for ~2 hours until, while
   drafting THIS report's §d, I checked the lineage, verified both
   commits, and corrected the row. Root cause: I verified the OTHER
   session's claim but not my own refutation of it. The
   verify-before-encoding discipline must apply symmetrically.
2. **The LOC chain needed three rounds to converge** (11989 → 9530
   wrong twice → 2508/7022 correct at its commit → 2529/7042 at the
   current head). The remedy is mechanical: every LOC figure carries its
   COMMIT and method, or it is noise.
3. **Pre-flight order inverted.** My first harvest lines would have
   flipped the wave report's landed-but-unstruck T17 item to DRIFT; the
   pre-flight tool caught it, but the right first move was HARVEST rule
   3 (verify against code, strike what landed) — I reached for rewording
   first and annotated the wave report only because the tool forced the
   question. Right outcome, backwards order.
4. **Three rounds of python splice assertions** in TODO_LIST (hardcoded
   line indices, a wrong tail-anchor, and one round where an unrelated
   `printf >> .gitignore` ran past a failed python because the two
   commands were newline- rather than `&&`-chained). All failures were
   assert-before-write — no corruption — but dynamic-anchor lookup
   should have been round one.
5. **Diffed the partial spike census** (pre-templ-generate) instead of
   the post-generation one; self-caught a command later and re-diffed
   correctly, but the first conclusion was derived from incomplete data.
6. **Exit codes masked by pipes** on the first nix build/treefmt pass
   (re-captured rigorously after noticing), and one `job_output` tool
   name typed INTO a bash call where it silently did nothing.
7. **The battery ran at a moving head** — the concurrent session pushed
   `c775eb6` mid-session and the daemon swept docs repeatedly, so "the
   battery at `80349b3`" became "the battery at whatever HEAD is when it
   runs". Also: the full `-race ./...` ran BEFORE my server.go fix; the
   post-fix re-race covered web + serve only, not the whole tree.

## e) WHAT WE SHOULD IMPROVE

1. **Every number carries its commit + method.** The LOC saga converged
   only when provenance became explicit; make it the format, not the
   archaeology.
2. **Verify your own refutation as hard as the claim you refute** (d1) —
   symmetric skepticism, no exceptions for one's own edits.
3. **Pre-flight BEFORE annotating**, not after: the freshness judge
   knows about landed-but-open items the annotator hasn't met yet; run
   it before touching either side.
4. **Wire contracts need LEVEL-pinned assertions** — leaf substrings are
   shape-blind and let a shipped shape change pass every Go test for a
   day. The flatness pin exists now; generalize the habit to the other
   /api surfaces.
5. **Run the FULL lane when the predecessor didn't.** Every fast gate
   was green while a shipped regression sat in /api/ops; only
   verify-all's browser probes parsed the actual JSON shape.
6. **Capture rc without pipes**; tool names never go inside bash.
7. **Dynamic anchors + `&&`-chained mutations** for any line-splice
   work — three assertion rounds was two too many.
8. **The concurrency ritual held**: two concurrent pushes, continuous
   daemon sweeps, a foreign commit mixed into one daemon commit — all
   absorbed via re-stat + worktree isolation without one lost byte. Keep
   re-stating after every idle window.

## f) UP TO 50 THINGS TO GET DONE NEXT (ranked; the bulk already lives in TODO_LIST — this is the delta)

| #   | Task                                                                                                                                                                                                                                                                               | Impact | Effort | Category      |
| --- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ | ------ | ------------- |
| 1   | Owner: decide the spike's fate (§g2) — merge as-is means a `queries.go` re-port onto `streamViews(batchSize)` + accepting the retired stall semantics (the golden prose dropped the `lens.web.export_cursor_stalled` reference); rework keeps a stall-equivalent contract; or park | High   | S      | Decision      |
| 2   | Owner: push cadence (§g3) — the /api/ops fix is ALREADY on origin (rode the daemon train `1eea626`/`b88b2ac`); only the annotation sweep (`8f72e31`) sits unpushed; CI on the newest head in flight (`44912647699`) — confirm green                                                | High   | S      | Decision      |
| 3   | Owner: migration story decision (d) on the Identity posture row — snapshot-restore-only vs the upstream usermgmt import-credentials ask                                                                                                                                            | High   | S      | Decision      |
| 4   | FM#11 payoff: re-run the structural assessment on the halved web (the true lineage is now in the row; the churn analysis is the skill's job)                                                                                                                                       | High   | L      | Quality       |
| 5   | CLI What/Why/Fix campaign: register templates for the remaining `lens.*` codes with render tests per batch (the six identity-export codes are the worked pattern)                                                                                                                  | Medium | M      | Feature       |
| 6   | Query-purity import guard test: the query package must not import net/http (pin the package doc's claim the way web→ingest is guarded)                                                                                                                                             | Low    | S      | Quality       |
| 7   | Re-run BenchmarkConcurrentPoll + BenchmarkIngestCDR after the relocations; refresh docs/http-stack.md if drifted                                                                                                                                                                   | Low    | S      | Quality       |
| 8   | Re-render the screenshot baseline after the export-note honesty prose change (accept or fix the pixel delta); also close §f14's sibling: the /api/ops fix touched no markup, so the delta should be the practiced noise floor                                                      | Low    | S      | Quality       |
| 9   | Confirm the systemd draft backup unit stages identity.db beside index.db (runbook step 7 vs the draft unit) — align or annotate                                                                                                                                                    | Low    | S      | Documentation |
| 10  | Decide the atomic-write draft's fate IF another atomic-publish consumer appears (the export artifact already duplicates the discipline locally)                                                                                                                                    | Low    | S      | Cleanup       |
| 11  | Dedupe web's local `webObservation` against the querytest fixture IF shapes truly match (verify shapes first; dedup gate is 0)                                                                                                                                                     | Low    | S      | Cleanup       |
| 12  | Decide the erraudit `--enforce-coded-errors` posture for the cli_messages constructors (generic-return-exempt by name today)                                                                                                                                                       | Low    | S      | Quality       |
| 13  | Full sweep of docs/DOMAIN_LANGUAGE.md "Where used in code" cells for moved-file staleness (one read-through owed post-T17/T18)                                                                                                                                                     | Low    | S      | Documentation |
| 14  | Decide whether `query.DefaultExportBatchSize` sits beside `PageSize` in one rendering-contract const block                                                                                                                                                                         | Low    | S      | Cleanup       |
| 15  | Surface `identity-export` in /api/ops's identity card docs (the operator's discovery surface)                                                                                                                                                                                      | Low    | S      | Documentation |
| 16  | Spot-check that the query relocation shifted no analytics denominator's home (bench no-user-turn rule + denominators table)                                                                                                                                                        | Low    | S      | Quality       |
| 17  | One read-through of docs/http-stack.md's middleware chain vs internal/serve post-T18 (grep found no stale paths; the prose read is owed)                                                                                                                                           | Low    | S      | Documentation |
| 18  | Re-verify TestIdentityExportWhileAnotherInstanceIsLive against a REAL live serve once (closing the "in-process only" gap the test comment names)                                                                                                                                   | Low    | M      | Quality       |
| 19  | Investigate the docs-gates quirk: `annotate <dir>` exits 1 despite `annotate_max: 247` — does the ratchet apply only in `run` composition? (c5)                                                                                                                                    | Low    | S      | Tooling       |
| 20  | Candidate: draft the upstream filing for the pre-commit templ sandbox false-positive (bitten 2026-10-28 and in every full flake check since; owner go-ahead per the filings convention) (§g3)                                                                                      | Low    | S      | Upstream      |
| 21  | The watchdog additions routed this session: usermgmt v4.20.0+ import/export watch (item 5) + metaengine bump → spike-note re-verify (item 6) — ride the Consolidated upstream watchdog row                                                                                         | Low    | S      | Watch         |
| 22  | Coverage attribution re-measure (per-package shifts post-relocations; rides the next CI coverage run — Coverage row note)                                                                                                                                                          | Low    | S      | Quality       |
| 23+ | Everything else stands in TODO_LIST (Blocked/Low/Watch lanes, 24+ inherited rows incl. GO-2026-6859, T15 MOS window 2027-01, PII, CSP flip, stamp-churn doctrine, the 2026-11-20 owner decision pack) — TODO_LIST is the backlog; this report adds no new tables beyond its own §f | —      | —      | —             |

## g) THREE QUESTIONS I CANNOT ANSWER MYSELF

1. **Push cadence for the fix train** (carrying §g3 forward with new
   facts): the /api/ops wire fix and its pin test are ALREADY on origin
   (they rode the daemon train past my re-stat window), CI was in
   flight at writing, and only the annotation sweep sits unpushed. Do
   you want receipt-grade commits pushed immediately (the concurrency
   doc's ratified doctrine) even when they touch shipped operator
   surfaces — or held for your review, in which case should I treat
   daemon-swept pushes of MY work as a process bug to escalate?
2. **The spike's fate** (§g2, now with full evidence): the branch is
   proven (same lint set, prose-only golden delta, 93 lines retired,
   both stall rejections dropped) but merging conflicts with the DI'd
   `queries.go` and needs a re-port. Merge as-is (accepting the
   contract change the golden prose encodes), rework to keep a
   stall-equivalent rejection, or park it?
3. **The pre-commit templ sandbox false-positive** (new §g3): it has
   red-flashed every full `nix flake check` for a month (documented
   2026-10-28, re-confirmed today on a clean tree). File it upstream —
   and if so, to WHICH project owns the sandboxed templ hook in
   checks.pre-commit (treefmt-nix? go-nix-helpers? your flake module?) —
   or keep absorbing it as documented noise with CI as truth?

---

_Session receipt: origin/main = `dc6e071` (CI run `44912647699` in
flight at writing); local main = `8f72e31` (ahead 1 — the 13-05/wave
annotation sweep); the /api/ops fix rode `1eea626` + `b88b2ac` to
origin; tree clean; spike branch `251a4ad` local-only, untouched; the
only uncommitted file at writing is this report + its README row (the
daemon sweeps them per the ratified doctrine)._
