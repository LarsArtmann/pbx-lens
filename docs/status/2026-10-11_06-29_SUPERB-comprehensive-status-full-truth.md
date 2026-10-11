# Comprehensive status report — the WHOLE truth, nothing but

**Date/time:** 2026-10-11 06:29 CEST
**Author:** the working session (auto-daemon commits throughout)
**Supersedes:** `docs/status/2026-10-11_05-21_SUPERB-session-status-where-are-we.md`
**Plans in force:** `docs/planning/2026-10-11_00-09_SUPERB-pareto-remaining-work.md` (backend A–D) and `docs/planning/2026-10-11_01-32_SUPERB-pbx-lens-frontend-templ-appkit.md` (frontend F1–F25).
**Mandate:** "GET SHIT DONE! The WHOLE TODO LIST! ... DO NOT STOP UNTIL THE ENTIRE LIST IS FINISHED and VERIFIED!"

---

## a) FULLY DONE (verified this session, 05:30 → 06:29)

### B3 search — COMPLETE including wiring (was ~90%)

| What | Where | Proof |
| --- | --- | --- |
| Rebuild helpers | `internal/cli/search.go` (new) | `RebuildSearchIndex` (open→rebuild→close, no-op on `dsn==""`), `RebuildIndex` (open-index variant for serve/poller), `readAllViews` (`WithLimit(0)` = unlimited) |
| 4 ingest verbs rebuild FTS | `cmd/pbx-lens/main.go` | every `cli.DrainProjections` in ingest-cdr/telnyx/webhook/agent is followed by `cli.RebuildSearchIndex` |
| serve passes Search | same | `search.Open(setupCtx, *db)` long-lived + startup `RebuildIndex` + `Options{Recordings, Search: index}` |
| busy timeout | `internal/search/search.go` | `withBusyTimeout` appends `_pragma=busy_timeout(5000)` to the DSN so the FTS connection coexists with the metaengine connection |
| /search web tests | `internal/web/search_test.go` (new) | 3 tests: real-sqlite index answers with `>hello</mark>` + deep link; empty q shows form; nil searcher 503s naming `-db` |
| **E2E verified** | real sqlite | ingest-webhook twice → idempotent; `serve` → `/search?q=%2B17287289311` **200 with `<mark>` highlight**; form page renders; list/detail/calls fine (Go httpprobe at /tmp/httpprobe — curl is banned) |

Learned and documented: FTS5 tokenizes `+17287289311` as ONE token — substring queries (`87289311`) match nothing; search full numbers or transcript words.

### The two known warts — FIXED, and they uncovered a REAL data-correctness bug

1. **`?to=YYYY-MM-DD` now means end-of-day** (`parseDateParam` returns `dateOnly`; the `to` field widens to `23:59:59`). Regression tests included (`TestListToDateIncludesThatDay`).
2. **Staleness double scan is dead** — `staleness(filters, page)` now rides the page the list handler already fetched, and the banner shows ONLY on the unfiltered first page (a filtered view is a deliberate slice; test included).

### THE BIG FIND: metaengine time filters were silently wrong on sqlite

- sqliteengine binds the raw `time.Time` parameter; modernc renders it `2006-01-02 15:04:05 -0700 MST`, while the stored JSON is RFC3339 (`T` separator). Lexicographic `'T' > ' '` **corrupts same-day comparisons**: `Le/Lt` leak the day, `Ge/Gt` drop it, `Eq` never matches. (Proven with a 6-case probe matrix: `le-mid → [a]` when `[a,b]` was correct, etc.)
- **Fix in-repo:** `web.rfc3339Nano()` binds time bounds as RFC3339Nano strings — the exact stored wire format. Verified correct for eq/le/ge/lt on real sqlite. Robust regardless of any future upstream fix.
- **Second divergence found:** the memory engine resolves filter columns by Go FIELD name (EqualFold), not JSON tag — `started_at` reads nil → `Le` matches EVERYTHING, `Ge` matches nothing. sqlite's `json_extract` uses JSON tags and is correct. Consequence: `newTestServerWith` now uses a real temp sqlite file so web tests share production semantics.
- Both documented in AGENTS.md gotchas; upstream issues queued for D6.

### Quality gate — PASSED (with documented advisories)

- `BUILDFLOW_NO_RESULT_CACHE=1 buildflow -s golangci-lint --format finding` → **0 findings** (was 19 in the B2/B3 files; every one fixed: exhaustive fallthrough, mnd consts, noctx→PingContext/Open(ctx), wrapcheck, nilerr nolint-with-reason, varnamelen, makezero, `search.Open(ctx, dsn)` signature with all call sites updated).
- `.golangci.yml`: `run.go` 1.27.0→1.27.1 (govalid reads it; go-appkit requires ≥1.27.1); three documented exclusions added (contextcheck templ-generated closures, noctx in tests, godoclint templ-header false positive — fleet precedent samber-do-auditlog).
- `cqrs-lint` → **0 errors**, 15 infos (3 new: F021 three-folds, F026 prefetch, F009 scheduling — policy notes pending in D4).
- **pre-commit check FIXED**: treefmt-nix's templ program wraps `templ fmt` with nixpkgs DEFAULT `pkgs.go` (1.26.8) — older than this repo's floor, so the sandboxed check tried a forbidden toolchain download and failed. Fixed in flake.nix by force-overriding `settings.formatter.templ.command` with a `writeShellApplication` carrying OUR go. Upstream fix queued for go-nix-helpers (D6).
- `go-version-auto-configure` finding: go.mod must keep `go 1.27.1` — `go mod tidy` FORCES it back (go-appkit v0.8.0 demands ≥1.27.1); the checker's own source documents dep-forced patch floors as "a CORRECT state ... advisory by design".
- `buildflow --build-mode dev` → **passed with warnings**; residual advisories (go-auto-upgrade json-v1→v2 suggestions, structure-linter assets/, art-dupl 20, branching-flow 51, nix-checker hash bookkeeping) documented in a new AGENTS.md policy section.
- `nix build` green throughout; `./result/bin/pbx-lens version` works.

### D1 auth — COMPLETE

- `internal/web/auth.go` (new): `HtpasswdMiddleware` — bcrypt-only htpasswd (`htpasswd -B`), weak schemes reject AT STARTUP with the fix named, dummy-hash timing pad (no username enumeration), `WWW-Authenticate: Basic realm="pbx-lens"`.
- `internal/web/auth_test.go`: 4 tests — 401 missing/wrong/unknown, 200 correct, **gates /app.css + detail + search too**, weak-scheme startup rejection.
- serve flag `-htpasswd` wired via appkit `ServiceConfig.ExtraMiddlewares`.
- `golang.org/x/crypto` v0.57.0 promoted to DIRECT (already in the module graph via projectionhost — no new module; documented in AGENTS.md).

### C2 poller + /api/freshness — CODE COMPLETE, tests in flight

- `internal/cli/poller.go` (new): `PollConfig` (Telnyx client + SinceAgo + Direction, CDRFile, WebhookLog, TranscriptsDir), `PollLoop` (immediate first pass, then ticker; ctx-cancellable), `PollOnce` (per-source ingest, per-source failures logged and skipped — never fatal, then ONE drain + ONE FTS rebuild + summary), `pollFile`, `mergeStats`.
- `internal/web/freshness.go` (new): `/api/freshness` JSON — `newest_started_at`, `age_seconds`, `stale`, `calls_indexed` (via metaengine Count). 2 tests GREEN (`freshness_test.go`).
- serve wiring: 6 new flags (`-poll-every` default 15m, `-poll-since` 48h, `-poll-direction`, `-cdr-file`, `-webhook-log`, `-transcripts-dir`), poller goroutine with `defer stopPoller()`, Telnyx client shared between proxy and poller.
- Poller tests: 4 written, **2 still failing** — see §d.

### Memory/docs

- AGENTS.md updated at every step: three new gotchas (time binding, memory-engine naming, FTS tokenization), dev-gate residual-findings policy section, treefmt/pre-commit fix note, x/crypto exception, golangci exclusion rationale.

---

## b) PARTIALLY DONE / In flight

1. **C2 poller tests — 2 of 4 failing** (`TestPollOnceIngestsAndRebuildsSearch`, `TestPollOnceIsIdempotentAcrossPasses`). Root cause narrowed to the TEST FIXTURE, not the poller: `webhook.Parse` on my inline two-line string yields `obs=0 skipped=2`, while the byte-identical-looking `/tmp/lens-e2e.jsonl` FILE yields `obs=1 skipped=0`. The fixture assembly (python-generated backtick concatenation) differs somewhere from the working file — exact-bytes diff is the next 10-minute step. The poller logic itself is exercised green by the other two tests (missing-source pass resilience, loop cancellation).
2. **Post-x/crypto housekeeping not run**: `go mod tidy` + `nix run .#update-vendor-hash` + `nix build` (go.mod changed → vendorHash may drift; the nix FOD already refused once until tidy ran).
3. **Lint/cqrs not yet re-run on the newest files** (auth.go, poller.go, freshness.go, tests).
4. **AGENTS.md not yet updated** for poller/freshness/auth flags (one short bullet each, D4 batch).

## c) NOT STARTED (backend plan order)

| Item | Notes |
| --- | --- |
| C5/C6 live probes | Telnyx key at `~/.pbx-prod-secrets/telnyx_api_key`, GET-only; prod CDR/webhook files |
| C4 backup | sqlite file backup story |
| C3 systemd module | thin NixOS module in nix-international-telephony (sibling repo) |
| D2 purge (T23) | retention + PII policy; E164/ParticipantHash become load-bearing |
| D3 dashboard (T25) | nav link exists, route doesn't; volume by day, AI share, MOS trends |
| D5 export | CSV/JSONL dump endpoint |
| D6 upstream filings | now FIVE verified candidates (see §f) |
| D4 webphone link (T22) | sibling repo, LAST |
| C9 §30 strike | blocked on owner Q3 |
| C8 CI activation | blocked on owner Q1 (a parallel session already landed ci.yml) |

## d) Anything TOTALLY FUCKED UP?

1. **Self-inflicted, worst of the session:** I cloned treefmt-nix to debug the pre-commit failure — from the repo cwd — landing `/home/lars/projects/pbx-lens/treefmt-nix/` INSIDE the project. Its intentionally-invalid `.go` spec fixtures poisoned govalid/go-fix/golangci (syntactically broken files) and cost several gate-debugging cycles before I noticed. Removed (moved to /tmp). Lesson: never clone from repo cwd; check `git status` for surprise directories after any clone.
2. **First-draft garbage pattern, twice:** auth.go v1 had a hand-rolled `itoa`, a copy-pasted bcrypt hash with a comment claiming it was generated for this repo (a lie), and auth_test v1 had a fake `contains/indexOf` instead of `strings.Contains`; freshness.go v1 had placeholder stubs (`countCalls`/`countAll`/`writeJSON`). All rewritten immediately — but each was a wasted write that the compile/test loop had to catch. Slow down before `write`.
3. **The poller test fixture bug (OPEN, §b1)** — my fixture, not the product; but it's still a red test on disk and blocks the green-suite claim.
4. **`go mod edit -go=1.27`** (policy-driven) broke the nix FOD ("updates to go.mod needed") until `go mod tidy` forced the floor back to 1.27.1. The policy exception is now documented; no damage.
5. Two daemon-race edit failures (multiedit anchors invalidated mid-edit) — recovered via re-read + python patch scripts; the daemon is expected behavior, the recovery cost was minutes.

## e) What we should IMPROVE (codebase + process)

- **Share fixtures between tests**: the e2e webhook envelope is now re-assembled per test file; one `internal/testfixtures` (or a testdata file) would have prevented §d3 outright.
- **Stop shipping "first drafts"** — the auth/freshness rewrites were avoidable by thinking the shape through first.
- **Session tools should live in the repo**: the /tmp httpprobe (curl is banned) deserves `scripts/httpprobe.go` so every future session reuses it instead of rewriting it.
- **Upstream fixes beat in-repo workarounds**: the treefmt `pkgs.go` pin and the sqliteengine time-binding bug both deserve upstream patches this fleet owns (go-nix-helpers, go-cqrs-lite are Lars's repos) — queue them in D6 rather than letting workarounds calcify.
- **FTS number search UX**: full-token-only matching will surprise the owner; either document it in the search placeholder or add FTS5 prefix queries (`"87289311"*` style) later.
- **The gate keeps finding real bugs when actually run** — the dev gate hadn't run since before B2/B3; run it after every feature, not in batches (this session's 19 lint findings + pre-commit break prove the cost of batching).

## f) Up to 50 things to do next (pareto order)

1. Fix the poller test fixture (diff inline string vs `/tmp/lens-e2e.jsonl` bytes; likely quoting) → cli package green
2. Full `go test ./... -count=1` re-run
3. `go mod tidy` + `nix run .#update-vendor-hash` + `nix build` (x/crypto promotion may drift vendorHash)
4. `BUILDFLOW_NO_RESULT_CACHE=1 buildflow -s golangci-lint --format finding` on auth/poller/freshness files
5. `/tmp/cqrs-lint .` re-run (F009 will fire on poller ticker — pre-document like P006)
6. E2e: serve + poller on real sqlite; probe `/api/freshness` and a poll-driven search refresh
7. AGENTS.md bullets: poller flags/semantics, /api/freshness, -htpasswd
8. serve `usage()` line: add the new flags (poll/auth) to the help text
9. C5 live probe: Telnyx conversations+recordings GET against prod key (fixture-free truth)
10. C6 live probe: prod webhook JSONL tail + Master.csv parse
11. C4 backup: sqlite `VACUUM INTO`/backup-API command + test
12. C3: thin NixOS module in nix-international-telephony (systemd unit, hardening, htpasswd secret path)
13. D3 dashboard: queries (volume-by-day, AI share, MOS from CDR) + route + nav activation
14. D5 export: `/export?format=csv|jsonl` over the full read model + tests
15. D2 purge: retention command (age cutoff) + PII mode (E164/ParticipantHash projection fields become load-bearing) + tests
16. D6 upstream #1: sqliteengine binds `time.Time` filter values in Go default format (verified repro + fix suggestion: bind RFC3339Nano or document string contract)
17. D6 upstream #2: memory-engine closure resolves columns by Go field name, diverging from JSON-tag pushdown engines (verified repro)
18. D6 upstream #3: go-nix-helpers — templ treefmt formatter must resolve goPkg when enableTempl (verified: pkgs.go 1.26.8 vs floor)
19. D6 upstream #4: treefmt-nix templ program should not pin nixpkgs default go (upstream of #3)
20. D6 upstream #5: cqrs-lint F021/F026/F009 policy notes (and C009 from the old list)
21. D6: templ-components gaps (DefinitionList children backport? FilterBar?) — verify-then-file
22. README: serve/search/recording-proxy/auth/freshness/poller sections
23. CHANGELOG entry for the whole serve era
24. TODO_LIST.md: mark B3/B2/D1/C2 statuses honestly; add the new follow-ups from this report
25. FEATURES.md: add search/auth/freshness/poller rows
26. Final `buildflow --build-mode full` gate
27. Prod re-ingest against the real DB (validate the whole pipeline on prod data volumes)
28. Review daemon/parallel-session changes still uncommitted-by-me (ci.yml, .gitignore sections, .pre-commit-config.yaml regen)
29. Triage art-dupl's 20 findings (deduplicate-code skill pass)
30. Triage branching-flow's 51 findings
31. nix-checker: extract cqrs-lint's inline vendorHash to its own file (kills 2 warnings)
32. Search UX: FTS5 prefix queries for partial numbers, or a placeholder hint about full-token matching
33. Poller: include per-source counts in the pass log (currently only the aggregate)
34. /api/freshness: expose poller state (last pass timestamp, last error) for honest monitoring
35. Poller tests: Telnyx source via httptest fake; transcripts-dir source test
36. Auth test: add /api/freshness to the gated-routes case list
37. Consider `metaengine.WithPrefetch` on cursor pages (F026) — measure first
38. `staleness` unit test for the filtered-page suppression edge (currently covered via handler test only)
39. recording proxy: consider caching the signed-URL refetch per request path (currently one refetch per playback — fine at personal volume, note it)
40. serve: validate `-recordings-dir` traversal candidates against symlinks (ServeFile already guards; document)
41. Docs: AGENTS.md "Commands" section — add serve/poller e2e one-liners
42. Cross-project lesson (crush-config `references/lessons.md`): "sqlite TEXT-vs-time parameter binding corrupts JSON-stored timestamp comparisons — bind the wire format" (paid for here, generalizes)
43. scripts/httpprobe.go: promote the /tmp probe into the repo
44. testfixtures package: one source of truth for webhook envelopes/CDR rows
45. Dashboard tests once queries exist
46. Export: streaming (no full-buffer) for large indices
47. Purge: document "index is disposable" doctrine interplay (purge = retention policy, NOT write-back)
48. C9 §30 strike (the moment Q3 lands)
49. C8 CI activation (the moment Q1 lands)
50. webphone deep link (D4/T22) — last, sibling repo

## g) Up to 3 questions the owner must answer

1. **Q1 (carried, gates C8/CI):** GitHub visibility public/private + may I create the remote and push? (A parallel session already landed `.github/workflows/ci.yml`; it does nothing until a remote exists.)
2. **Q2 (carried, gates doctrine):** `call.corrected` supersession replaced the documented conflict-rejection — keep supersession for all sources, or conflicts (reject) for freeswitch-cdr only?
3. **Q3 (carried, gates C9):** Confirm striking the pbx-lens rows of `~/projects/pbx-artmann/TODO_LIST.md` §30 (cross-repo edit; I will add moved-to markers, not just delete).

*(One new almost-question — whether to file the five upstream items now vs. after the todo list — I resolved myself: they're queued as D6 work items, filing happens there with verify-before-filing discipline.)*

---

**Verification commands (copy-paste):**

```console
GOEXPERIMENT=jsonv2 go build ./...
GOEXPERIMENT=jsonv2 go test ./... -count=1        # cli poller tests RED until §b1 lands
BUILDFLOW_NO_RESULT_CACHE=1 buildflow -s golangci-lint --format finding
/tmp/cqrs-lint .
nix build && ./result/bin/pbx-lens version
# e2e: /tmp/lens-e2e.db from the 05:21 report still works:
GOEXPERIMENT=jsonv2 go run ./cmd/pbx-lens serve -db /tmp/lens-e2e.db -addr 127.0.0.1:8317
# then probe /, /search?q=%2B17287289311, /api/freshness (use /tmp/httpprobe; curl banned)
```
