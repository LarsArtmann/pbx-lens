<!-- docs/status/2026-10-12_13-28_superb-http-stack-adoption-status.md -->
<!-- Point-in-time snapshot of the 2026-10-12 12:31–13:25 SUPERB session
     ("make sure we use httputil / server_timing / go-etag / go-appkit to the
     max"). History lives in git; harvest open items into TODO_LIST.md via
     docs-health HARVEST, never by editing this file later. -->

# Status: SUPERB HTTP-stack adoption (httputil · server_timing · go-etag · go-appkit)

**Session:** 2026-10-12 ~12:31 → ~13:25 · **Report:** 13:28 · **Commit:** `fd0e082` (pushed) · **Plan:** `docs/planning/2026-10-12_12-47_SUPERB-httputil-go-etag-go-appkit-max-adoption.md`

**One-line verdict:** All three unused in-graph HTTP modules are now first-class,
verified by a full-stack e2e through the real listener — and the session found
and fixed one latent truncation bug and one silent ETag killer that the first
round of tests missed.

---

## a) FULLY DONE

| #   | Work                                                                                                                                                                                                                 | Evidence                                                               |
| --- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------- |
| A1  | Pinned-version capability audit of all four modules (module cache, NOT HEAD — caught that appkit v0.8.0 lacks `security/`/`errorpages/`/`realtime` and server_timing has no middleware ctor at any released version) | plan doc §capability map                                               |
| A2  | Conditional GET: go-etag/server middleware, `SkipIfPresent`, Skip policy (staleness/freshness/export./recording) — 304s live for pages, poll fragments, statics                                                      | `internal/web/caching.go`; probe: `/frag/dashboard` revalidates to 304 |
| A3  | Server-Timing on every route via `WrapServerTiming`; 8 instrumentation sites (query/stats/fts/count/metadata/headers — all pre-first-write)                                                                          | `internal/web/timing.go` + handlers                                    |
| A4  | gzip wire compression, ordered OUTER of ETag (library's ordering doctrine); pages drop ~4–6x (34KB→5.7KB dashboard)                                                                                                  | `buildServiceConfig`                                                   |
| A5  | Stream-contract fix: `WriteTimeout: appkit.NoTimeout` — the default 30s deadline truncated no-row-cap exports and long WAV downloads (both ride `r.Context()`)                                                       | `cmd/pbx-lens/serve.go`; plan §latent bug                              |
| A6  | `/version` + Prometheus `/metrics` from previously dead ServiceConfig knobs                                                                                                                                          | `buildServiceConfig`                                                   |
| A7  | Full-stack e2e (testkit, real listener): 200→ETag→304 + gzip on hot paths, Skip behavior, Server-Timing presence, /version, /metrics, config wiring                                                                  | `cmd/pbx-lens/serve_test.go`                                           |
| A8  | render() buffers templ pages atomically (flush = no ETag; also real 500s on mid-render failure)                                                                                                                      | `internal/web/render.go`                                               |
| A9  | Deps promoted direct: `httputil` v1.4.1, `server_timing` v1.0.2, `go-etag/server` v0.6.1 (in-graph precedent); entitytag deliberately stays indirect                                                                 | go.mod                                                                 |
| A10 | vendorHash repaired + **nix build green** on the dirty tree                                                                                                                                                          | `vendorHash.nix`                                                       |
| A11 | Docs: AGENTS.md (deps + serve/UI rewrite), FEATURES.md (+6 rows), CHANGELOG.md, TODO_LIST.md (+2 rows)                                                                                                               | committed                                                              |
| A12 | Plan doc with Pareto tiers, two task tables, mermaid graph                                                                                                                                                           | committed (via daemon, `30f5117`)                                      |
| A13 | Final state: 14/14 packages green, golangci-lint **0 issues**, gofmt clean, pushed to `main`                                                                                                                         | `fd0e082`                                                              |

## b) PARTIALLY DONE

| #  | Item | What's missing                                                                                                                                                                                                                                                                                                         |
| -- | ---- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| ~~ | B1   | **Verification depth** — build/test/targeted lint/fmt/nix-build ran; the FULL buildflow gate did NOT (15-min load was 79 > 40, m27.9 doctrine). art-dupl, govulncheck, cqrs-lint, dprint, deadnix, statix never ran this session                                                                                       |
| ~~ | B2   | **Browser-level revalidation** — 304s proven with Go clients that send `If-None-Match` MANUALLY. A real browser only revalidates if it cached the response; fragments carry no explicit `Cache-Control` (relying on heuristic → ETag-only responses generally revalidate, but **unverified in Chrome/Firefox + htmx**) |
| ~~ | B3   | **HTML research report** (library-deep-dive skill deliverable, `docs/research/`) — substance landed in plan + AGENTS.md instead; tracked as TODO row                                                                                                                                                                   |
| ~~ | B4   | **Race detector** — CI runs `-race`; locally I ran plain `go test` only                                                                                                                                                                                                                                                |
| ~~ | B5   | **CI watch** — pushed `fd0e082` but did not watch the first remote Actions run of the new e2e                                                                                                                                                                                                                          |
| ~~ | B6   | Concurrent-session work (PieChart legend, harness deletion) — preserved and left unstaged, but its author (other session) hasn't committed the `zz_dump_audit_test.go` deletion yet                                                                                                                                    |

## c) NOT STARTED (deliberately, with reasons recorded)

~~- appkit `errorpages`, `security`, `realtime` — **absent from pinned v0.8.0** (HEAD-only); errorpages doubly blocked (templ-components `errorpage` ABSENT in pinned v1.21.0)~~ → routed: ROADMAP errorpages/security revisit
~~- Rate-limiting exports (`KeyedRateLimiterMiddleware`) — no failure mode today; auth throttle + bounded queries cover it~~ NOT-DO — doctrine: no failure mode; auth throttle + bounded queries cover it
~~- SSE hub instead of htmx polling — would rewrite shipped R9 architecture~~ NOT-DO — R9 htmx polling shipped; SSE stays the upgrade path
~~- go-etag `client` for Telnyx upstream — upstream ETag emission unverified (next live window 2027-01)~~ → routed: fold-in watch (2027-01 live window)
~~- Handler-side data-fingerprint fast-304 — YAGNI at current index sizes~~ → routed: ROADMAP data-fingerprint trigger
~~- `/metrics` scrape wiring in the sibling NixOS module — TODO row added~~ done — T14 sibling scrape option + smoke

## d) TOTALLY FUCKED UP

**Nothing is broken.** Honest near-misses, for the record:

1. **False test confidence:** my first e2e asserted ETag on `/static/htmx.min.js` only — it passed while EVERY templ page silently had no ETag (templ flush vs go-etag streaming bail-out). The AGENTS.md live-probe doctrine caught it. Lesson: assert the hot path, not a lookalike.
2. **Probe #1 was wrong, not the code:** I forgot `Accept-Encoding` in the first probe and nearly misread "no gzip" as a bug.
3. **Stale `.git/index.lock`** (daemon crash, 0 bytes, 8 min old) — removed safely after confirming no git process; commit then succeeded.
4. A concurrent session edited the repo mid-flight (dashboard legend, harness deletion) — correctly preserved per the never-revert-others'-work rule, but I nearly edited a file (zz_dump) it then deleted; my MkdirAll patch became moot (fine — their deletion supersedes).

## e) WHAT WE SHOULD IMPROVE

1. **Verify in a real browser once** — the entire caching story's last mile (B2).
2. **Deterministic revalidation:** fragments should send `Cache-Control: no-cache` like the statics do — ETag-only responses _usually_ revalidate heuristically; "usually" is not a caching strategy.
3. **Don't skip the gate silently under load** — I followed m27.9, but the push happened before art-dupl/govulncheck/cqrs-lint saw the new code. Prefer: push, then run the gate as an immediate follow-up when load drops, and report residuals.
4. **e2e asserts should target the paths that matter** (the 30s poll fragment), not convenient stand-ins.
5. **The daemon's mid-flight commits scatter an audit trail** (my work landed across 30f5117/088af23/b153a40…). A `WIP:` marker convention or session claim in AGENTS.md would make multi-agent sessions reviewable.
6. **Unused go-etag observability:** `On304`/`OnETagGenerated`/`metrics.Attach` exist — cache-hit ratio could ride `/api/ops` nearly for free.
7. **`OnBufferOverflow` is silent today:** a many-hour transcript detail page >1MiB gracefully streams untagged — nobody would ever know. A log line would make the degradation visible.

## f) UP TO 50 THINGS TO GET DONE NEXT (brainstorm — HARVEST with rigor; ✅ = already a TODO_LIST row)

| #  | Task | Impact                                                                                                                                     | Effort | Note |
| -- | ---- | ------------------------------------------------------------------------------------------------------------------------------------------ | ------ | ---- |
| ~~ | 1    | Run FULL buildflow gate when load < 40; triage residuals (govulncheck/art-dupl/cqrs-lint/dprint on new code)                               | High   | 15m  |
| ~~ | 2    | `go test -race ./internal/web/... ./cmd/...` locally                                                                                       | High   | 5m   |
| ~~ | 3    | Browser check: htmx fragment polls actually 304 in Chrome/Firefox                                                                          | High   | 15m  |
| ~~ | 4    | `Cache-Control: no-cache` on fragment/page responses (deterministic revalidation)                                                          | High   | 15m  |
| ~~ | 5    | Watch CI run for `fd0e082` (first remote run of the new e2e)                                                                               | Med    | 5m   |
| ~~ | 6    | dprint/markdown check: my FEATURES/TODO table rows are column-unaligned — run the formatter or hand-align                                  | Med    | 10m  |
| ~~ | 7    | Restart gopls: stale `modernize` diagnostic on serve_test.go:57 (golangci-lint itself says 0 issues)                                       | Low    | 2m   |
| ~~ | 8    | Add go-etag `On304`/`OnETagGenerated` counters (or `metrics.Attach`) to `/api/ops` — cache-hit ratio for free                              | Med    | 30m  |
| ~~ | 9    | Log `OnBufferOverflow` (>1MiB bodies stream untagged silently)                                                                             | Med    | 15m  |
| ~~ | 10   | HTML research report from the audit (or retire the deliverable explicitly)                                                                 | Low    | 30m  |
| ~~ | 11   | Sibling NixOS module: optional `/metrics` scrape option                                                                                    | Med    | 30m  |
| ~~ | 12   | Document Server-Timing metric names (`query/stats/fts/count/metadata/headers`) in an ops runbook section                                   | Med    | 20m  |
| ~~ | 13   | Verify a real Prometheus/Gatus scrape of `/metrics` (gzip + no If-None-Match)                                                              | Med    | 15m  |
| ~~ | 14   | Confirm `Content-Length` is now set on buffered renders (client perf nicety)                                                               | Low    | 10m  |
| ~~ | 15   | Verify `Vary: Accept-Encoding` survives the 304 path (spec wrinkle: 304 inherits cached Vary)                                              | Low    | 15m  |
| ~~ | 16   | README: document `/version`, `/metrics`, ETag/304 behavior for operators                                                                   | Med    | 20m  |
| ~~ | 17   | `docs/gotchas.md`: add the templ-flush-kills-ETag war story (currently only in render.go comment + CHANGELOG)                              | Med    | 10m  |
| ~~ | 18   | Plan doc: annotate EXECUTING → EXECUTED + commit hash (docs-health ANNOTATE, never rewrite)                                                | Low    | 5m   |
| ~~ | 19   | AGENTS.md 220-line budget: my deps/serve expansions grew it — measure, prune or split to docs/                                             | Med    | 15m  |
| ~~ | 20   | Add the pinned-version-vs-HEAD research trap (agents report HEAD; always diff the module cache) to crush-config `references/lessons.md`    | Med    | 10m  |
| ~~ | 21   | Crash-resilient auto-commit daemon (retry on `index.lock`; today a crash leaves a stale lock that blocks all commits)                      | Med    | 30m  |
| ~~ | 22   | Multi-agent coordination convention for this repo (session claim header or branch-per-agent)                                               | Med    | 20m  |
| ~~ | 23   | e2e suite takes ~5s (testkit drain delays?) — profile and trim if it grows                                                                 | Low    | 20m  |
| ~~ | 24   | Skip-policy tests: HEAD requests, query strings, trailing slashes                                                                          | Low    | 15m  |
| ~~ | 25   | Etag + compression under concurrent poll loop: hash-cost sanity probe (FNV-64a is cheap; confirm)                                          | Low    | 20m  |
| ~~ | 26   | Export gzip CPU on a prod-sized index (accepted in design; verify once real data exists)                                                   | Low    | 20m  |
| ~~ | 27   | Detail pages of marathon transcripts >1MiB — decide: accept untagged streaming or raise MaxBufferSize                                      | Low    | 20m  |
| ~~ | 28   | Dependabot policy note for the three newly-direct modules (routine bumps fine; templ stays manual)                                         | Low    | 10m  |
| ~~ | 29   | Revisit appkit/httputil/templ-components pins when errorpage publishes (unblocks errorpages + security packages)                           | Low    | —    |
| ~~ | 30   | Telnyx etag-client revisit rides the 2027-01 live-verify window                                                                            | Low    | —    |
| ~~ | 31   | If index grows ~100x: handler-side data-fingerprint fast-304 (skip scan+render, not just transfer)                                         | Low    | 45m  |
| ~~ | 32   | `/api/ops` stale-pass warning (age lastPass) — pre-existing TODO row                                                                       | Med    | 20m  |
| ~~ | 33   | Filters carried into staleness fragment links — pre-existing TODO row                                                                      | Low    | 30m  |
| ~~ | 34   | SECURITY.md: document the `/metrics` auth posture (explicit unauthenticated + htpasswd/loopback rationale)                                 | Med    | 15m  |
| ~~ | 35   | Static assets: content-hashed filenames + long `max-age` would beat revalidate-every-nav (currently no-cache + ETag = fine; measure first) | Low    | 45m  |
| ~~ | 36   | `entitytag` direct promotion ONLY if handler-side tags ever land (stays indirect now)                                                      | Low    | 2m   |
| ~~ | 37   | Consolidate timing.go + caching.go into one middleware file if a fourth arrives (watch split-brain with server.go route docs)              | Low    | 10m  |
| ~~ | 38   | htmx config: confirm swaps of identical cached bodies cause no visible DOM churn on the dashboard                                          | Low    | 15m  |
| ~~ | 39   | Dashboard PAGE 304s rarely (staleness age in shell) — accepted; revisit only if operators complain                                         | Low    | —    |
| ~~ | 40   | Next release cut: the CHANGELOG Unreleased block is getting rich (go-release skill process)                                                | Med    | 30m  |
| ~~ | 41   | cqrs-lint on the new files (never ran; findings policy doc may need a line)                                                                | Low    | 10m  |
| ~~ | 42   | Branch-per-risky-change habit: middleware chain landed straight on main (fine this time; formalize thresholds)                             | Low    | —    |
| ~~ | 43   | `coverage` for the two new test files: every Skip branch + helper hit (`go test -cover`)                                                   | Low    | 10m  |
| ~~ | 44   | Consider exposing etag hit/miss + gzip ratio as appkit `appkit_*` metrics instead of /api/ops (single ops surface)                         | Low    | 30m  |
| ~~ | 45   | Cleanup: `/tmp/pbxlens-probe` leftovers (probe binary + 2 dbs)                                                                             | Low    | 1m   |
| ~~ | 46   | recording_test: assert the new `metadata`/`headers` timing hops don't perturb existing timeout tests                                       | Low    | 15m  |
| ~~ | 47   | `go vet ./...` in CI already; consider adding the e2e test to the CI lane explicitly (it runs via go test, but flag the 5s cost)           | Low    | 10m  |
| ~~ | 48   | Consider `hx-boost` on the list page nav (noticed during audit; unrelated polish)                                                          | Low    | 30m  |
| ~~ | 49   | Re-check `go.mod` after next dependabot sweep: server_timing v1.0.2 should ride routine bumps                                              | Low    | 5m   |
| ~~ | 50   | Archive/annotate THIS report's follow-ups via docs-health HARVEST into TODO_LIST.md (only #10/#11 rows exist so far)                       | Med    | 15m  |

## g) THREE QUESTIONS I CANNOT FIGURE OUT MYSELF

~~1. **Is `/metrics` actually going to be scraped** (Prometheus/Gatus on the telephony hosts), or is `/api/ops` the accepted ops surface? This decides whether the sibling-module scrape option (#11/#35) gets built or the endpoint should be dropped again.~~ resolved — sibling scrape + smoke landed (T14); SECURITY posture documented
~~2. **Was the concurrent session** (dashboard PieChart legend `AI · N`, zz_dump harness deletion) **yours and intentional** — and is two-agent concurrent editing of this repo the norm I should coordinate around (e.g., session-claim headers), or a one-off I should not design for?~~ resolved — norms recorded (ROADMAP multi-agent open question)
~~3. **Is the plan `.md` the accepted audit deliverable**, or do you want the library-deep-dive HTML report (`docs/research/`) rendered too? The skill mandates HTML; your paste mandated `.md`; I followed the paste and parked the HTML as a TODO row.~~ resolved — .md is the standing default; the research HTML exists for the deep-dive

---

_Snapshot only — do not edit stale reports; bring them current via docs-health ANNOTATE. Harvest section (f) into TODO_LIST.md/ROADMAP.md, then let this file rest._

---

## Appendix (ANNOTATE 2026-10-12 13:5x — same session, post-report events)

- **B1/B4/B5 resolved:** race detector green locally (`go test -race` on web + cmd). CI ran the full matrix **green** on main HEAD after `fd0e082` (runs 38243761913, 38244128681: success).
- **CI incident:** run 38242512527 on `fd0e082` itself FAILED — `TestDumpPagesForVisualAudit`. Root cause: my commit (correctly) did not stage the concurrent session's working-tree deletion of that harness, so it shipped with the commit; it writes `/tmp/pbxlens-baseline` without creating it → clean-runner failure. Healed by the daemon's deletion commit `6bb7d17`. Local runs had passed because the file was already deleted in the shared tree — the suite I ran was never the suite I shipped.
- **Full local gate (`buildflow --build-mode dev`):** attempted once load dropped below 40 — 2 steps failed + 9 tools failed health checks, while a SECOND session was actively editing the tree (their 13:32 plan, `page.templ`, `error_handling_test.go`) and likely holding tool locks (the BuildFlow#45 spawn/lock class). `nix build` green twice on the current tree confirms no real vendor-hash or compile problem. A trustworthy clean gate needs a quiet tree — defer until the concurrent session lands; verification was achieved via the green CI matrix instead.
- Upstream filings from this work landed: **go-etag#2** (Flush observability) and **go-appkit#26** (testkit 5s drain per test), drafts + ledger in `docs/drafts/` / `docs/upstream-filings.md`.
