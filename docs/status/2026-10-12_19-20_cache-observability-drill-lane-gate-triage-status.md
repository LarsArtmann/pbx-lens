# Status 2026-10-12 19:20 — cache observability, drill lane, golden time bombs, gate triage under co-tenant load

Single-session report. Scope: everything this session touched, honestly graded,
plus what a concurrent session did to the same tree while I worked (it mattered:
three gate failures and two rebuild storms trace to the overlap, not to code).

## Self-critique first (what I forgot / did worse / can improve)

- **Forgot:** to check `git log -1` / `git status` before each gate run. The
  auto-commit daemon raced the gate's own repair steps twice (go-mod-normalize
  rewrite → daemon commit → go-generate's bogus "go mod tidy" failure) and I
  burned two full gate runs before writing the triage rule down. The lesson is
  now in docs/buildflow-dev-gate.md; it should have been checked pre-run.
- **Forgot:** to re-read the go-etag API doc before writing test expectations —
  `OnBufferOverflow` documents "the configured limit that was exceeded"; I
  asserted the body size. My test was wrong, the library was right.
- **Did worse:** hit-ratio denominator was wrong in the first cut
  (`generated + notModified`); the endpoint test caught it because I tests-first'd
  the observable semantics — good — but I had modeled the middleware from prose
  instead of reading etag.go's call sites first. Read-the-source-first would
  have produced the right model in one step.
- **Did worse:** debugging the golden failure from the diff instead of isolating
  the baseline immediately. One `git worktree` at the parent commit settled it
  in 30 s (pre-existing failure); I spent several minutes on my own diff first.
- **Sloppy:** my caching_test.go edit duplicated a `httptestRequest` signature
  line and broke package syntax (vet caught it in seconds, fixed immediately) —
  a mechanical error from restructuring an edit block by hand.
- **Can improve:** a gate run on a co-tenanted machine should start with a
  quiescence wait (tree clean + load < threshold) as a precondition, not as a
  post-mortem. And golden suites deserve a "regenerate twice, diff, fail on
  drift" self-check so capture-time nondeterminism can never land.

## a) FULLY DONE

1. **Backup restore drill as a named CI lane** (Medium TODO row): Build & Test
   step after the e2e probe (`.github/workflows/ci.yml`), exact-`-run` command,
   rationale comment. Verified: actionlint exit 0, drill test green with the
   CI command. Row closed in TODO_LIST (ninth pass).
2. **`/api/ops` cache section (R26, Medium TODO row)**: `CacheCounters`/
   `CacheStats` (internal/web/caching.go:13-72), `ETagMiddleware(stats, log)`
   hooks (OnETagGenerated / On304 / OnBufferOverflow with warn log naming the
   limit), `Options.Cache`, `composeOpsPayload` merge that keeps the R15
   top-level shape (`sources`, `last_pass`) and adds `cache` as a sibling;
   503 only when neither counter source is wired. serve wiring passes the one
   shared counter set through `buildServiceConfig` into both the middleware
   and `web.Register`. Empirical discovery: go-etag fires OnETagGenerated on
   304 resolution too, so `ETagGenerated` doubles as the tag-resolved request
   count and `hit_ratio = not_modified / etag_generated` (doc comment records
   this). Rows closed: cache hooks, README ops notes, SECURITY.md posture,
   gotchas templ-flush — five TODO rows total.
3. **Tests for all of the above**: TestETagMiddlewareCounters (200→tag, 304
   replay, ratio 0.5), TestETagMiddlewareOverflowWarns (untagged streaming,
   counter, warn args = configured limit), TestOpsEndpointServesCacheSection
   (cache-only answers 200 — the new contract; both sections share top level;
   R15 keys unchanged). All green under -race.
4. **Two pre-existing golden-suite time bombs fixed** (found because
   TestGoldenPages failed at a pre-change baseline worktree — NOT my
   regression): (a) `page_calls` pinned the staleness-banner state the wall
   clock produced (broke for every run after 17:00 local once the 09:00-UTC
   seed crossed staleAfter); (b) `page_dashboard` pinned bare ISO bucket
   labels that shift at Berlin midnight. Scrubber now reduces the staleness
   region's content to `STALENESS_REGION` (attributes stay pinned; banner
   behavior keeps its handler tests at server_test.go:249-259) and bare dates
   to `DATE` (negative guard spares fixed-seed `datetime="…T…"` attributes).
   Goldens regenerated (`-update`), suite green twice.
5. **Flake repairs**: cqrs-lint `fetchFromGitHub` source hash went stale
   (GitHub tarball bytes changed under the pinned rev) AND its `vendorHash`
   was a byte-for-byte copy of the source hash — a paste bug that had gone
   unnoticed because the source FOD failed first. Fixed with nix-reported
   values; devShell builds, `cqrs-lint .` exits 0. `vendorHash.nix` repaired
   through the sanctioned `nix run .#update-vendor-hash` (verified green).
6. **Doc rows landed**: gotchas.md templ-flush entry (buffered `render` is
   what makes the 304 contract work); SECURITY.md `/metrics` unauthenticated
   opt-in documented as deliberate (appkit mounts it outside the htpasswd
   chain; perimeter = bind/firewall; also fixed the stale "no tagged
   releases" claim on sight); README operational-surfaces block (/version,
   /metrics, health probes, ETag→304 + Server-Timing + no-cache contract,
   corrected /api/ops 503 semantics).
7. **Project docs maintained**: TODO_LIST ninth pass (5 rows closed);
   CHANGELOG Added (cache section, drill lane, docs set) + Fixed (golden time
   bombs); FEATURES /api/ops + restore-drill rows updated; durable lessons
   recorded (buildflow-dev-gate: daemon-vs-gate race triage rule; gotchas:
   goldens must never pin wall-clock-coupled structure).
8. **Verification set (individual)**: full suite 14/14 packages green under
   `-race` AND plain (twice), `go vet` clean, golangci-lint 0 issues on both
   touched packages, `go tool templ generate` green, cqrs-lint green,
   actionlint green, `nix build` green twice, `go build ./...` green.

## b) PARTIALLY DONE

1. **One fully-green single `buildflow --build-mode dev` run**: best attempt
   reached **76 success / 0 failed** in step results with only the `nix-build`
   lane killed (OOM/timeout) — twice — during a concurrent session's load
   bursts (15-min load 40→80). The casualty was verified individually
   (direct `nix build` green on the same tree) per the documented
   load-starvation doctrine, so the STATE is verified; the cosmetically-clean
   single-run gate is not. dprint auto-fixed 3 doc files and nix-checker 2
   files during the attempts (all formatting).
2. **`go.sum` tidy delta**: the gate's module-update left the correct tidy
   pruning (stale `/go.mod` sums after the utils indirect→direct promotion);
   it is the state CI's tidy check demands, but it landed uncommitted and its
   fate belongs to the daemon sweep.
3. **TODO_LIST row inventory vs the concurrent session**: commits `ba6ec8f`
   (export-stall note on the list page) and `83fe566` (ingest.go shared
   sequence extraction) close two backlog rows I did NOT touch; whether their
   rows are already removed from TODO_LIST is unverified by me.

## c) NOT STARTED (this session)

- All remaining Low-impact TODO rows except the three doc ones I closed:
  coverage pass (caching_test/serve_test hot paths), E2E cache-semantics
  hardening (Vary on 304, Content-Length, skip-policy variants, timing
  budget asserts), concurrent-poll hash-cost probe, `/api/ops` stale-pass
  warning, export memory-budget test, HTTP-stack audit as HTML research
  report (a concurrent session appears to be on it —
  `docs/research/…http-stack-adoption-audit.html` is dirty right now),
  sibling `/metrics` scrape, BuildFlow spawn-starvation upstream filing,
  status-report annotate/archive pass, recording-proxy injected-deadline
  refactor.
- Everything owner-blocked (untouched, correctly): v0.2.0 cut, prod
  re-ingest, host deploys, browser-level 304 check, upstream fold-ins,
  > 1MiB transcript policy, dependabot confirmations, Telnyx DTO live
  > re-verify, schema-v2 hash encoding.

## d) TOTALLY FUCKED UP

Nothing data-destructive or wrong-shipped. The honest list of damage:

1. **Six gate runs for one verification** — three failed runs were avoidable:
   two hit the daemon-commit race, one hit co-tenant OOM. Cost: ~40 min of
   wall clock and nix rebuild churn (each daemon sweep re-dirtied `-dirty`
   derivations and forced rebuilds). The tree and every artifact are correct;
   the process was noisy.
2. **Golden churn overlap**: I regenerated the goldens mid-session; the
   concurrent session regenerated two of the SAME files on top (their export
   note changed page output). Net state is consistent and green, but for a
   while two agents were writing the same fixture files with no coordination.
   That is a real collision risk, not just noise.
3. **flake.nix vendorHash paste bug** (source hash pasted as vendorHash) was
   pre-existing from an earlier session — I fixed it, but it should never
   have passed: the first cqrs-lint build after that edit would have caught
   it. Whatever review discipline let a copy-pasted hash land, it didn't run.

## e) WHAT WE SHOULD IMPROVE

1. **Gate precondition check**: tree-clean + load-quiet before `buildflow`
   runs; the triage rule is documented — make it a habit (or a wrapper).
2. **Concurrency protocol**: claim TODO_LIST rows (`IN_PROGRESS` status or a
   session tag) before starting work so two sessions don't regenerate the
   same goldens or fight the same files. Today's overlap worked out; it need
   not have.
3. **Read-the-source-first for third-party semantics**: two of my three test
   corrections came from modeling a library off its prose instead of its call
   sites (etag.go hooks) — the fix is cheap and mechanical.
4. **Golden determinism self-check**: a test that renders, scrubs, renders
   again under a shifted clock (or just double-render diff) would have caught
   both time bombs at capture time instead of at 17:00.
5. **nix-build lane resilience**: OOM-kills under co-tenant load are now the
   gate's dominant failure; a retry-with-backoff or a memory-aware skip (with
   the doctrine's individual-verification fallback made explicit in the
   failure message) would stop the thrash. Feeds the BuildFlow#45 evidence.
6. **Hash-hygiene guard**: a flake check that fails when a `vendorHash` equals
   its own `src` hash (trivially detectable nonsense) would have caught
   today's cqrs-lint paste bug at eval time.

## f) Up to 50 things to get done next (impact-ordered where possible)

1. Land or consciously discard the concurrent session's in-flight files
   (`export_stall_test.go`, `docs/research/…audit.html`) — confirm their gate
   state before stacking more work.
2. Reconcile TODO_LIST against commits `ba6ec8f`/`83fe566` (export-stall UI
   row: the list-page note shipped; the export-UI loud-fail surface may
   remain) and remove/reword rows as done.
3. Get one fully-green `buildflow --build-mode dev` run on a quiet machine
   (all 76+ steps including nix-build in a single run).
4. Cut **v0.2.0** (owner cadence): Unreleased now carries the HTTP stack +
   cache section + drill lane + export-stall note + ingest extraction.
5. Production re-ingest for ContentHash v2 (needs the deploy path).
6. Deploy both hosts (`aiDids` option; `-max-age`/`-recording-timeout`
   already wired in the sibling module).
7. Browser-level 304 + DOM-churn check (owner, no browser in agent env).
8. Fold in go-etag#2 (Flush observability) + go-appkit#26 when upstream
   lands; today's overflow counter gives `OnBufferOverflow` a consumer.
9. Fold in templ-components#48/#49/#50 when landed (retire local
   workarounds).
10. Decide the >1MiB transcript policy — `/api/ops buffer_overflows` now
    gives the evidence source for the accept-vs-raise decision.
11. Coverage pass asserting the cache hot path (fragment conditional GET) in
    caching_test/serve_test, not static stand-ins.
12. E2E cache-semantics hardening: `Vary: Accept-Encoding` on 304s,
    `Content-Length` on buffered renders, HEAD/query/trailing-slash skip
    variants, recording timing-hop budget asserts.
13. Concurrent-poll hash-cost sanity probe (CPU ceiling under overlapping
    30s/60s polls — hit_ratio now observable while it runs).
14. `/api/ops`: warn when the reported poller pass is stale (age lastPass) —
    the cache section now sets the payload precedent for a `warnings` array.
15. Export memory-budget test (pin `exportBatchSize` under bounded
    allocation).
16. Recording-proxy timeout tests → injected deadlines (kill the budget+sleep
    pattern).
17. Render the HTTP-stack audit as the library-deep-dive HTML report (or
    confirm the concurrent session's draft is it).
18. Sibling NixOS module: scrape `/metrics`, smoke-test with a real scraper.
19. File the BuildFlow spawn-starvation mode upstream (evidence: today's
    OOM-killed nix-build lane ×3).
20. Add the flake guard from §e.6 (vendorHash ≠ src hash check).
21. Add shellcheck + lychee to the devShell (gate currently nix-runs them
    WITHOUT project deps; "9 tools unavailable" health warnings).
22. ANNOTATE + archive the 2026-10-11/12 status reports (this one joins the
    harvest set) per docs-health ANNOTATE mode.
23. R27.x advisory residue: vulnix re-triage (22 findings in today's gate —
    all nixpkgs build-toolchain CVEs), art-dupl pass, nix-checker cleanup,
    branching-flow pass.
24. README screenshots (R27 residue).
25. Resolve the templ version skew (flake pkgs.templ 0.3.1020 vs go.mod
    0.3.1070) — go-nix-helpers upstream or flake bump.
26. Watch dependabot-core#16538; owner decides gomod pinning.
27. Confirm the weekly dependabot run proposes bumps for the newly-direct
    deps (httputil, server_timing, go-etag/server).
28. Live-verify Telnyx Detail Records + insights DTOs (2027-01 window).
29. Schema v2: length-prefixed ContentHash field encoding (scheduled).
30. PII policy enforcement (E164 + ParticipantHash are stored, unenforced).
31. Webphone read-only deep link (sibling-plan item).
32. CI: consider a lane that rebuilds the flake's cqrs-lint tool so hash
    drift like today's fails in CI, not only in devShell gates.
33. Consider surfacing `hit_ratio` on the dashboard UI (operator visibility
    beyond /api/ops).
34. `/api/ops` consumers: document the new `cache` section in
    docs/http-stack.md (the doc's line 46 still calls the wiring future work).
35. Retire the "Wire the go-etag cache hooks" mention from
    docs/status/2026-10-12_13-49 §f.10/11 as resolved (annotation pass).
36. Verify dprint's 3 auto-fixed files (FEATURES.md, docs/assets/README.md,
    docs/templ-components-adoption.md) are content-identical modulo
    formatting.
37. Confirm the go.sum tidy pruning landed (daemon sweep) and CI's tidy step
    stays green.
38. Double-check the scrubber's bare-date regex against a bucket label at
    end-of-string (the `$` alternative) with a unit case in
    TestGoldenScrubberCoversVolatileStrings.
39. Extend TestGoldenScrubberCoversVolatileStrings to cover the two NEW
    patterns (staleness region, bare dates) — currently only the original
    five are self-tested.
40. Decide whether `STALENESS_REGION`/`DATE` deserve named constants shared
    between scrubber and any future assertion (YAGNI unless reused).
41. Review `composeOpsPayload`'s non-map poller wrap (`{"poller":…}`) — no
    test covers it (no producer exists); add one or simplify the contract.
42. Add a rate/A.B test for CacheCounters concurrency (`-race` covers the
    current tests; a hammer test would document the contract).
43. Go-error-modernization sweep on the new code paths (errors.AsType
    policy) — new code returns none, confirm nothing to migrate.
44. README: screenshot of `/api/ops` cache section (ops story).
45. CHANGELOG: when v0.2.0 is cut, fold today's Unreleased section under the
    version heading (repo convention).
46. docs/upstream-filings.md: add today's observation to BuildFlow#45
    evidence (three OOM-killed nix-build lanes under co-tenant load).
47. Consider `On304`/`OnETagGenerated` upstream clarification request
    (etag.go docs are accurate but subtle — "fires first during tag
    resolution" cost me a test cycle; a doc PR candidate).
48. Verify the drill CI step's placement survives future step insertions
    (it must stay after templ generation — currently implicit by position;
    a comment says so, keep it true).
49. Re-run `nix flake check` standalone once quiet (the gate's nix-flake-check
    failure today was diagnostic-free and needs a clean verdict).
50. Decide the session-collision policy question (§g.1) and write it into
    AGENTS.md so the next concurrent session doesn't re-litigate it.

## g) Questions I cannot answer myself

1. **The concurrent session** (export-stall note, ingest.go extraction, the
   dirty `docs/research/…audit.html` and `export_stall_test.go` right now):
   is it still running and authoritative for internal/web + TODO_LIST? I
   stayed off its files once identified; if it is abandoned mid-flight I
   need to know whether to finish, revert, or quarantine its uncommitted
   work.
2. **v0.2.0 timing**: cut now with today's Unreleased set, or hold until the
   upstream fold-ins (go-etag#2, templ-components#48-50) land so the release
   can retire the local workarounds in the same stroke?
3. **Gate policy under co-tenant load**: is "individually verified casualty"
   (today's doctrine path) an acceptable terminal state for a session, or do
   you want pushes/green-light decisions to require one fully-green single
   gate run even if that means waiting out the other session?

— end of report —
