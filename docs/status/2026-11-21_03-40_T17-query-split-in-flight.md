<!-- docs/status/2026-11-21_03-40_T17-query-split-in-flight.md — point-in-time;
     TODO_LIST.md stays the living source. -->

# Status — T17 query split, in flight (2026-11-21, 03:40 CET)

**Session scope:** the 2026-11-21 handoff session — TODO_LIST harvest to the
wave's end state, T18 serve composition move, then the T17 query split
(in flight). Four unpushed commits on top of `a82a283`; **HEAD `211a302`
is `[red]`-tagged by design** (the test lane is mid-migration) and NOT
pushed. Prod build is GREEN at HEAD and in the worktree.

**Gates at report time:** `go build ./...` ✓ · `go test ./internal/serve/
./cmd/pbx-lens/` ✓ (T18's suites) · golangci-lint on serve/cmd 0 issues ✓ ·
cqrs-lint exit 0 ✓ (after the flake FOD repair) · dedup gate 0 groups ✓ ·
docs counts + links ✓ (at the harvest commit; re-run pending) · AGENTS
budget 219/219 ✓ · `nix build` ✓ (T18 receipt) · `go vet
./internal/query/ ./internal/web/` ✗ — TWO known test-lane errors left
(below). NOT VERIFIED this session: full `go test ./...` (blocked by the
two vet errors), nix build after T17 (flake untouched by T17), CI (nothing
pushed).

---

## a) FULLY DONE (verified, receipts)

1. **TODO_LIST.md now reflects the wave's true end state** (commit
   `bba74db`): six landed rows deleted after code verification (go.mod
   pins tc v1.27.2 / httputil v1.9.0; the ingest-edge import guard; the
   CI govulncheck pin; `NoRegistration`/`AccentColor` in
   internal/identity/identity.go:244,247), the five stale 🟢 DONE rows
   from 2026-11-12 removed per the docs-health model, T17/T18/
   identity-export routed as Unblocked openers, FM#11 + PC-2 + filings
   rows rewritten, the T26 decision pack added as a Blocked row, and the
   2026-11-20 dependency-hygiene drafts got their docs/drafts/README.md
   row. Docs gates green at that commit (42 live reports, 94 links, 0
   broken).
2. **Handoff report written** — the harvest is documented in
   `docs/status/2026-11-21_02-28_handoff-harvest-and-fresh-session-openers.md`
   (live table updated).
3. **T18 COMPLETE — the serve composition moved out of the binary**
   (commit `46f31bf`): `internal/serve` owns the stack — flag surface
   (`serve.Flags`/`RegisterFlags`), identity gate, middleware chain
   (`buildServiceConfig` now takes the version as a parameter), poller
   wiring, signal-owned shutdown behind `serve.Run` returning errors —
   and the cmd verb is a 13-line shell (`serveCmd`). The F51 dedup
   landed: the eight serve suites drive ONE tuned constructor
   (`serveTestStack(tb, stackTune{version, authMW, identity, options,
   seed})`) instead of four bespoke assemblies. Receipts: serve package
   tests 10.7s→0.7s pass, cmd tests pass, full `go test ./...` green,
   golangci-lint 0 on both packages, `nix build` green (binary prints
   0.4.0), AGENTS.md bench path updated (budget still 219/219).
4. **cqrs-lint's flake FODs repaired** (in `46f31bf`): the source hash sat
   at the all-zeros placeholder and its own vendorHash had drifted — both
   filled with their surfaced `got` values; `nix develop -c cqrs-lint .`
   is green again (rc=0, INFO-only findings covered by the policy
   ledger). Complementary to the concurrent session's `vendorHash.nix`
   restore (`0b83c4f`, different FOD).
5. **T17 prod move COMPLETE at HEAD** (`211a302`, amended twice while
   unpushed): queries.go, dashboard.go, quality.go, segments.go,
   anomaly.go, coverage.go, representative.go and the Freshness payload
   type moved to `internal/query` with exported view models; the
   transcript-tail signal moved as `query.TranscriptTail`/
   `TranscriptTailThreshold`; `coverageStaleAfter`→`CoverageStaleAfter`,
   `dashboardWindowDays`→`DashboardWindowDays`, `dayFormat`→`DayFormat`,
   `dashboardLocation`→`DashboardLocation`, `percentScale`→`PercentScale`
   exported as the web-facing seam; HTTP request parsing (parseFilters,
   parseRange, parseDateParam, parseWindowParam) stayed in web per the
   ratified interface surgery. All four .templ callers + all eight
   non-test web callers qualified through `query.`; templ regenerated;
   **prod build green, tree-syntax clean (gofmt -e scans pass)**.

## b) PARTIALLY DONE

1. **T17's test lane** — the in-flight half. Three hybrid/pure test files
   moved to `internal/query` (anomaly_test, the query half of quality,
   representative_test minus its page test), a pure trend test extracted
   (`quality_trend_test.go`), the mergeCall + siblingKinds unit tests
   moved out of detail_join_test.go into `query/merge_call_test.go`, and
   the page-collapse render test moved back to
   `web/collapse_render_test.go`. The shared fixtures consolidated into
   **`internal/query/querytest`** (CallStarted, Observation, System,
   AICallPair — the identitytest pattern) with thin delegates in both
   suites. REMAINING (enumerated, ~30–60min): give `query/merge_call_test.go`
   its fixture delegates (`aiAgentTranscript`/`aiAgentCDR` → querytest),
   and `web/export_consumer_test.go` needs `exportBatchSize` (export it
   as `ExportBatchSize` or shrink the test through the public seam) —
   then vet, the full `go test ./...`, and the [red] tag can come off.
2. The identity-export verb and the ScanPage spike: NOT STARTED (the
   openers queue in TODO_LIST).

## c) NOT STARTED

1. Post-move structural re-measure (the FM#11 close-out proof).
2. Everything behind owner gates (decision pack, TLS, coverage call) —
   untouched, as gated.

## d) TOTALLY FUCKED UP (honest)

1. **My own tooling corrupted files for ~40 minutes.** The string-skip
   regex in my qualification script (`"[^"]*query\.[A-Z][^"\n]*"` — the
   FIRST version lacked the `\n` exclusion) stripped `query.` prefixes
   ACROSS LINES between two quotes on different lines, producing
   `const query.staleAfter`, bare field assignments (`Direction = dir`),
   and the whole "files keep reverting" mystery. I burned real time on a
   concurrent-session theory (there ARE eight other crush sessions live,
   and one DID sweep untracked files — `git clean` took
   query/freshness.go once, recreated) before the in-process retry loop
   proved the corruption was deterministic and mine. Lesson: a "mystery
   revert" that reproduces inside one process is your own regex.
2. **One corrupted commit shipped into HEAD** — server.go's stripped
   state rode an amend; recovered by restoring server.go from `46f31bf`
   and re-qualifying with per-local renames (the two `query` locals in
   parseRange/detail renamed `vals`/`q`). Current HEAD server.go is
   valid (gofmt-parses, build green).
3. **The [red] commit exists by necessity, not planning**: the amend-
   fast-to-beat-the-sweeper flow committed before tests could run. The
   tag is honest; the test lane closure is one bounded step away.
4. **quality_test.go was split, not moved** — first moved whole to
   query, then its HTTP half had to come back to web. Wasted one
   round-trip by not classifying the file before moving it.

## e) WHAT WE SHOULD IMPROVE

1. **Mechanical package-moves deserve a written plan first** (the wave
   report said exactly this in §e9): symbol list, stay/move test
   classification, fixture ownership BEFORE the first mv. The T18 move
   (one file + tests, one constructor) went clean in one pass; T17
   (eight files, hybrids, unexported-field surgery) paid for being
   improvised.
2. **Classify hybrid test files by their HELPERS, not their subject** —
   getBody/newTestServer usage is the web marker; direct query-struct
   usage is the query marker.
3. **Export-the-seam decisions early**: DashboardStats's unexported
   fields (rangeFrom/rangeTo) and todayHref surfaced only at templ
   compile time — a pre-move grep for `stats.[a-z]` in templ files would
   have listed the exports on paper first.
4. The concurrency claims ritual works: the roster + claims were written
   before the contested files, and one untracked-file sweep was absorbed
   without loss (rule 7's re-stat saved the recreated file).

## f) UP TO 15 THINGS TO DO NEXT (ranked)

1. Close the T17 test lane: querytest fixture delegates for
   merge_call_test; exportBatchSize decision (export vs seam test);
   full `go test ./...` + `go vet`; THEN re-commit untagged (green) and
   push.
2. golangci-lint + cqrs-lint + dedup gate + docs gates on the T17 tree;
   dprint fmt.
3. CHANGELOG: T17 entry (the current one covers T18 + flake repair).
4. TODO_LIST: delete the T17 row as landed, update FM#11 to "moves
   landed, re-measure open".
5. `pbx-lens identity-export` CLI verb (the S-effort opener).
6. Push the four commits (batched, CI-watch).
7. Post-move structural re-measure (FM#11 payoff proof) — needs both
   moves in one tree.
8. ScanPage spike (branch-only, by disposition).
9. Refresh the 02-28 handoff report's opener list to match reality.
10. Watch for the concurrent sessions' next sweep on internal/query
    (now tracked — sweeps become harmless).

## g) THREE QUESTIONS I CANNOT ANSWER MYSELF

1. **Push cadence for the [red]-tagged head**: push `211a302`+the green
   closure as one train once tests pass (my default), or do you want
   the T17 test-lane commit reviewed first?
2. **exportBatchSize**: export the 5k-row export batch const as
   `query.ExportBatchSize` (the stall-boundary tests shrink it), or keep
   it unexported and move its two test consumers into internal/query?
   I lean export — it is a documented contract surface (5k-row pages).
3. **The remaining opener order**: identity-export verb before or after
   the ScanPage spike (both unblocked; the verb is S-effort and shippable,
   the spike is branch-only exploration).

---

_Session receipt: 4 commits `a82a283..211a302` UNPUSHED (harvest, T18,
flakes+T17-prod `[red]`); prod build green; test lane 2 enumerated
errors from green; eight concurrent crush sessions observed, one sweep
absorbed via the claims ritual._
