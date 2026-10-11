<!-- docs/status/2026-10-11_20-26_dispatch3-reuse-types-and-gate-status.md
     Status report for the 2026-10-11 ~19:40-20:26 dispatches: reuse-first
     execution round (go-atomic-write evaluation, typed boundary models,
     AgeGate reuse, auth throttle, CI lane) + gate verification. Format .md
     per this repo's recorded dispatch-report convention. All numbers
     re-verified live at 20:26 CEST. -->

# Status — Reuse-First Round: go-atomic-write Evaluation, Typed Models, Gate Verification

Date: 2026-10-11 20:26 CEST · Baseline: dispatch-two's 127 test functions
green, one pending full-gate rerun.

## a) FULLY DONE

| Item | Evidence |
| --- | --- |
| **go-atomic-write evaluated per your pointer (reuse-first, verified at source)** | Read `/home/lars/projects/go-atomic-write` README + `atomicwrite.go` + `rename_unix.go`: its stage→fsync→rename→dir-fsync chain is exactly backup's need, BUT its exported API (`Write`/`WriteFunc`/…) takes Go-owned data while backup's staging file is written by sqlite itself (`VACUUM INTO ?`); `atomicRename`/`syncDir` are unexported; module NOT in pbx-lens's graph. Decision recorded in AGENTS.md dependency doctrine; adoption deferred until an exported commit primitive exists |
| **Backup hardened with the library's unique-staging discipline** (the one real lesson adoption could not carry): staging name is now per-run unique (`out.tmp-<pid>-<nanotime>`) — two concurrent backup runs can no longer corrupt each other through the shared `out+".tmp"` path; deferred best-effort staging cleanup replaced the stale fixed-name pre-clean | `internal/backup` tests green (5 tests incl. overwrite + verify paths exercise the new flow) |
| **Upstream ask drafted**: `docs/drafts/go-atomic-write-commit-primitive.md` (Commit primitive for foreign-producer staging files; owner-review box unchecked) | draft file; not posted (your repo — API decision is yours) |
| **Poller admission path unified** (killed the dispatch-two asymmetry): `AgeGate.Apply` + `GateIngest` + `SeedsFor` in `internal/cli` — one gate, one stats finisher; the telnyx seed co-filter is a shared helper, not inline logic | `internal/cli` green |
| **One-shot `ingest-* -max-age` (all four commands)**: `ingest-cdr`, `ingest-telnyx`, `ingest-webhook`, `ingest-agent` now carry the same retention gate as the serve poller — a CLI backfill cannot silently re-import over-age rows; ingest-telnyx also drops refused enrichment seeds | `TestGateIngestRefusesOverAgeAndReportsAgainstBatch` (offers 2, records 1, rejects 1 with `retention:age-cutoff`; zero MaxAge admits everything); suite green |
| **Typed boundary models**: `calls.RetentionReason` (typo'd audit reasons no longer compile; event payload stays string — wire-identical), `web.PIIMode` (`PIIModePlain`/`PIIModeHash`), `web.RecordingFetcher` (consumer-side interface; `*telnyx.Client` satisfies as-is, web no longer imports the concrete client type) | all affected packages green; app.go's `calls.Retire(cmd.Reason, …)` type-aligned with zero changes |
| **Auth failed-attempt throttle**: shared `x/time/rate` bucket (6s interval, burst 5 ≈ 10/min sustained) on the bcrypt-failure path only; successes never wait; promoted `golang.org/x/time` indirect→direct via the D1 precedent (already in the module graph — no new module) | `go.mod` line 22; all auth/flow tests green, none exceeds the burst; deliberate NO timing-test pin (would add ~6s flaky time) |
| **Checked-in UI e2e probe**: `TestLiveServerEndToEnd` — real `httptest.NewServer` listener over the registered mux, real sqlite index, real HTTP on `/`, `/dashboard`, `/search?q=…`, `/api/freshness`; wiring reuses the one-flow pattern (search.Open + Rebuild) | test green in `internal/web` |
| **CI cqrs-lint lane**: new `cqrs-lint` job pins the same flake rev (`afaffa19`), runs `cqrs-lint .` after templ generation; errors fail, documented warnings live in docs/cqrs-lint-policy.md | `.github/workflows/ci.yml` |
| **Docs closure**: TODO_LIST 7 rows flipped to DONE with fresh evidence; DOMAIN_LANGUAGE: Excerpt row updated to `calls.Excerpt` semantics + new `TurnRole` + `Age gate` rows; AGENTS.md dependency doctrine + excerpt wording; CHANGELOG 6 new entries; FEATURES age-gate/auth rows updated | files on disk |
| **P9 read-only upstream check**: all five D6 filings verified OPEN via gh api (go-cqrs-lite#62/#63/#64, go-nix-helpers#7, treefmt-nix#545); state stamped into docs/upstream-filings.md; nothing posted | gh api output 20:0x |
| **Standalone `nix build` verified green** after the go.mod change: `result/bin/pbx-lens version` works — the vendor hash SURVIVED the x/time promotion (module tree unchanged), confirming the gate's nix-build failure was resource starvation, not a hash break | build output + binary runs |

Suite state at 20:26: **build green, 13/13 packages, 129 test functions
(was 127), gofmt + vet clean.**

## b) PARTIALLY DONE

| Item | Works | Open | Effort |
| --- | --- | --- | --- |
| **Full buildflow gate** | Last run: every step green EXCEPT `nix-build`, killed with "OOM or timeout"; the documented casualty-verification ran clean (`nix build` alone → green binary), so the failure is machine-load starvation (docs/buildflow-dev-gate.md pattern), not code | One clean full-gate rerun needed to close the loop (ideally at lower machine load; ~10 min) | S |
| **P9 filings** | All five verified open; drafts sit in docs/drafts/ | Posting/commenting needs your go (github-voice flow + your auth) | S per filing |
| **Cross-project lesson** ("read existing scaffolding before writing test wiring" — the 503 probe miss) | Identified | Belongs in crush-config references/lessons.md, by commit — cross-repo write needs a session on that repo | S |

## c) NOT STARTED (still open from the 19:37 report)

P10 (BuildFlow binary rebuild), production `-max-age` window in the sibling
NixOS module, first release tag (v0.1.0), auth htpasswd reload-on-change,
per-IP rate limiting (global bucket shipped), poller ops-metrics endpoint,
slog adoption, README screenshots, keyset-pagination probe, P13/P14/P15/P17
(named external blockers). Full ranked list: 19:37 report §f, minus this
dispatch's completed rows.

## d) TOTALLY FUCKED UP

1. **I skipped the reuse check you then ordered.** P6's fsync chain was
   hand-rolled with a house library (`go-atomic-write`) sitting in
   `/home/lars/projects/` that I never looked for. The save: source-reading
   it took one command and the verdict is defensible (API cannot own a
   sqlite-written staging file) — but the check should have happened BEFORE
   writing `syncFile`/`syncDir`, not after your pointer. This is the
   repeat offender: same class as the P12 "already annotated" discovery
   later in this dispatch.
2. **Two compile-broken edit batches** (discipline from last report
   violated again): (a) poller.go lost the callback's `, nil` return and
   kept an unused import when collapsing onto `AgeGate`; (b)
   server_test.go needed `context` + `filepath` + `search` imports I
   appended without checking. Both caught by immediate builds (the
   compile-after-batch habit DID work), but they were predictable.
3. **The e2e probe hit the 503 I designed.** I wrote `Options{}` into the
   probe although I knew nil Search answers 503 by design; flow_test.go
   had the correct wiring 60 lines away. Search-first would have saved a
   cycle. (Now fixed by literally reusing that pattern.)
4. **One test-logic bug**: the GateIngest test left `fresh` on the shared
   helper's fixed 2026-10-09 timestamp — over 24h old, so both rows got
   refused. Caught by running the test (as intended), fixed by making the
   fresh timestamp explicit.
5. **One stale-read edit rejection** (server_test.go): edited without
   re-reading after an external modification; re-read and re-applied. No
   damage, but it is the exact mechanical failure the editing doctrine
   warns about.

## e) WHAT WE SHOULD IMPROVE

1. **Institutionalize the sibling-library check**: before hand-rolling
   anything durability/atomicity/parsing-shaped, grep
   `/home/lars/projects/*/README.md` for the concept. Candidate: a
   crush-config lesson + a line in the project AGENTS.md ("sibling repos
   are the first dependency search path").
2. **Gate starvation is now a recurring paid-for cost** (two dispatches in
   a row). Mitigations to decide between: run the gate at low load, teach
   buildflow a `--max-jobs`/serial-nix mode, or split nix-build into its
   own gate step that retries once.
3. **Test scaffolding inventory**: flow_test.go's one-flow wiring
   (index.Open → Rebuild → newTestServerWith) is now reused by TWO tests;
   a `newTestServerWithSearch(t, observations...)` helper would remove the
   copy from the next consumer.
4. **The upstream drafts folder needs a decision loop**: five D6 drafts +
   one go-atomic-write draft now sit with unchecked owner boxes; a weekly
   "post or drop" pass would stop them fossilizing.

## f) Top next tasks (harvest ground; deltas vs 19:37 §f marked)

| # | Task | Impact | Effort | Category |
| --- | --- | --- | --- | --- |
| 1 | **Rerun the full buildflow gate at low load**; expect 0 errors; then retire the gate-pending note in this report | High | S | Quality |
| 2 | **Decide the go-atomic-write `Commit` ask**: post the issue yourself or have me file it (github-voice) | Medium | S | Feature (house lib) |
| 3 | Extract `newTestServerWithSearch` helper; collapse the two duplicate index wirings in web tests | Low | S | Cleanup |
| 4 | Sibling NixOS module: pick + set the production `-max-age` window (needs owner value; e.g. 720h vs 2160h) | High | S | Feature |
| 5 | First release tag v0.1.0 via go-release (CHANGELOG ready; gate green is the precondition) | Medium | M | Feature |
| 6 | P10: rebuild BuildFlow binary (5aaf336 → HEAD), re-run gate, delete the stale license-check note | Low | S | Cleanup |
| 7 | P9: post the five D6 filings after your review (drafts ready; all verified open today) | Medium | M | Quality |
| 8 | Post-or-drop pass over docs/drafts/ (six drafts, all owner-gated) | Low | S | Documentation |
| 9 | Add `ingest-* -max-age` to the sibling module's ExecStart options alongside serve's | Low | S | Feature |
| 10 | Auth: htpasswd reload on SIGHUP/mtime (restart required today) | Low | M | Feature |
| 11 | Auth: per-IP limiter upgrade if the PBX ever leaves loopback-only trust | Low | M | Feature |
| 12 | Poller ops endpoint exposing per-source rejection counters (today: log lines only) | Low | M | Feature |
| 13 | slog adoption for poller/serve logs (std log.Printf today) | Low | M | Quality |
| 14 | README UI screenshots of the four surfaces | Low | S | Documentation |
| 15 | Keyset-pagination probe beyond page 1 with >page-size data | Low | S | Quality |
| 16 | templ-components upgrade watch (SegmentBar unblock = P17) | Low | S | Feature |
| 17 | Live Telnyx DTO verification when an API key exists (P13, quarterly cluster) | Medium | S | Quality |
| 18 | ContentHash v2 length-prefixed encoding at schema v2 (P14, owner-scheduled) | Low | M | Feature |
| 19 | Recording-proxy timeout knob at traffic-data time (P15, owner-deferred) | Low | S | Feature |
| 20 | lessons.md (crush-config, by commit): "sibling repos are the first dependency search path" + "wire tests from existing scaffolding" | Low | S | Documentation |

## g) Top questions I can NOT figure out myself

1. **go-atomic-write is YOUR repo**: do you want the `Commit`-primitive
   issue filed as drafted (docs/drafts/go-atomic-write-commit-primitive.md),
   a PR with the exported function instead, or will you design the API
   yourself? (The draft's shape: `Commit(target, stagingPath string)`.)
2. **Gate scheduling**: the nix-build step has now starved twice under
   machine load. Should gates run only at idle times, or do you want me to
   add a serial/low-parallelism mode to the BuildFlow invocation in this
   repo's AGENTS.md commands?
3. **Release gate**: is "one clean full-gate rerun + your explicit go"
   the release precondition for v0.1.0, or do you want the sibling repo's
   `-max-age` window decided and deployed first so the module ships
   complete?

Waiting for instructions.
