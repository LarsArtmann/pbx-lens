# Status Report: pbx-lens — backend hardening, C2/C4/D5 landed, D3 MOS plumbing (2026-10-11 09:39 CEST)

> Scope: this session's run only (roughly 05:30–09:40 CEST 2026-10-11). A
> parallel session was ALSO committing to this repo during the same window
> (frontend/dashboard/systemd; see `5e31e41`, `2026-10-11_09-35_...`), so the
> tree now contains both sessions' work. This report attributes only what THIS
> session did, and flags the overlap explicitly.
>
> Session start: resuming the pbx-lens TODO mandate after a paused C2 (poller
>
> - `/api/freshness`), with 2 of 4 poller tests RED on a test-fixture bug.

---

## a) FULLY DONE

| Item                                              | Evidence                                                                                                                                                                                                                                                                                                                                                                                                                                                            |
| ------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Poller fixture bug fixed (the resume blocker)** | Root cause: the inline 2-line webhook fixture in `internal/cli/poller_test.go` carried 5 closing braces where the real `/tmp/lens-e2e.jsonl` has 4 → malformed JSON → `webhook.Parse` returned `obs=0 skipped=2`. Removed the extra brace in both tests. `go test ./internal/cli` green.                                                                                                                                                                            |
| **Post-x/crypto hygiene**                         | `go mod tidy` (no drift), `nix run .#update-vendor-hash` → landed `sha256-Z4Bpc17yPphQHnbryUD69nPiCM+128hT52NSfywnw2A=`, `nix build` green, `./result/bin/pbx-lens version` → `pbx-lens 0.1.0`.                                                                                                                                                                                                                                                                     |
| **golangci-lint: 16 findings → 0**                | Fixed real code smells in the session's own files: `auth.go` dummy-hash global removed (computed once in the middleware constructor), `auth_test.go` rewritten to `httptest.ResponseRecorder` (removes 5 `bodyclose`), `PollOnce` split into `pollCDR/pollWebhook/pollTranscripts/pollTelnyx` helpers (drops cognitive complexity 27→<25), `DefaultPollSince` const added, `golines` line wraps, `wsl_v5` blank lines. `buildflow -s golangci-lint` → **0 issues**. |
| **cqrs-lint: 0 errors**                           | 15 WARNING/INFO remain (E005×2, D014, F026, F007, B004, B005, F009); each is a documented deliberate non-fix. Gate (ERROR severity) passes.                                                                                                                                                                                                                                                                                                                         |
| **Dev quality gate PASSES with warnings**         | `nix develop -c buildflow --build-mode dev` → **70 success, 0 failed, 0 skipped**; golangci-lint 0 issues. (Running it OUTSIDE the devShell is a false failure — see d1.)                                                                                                                                                                                                                                                                                           |
| **C2 e2e verified live**                          | `serve -poll-every 2s -webhook-log <fixture>` ingests on startup and every tick; `/api/freshness` returned `{"age_seconds":12.3,"stale":false,"calls_indexed":1}` for a fresh call and `stale:true` for an old one (6h threshold correct); `/search?q=+17287289311` returned the polled call → poller rebuilds FTS. Second pass replayed to the same `calls_indexed=1` (idempotent).                                                                                |
| **C4 backup shipped**                             | New `internal/backup` package: `Snapshot` (sqlite `VACUUM INTO` via temp file + atomic rename, so a failed run never destroys the last good snapshot) and `Verify` (`PRAGMA integrity_check`). New `pbx-lens backup -db <dsn> -out <path>` verb. 4 tests green. Live e2e: snapshot written+verified, overwritable, contains `lens_search` (1 doc) + `meta_stream_log` (1 event) + `meta_map` (2 rows).                                                              |
| **D5 export shipped**                             | New `internal/web/export.go`: `GET /api/export.jsonl` (ndjson, one `CallView` per line) and `GET /api/export.csv` (stable header + rows), both honouring the list filters via a shared `listOptions`/`ExportCalls` path. 4 tests green. Live e2e both endpoints returned the polled call.                                                                                                                                                                           |
| **`ListCalls` refactor**                          | Extracted `listOptions(query, limit)` so the list page and the export share the exact filter bindings (the RFC3339Nano time-binding fix stays in one place).                                                                                                                                                                                                                                                                                                        |
| **MOS plumbing (D3/m65 groundwork)**              | `Observation.MOS float64` added; included in `ContentHash` ONLY when non-zero (so sources without MOS keep their historical hashes); projected into `CallView.MOS`; CDR parser reads an optional appended `rtp_audio_in_mos` column (guarded to the 1..5 scale; a non-MOS extra column degrades to 0). 2 CDR tests + all existing tests green.                                                                                                                      |
| **AGENTS.md memory updates**                      | New gotchas: run the dev gate INSIDE the devShell; the `tagliatelle.case.use-case` removal; `go.mod` floor restore; the new cqrs-lint non-fixes (F009 poller ticker, D014, F026, B004/B005).                                                                                                                                                                                                                                                                        |
| **All packages green**                            | `go test ./... -count=1` → 9 packages ok, 0 failures.                                                                                                                                                                                                                                                                                                                                                                                                               |

---

## b) PARTIALLY DONE

| Item                           | What exists                                                                                                                                                                                                         | What's missing                                                                                                                                                                                                                   |
| ------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **D3 dashboard (m64/m65/m66)** | A PARALLEL session landed `internal/web/dashboard.go` (`/dashboard`: totals, today, AI share, avg duration, 30-day volume series) + route + page. THIS session added the MOS data path (Observation→CallView→hash). | The dashboard does **not yet render a MOS panel/trend** from `CallView.MOS`; the MOS series isn't aggregated. m65 is half-landed.                                                                                                |
| **D3 support assumption**      | CDR parser accepts an appended MOS column.                                                                                                                                                                          | The sibling deployment's `cdr_csv` template is the default 13-field set (confirmed in `nix-international-telephony/api.py`), so no MOS reaches prod until the sibling adds `rtp_audio_in_mos`. Unverified against real prod CDR. |
| **C5 insights / C6 cost**      | The data model is ready (`Enrichment{CostMicros,CostCurrency,InsightSummary}` + `call.enriched` + `EnrichCallCmd` + `CallView` fields).                                                                             | The Telnyx fetch (conversation insights endpoint; Detail Records cost) is not implemented; no live probe run this session.                                                                                                       |
| **Dev-gate cleanliness**       | Gate passes with warnings inside the devShell.                                                                                                                                                                      | A parallel-authored `internal/web/dashboard.go:58` has 1 `mnd` (magic 24) finding — left untouched (not my file, possibly still in flight).                                                                                      |

---

## c) NOT STARTED

- **C5** T13 conversation insights fetch + store + detail summary card.
- **C6** T15 cost column via Detail Records poller (lag-tolerant, nullable).
- **D2** T23 retention + PII policy purge (E164/ParticipantHash become load-bearing).
- **C3** T7 systemd unit + thin NixOS module in the sibling repo.
- **D6** C009 upstream filing (verify-before-filing) — plus the other four upstream items queued (sqliteengine time binding, memory-engine column naming, go-nix-helpers templ formatter goPkg, treefmt-nix templ `pkgs.go`, cqrs-lint notes).
- **D4** final docs sync (README/CHANGELOG/FEATURES/ROADMAP), full-mode gate, prod re-ingest.
- **MOS dashboard panel** (aggregate `CallView.MOS` by day; show avg MOS).
- **CDR/webhook/telnyx parsers** do not set MOS (only CDR does) — intentional.

---

## d) TOTALLY FUCKED UP (honesty section)

1. **Chased the dev gate outside the devShell.** Ran `buildflow --build-mode dev` from a bare shell; `go-generate`/`workspace-build-verify` failed with “requires go >= 1.27.1 (running go 1.27.0)”. I spent real time diagnosing before realising the correct invocation is `nix develop -c buildflow …` (devShell ships go 1.27.2, `GOTOOLCHAIN=local`). Captured in AGENTS.md so it never costs this again.
2. **A dev-gate run rewrote `go.mod` to `go 1.27`**, after which `golangci-lint` reported “no go files to analyze”. I restored `go.mod` via `git restore`; the directive is dep-forced by go-appkit v0.8.0.
3. **`.golangci.yml` had an invalid key (`tagliatelle.case.use-case`)** that made the golangci runner terminate with “configuration contains invalid elements”. It predated this session, but I only found it by reading the dev-gate log carefully; removed it.
4. **The original poller test failure was a stray brace**, not the “subtle quoting/content difference” the prior report hypothesised. I should have byte-diffed the fixture immediately instead of trusting the hand-written diagnosis.
5. **Left `internal/ingest/cdr/cdr_test.go` with unused MOS fixtures for a few minutes** before adding the test functions — a clean tree should never have dead consts (staticcheck `unused` would have flagged them).
6. **LSP diagnostics were noise.** Repeatedly the editor LSP reported phantom errors (`no go.sum entry`, `undefined: render`, `unused exportColumns`) that the real build disproved. I burned cycles mentally filtering them; the rule “trust only build/templ-generate/buildflow” held.
7. **Did not reconcile with the parallel session early.** I noticed its commits only near the end; the dashboard overlap (their D3 vs my MOS groundwork) could have been coordinated sooner.

---

## e) WHAT WE SHOULD IMPROVE

- **Coordinate parallel sessions explicitly.** Two sessions editing pbx-lens in the same window produced overlapping D3 work; a quick `git log` at session start would have shown the other session's scope.
- **Never run the gate outside the devShell.** Make a `scripts/gate.sh` wrapper (`nix develop -c buildflow --build-mode dev`) so the correct command is the only command.
- **Byte-diff fixtures before theorising.** For parser-vs-fixture mismatches, compare `%q` output first.
- **Keep MOS honest.** The dashboard MOS panel must show “no data” clearly until real prod CDR carries `rtp_audio_in_mos`; don’t fabricate trends.
- **Wire the MOS panel now** (the data path exists; the UI is one aggregation + one template block).
- **Add a domain test asserting hash stability** for `MOS == 0` (protects against a future accidental format change mass-conflicting all streams).
- **Land the upstream filings** — several verified findings are sitting undocumented in-repo (sqlite time binding, memory-engine column naming).
- **Tighten the big-literal test style.** `TestExportJSONL…` etc. are fine, but the poller fixture duplication should become a shared `testdata` file.
- **Reduce AGENTS.md size** — preflight warns it exceeds the 220-line budget; move the cqrs-lint policy to `docs/`.
- **Decide the E003 flat-domain split** (owner question still open).
- **Run the C5/C6 live Telnyx probe** before writing code, as done for `ingest-telnyx`.

---

## f) UP TO 50 THINGS TO GET DONE NEXT

1. ~~Wire the MOS panel into the dashboard (aggregate `CallView.MOS` by day; avg per day; “no data” empty state).~~ done — landed (m65 MOS card, 11:48 continuation)
2. ~~Add a domain test: `ContentHash` unchanged when `MOS == 0`, changed when `MOS != 0`.~~ done — landed (TestContentHashMOSOnlyWhenReported)
3. Move the poller webhook fixture into a shared `testdata/` file. **→ ROADMAP theme 2 (test hygiene)**
4. ~~Fix `internal/web/dashboard.go:58` `mnd` magic 24 (coordinate with the parallel author or take it as covered work).~~ done — landed (utcDay const, 10:28 session)
5. ~~C5: add a Telnyx conversation-insights fetch to the client + tests.~~ done — landed (11:31 session)
6. ~~C5: store insights on ingest; render a detail summary card.~~ done — landed (11:31 session)
7. ~~C5: live GET-only probe to learn the real insights response shape; record findings in AGENTS.md.~~ done — landed (11:31 session)
8. ~~C6: Detail Records poller skeleton (lag-tolerant, nullable cost).~~ done — landed (11:31 session)
9. ~~C6: cost join onto the view + nullable display in list/detail/export.~~ done — landed (11:31 session)
10. ~~C6: live probe for Detail Records shape/pagination.~~ done — landed (11:31 session)
11. ~~C6: currency handling (CostCurrency) and a “cost unknown” UI state.~~ done — landed (11:31 session)
12. ~~D2: retention/purge job with a `-retention` policy flag.~~ done — landed (11:48 D2)
13. ~~D2: purge tests (age cutoff; participant-hash-only export mode).~~ done (docs-health pass 2026-10-12)
14. D2: make E164/ParticipantHash load-bearing in the purge path. **→ ROADMAP theme 4 (stored-row enforcement)**
15. ~~C3: systemd unit draft (DynamicUser, `ReadWritePaths=`, `sd_notify` already documented by the parallel session).~~ done — landed (11:48 C3)
16. ~~C3: thin NixOS module option draft for the sibling repo.~~ done — landed (11:48 C3)
17. ~~C3: sibling handoff note.~~ done — landed (11:48 C3)
18. ~~D6: verify the cqrs-lint C009 exemption at source, then file the issue.~~ done — filed (11:48: #62/#63/#64, #7, #545)
19. ~~D6: file the sqliteengine time-binding issue (with `web.rfc3339Nano` as the workaround).~~ done — filed (11:48: #62/#63/#64, #7, #545)
20. ~~D6: file the memory-engine column-naming issue.~~ done — filed (11:48: #62/#63/#64, #7, #545)
21. ~~D6: file the go-nix-helpers templ-formatter goPkg issue.~~ done — filed (11:48: #62/#63/#64, #7, #545)
22. ~~D6: file the treefmt-nix templ `pkgs.go` issue.~~ done — filed (11:48: #62/#63/#64, #7, #545)
23. ~~D4: README sync (features, commands including `backup`/export/dashboard).~~ done — README serve-era sections landed (09:35 F22 + later)
24. ~~D4: FEATURES.md reconciliation (dashboard DONE, export DONE, insights/cost PARTIAL).~~ done — rows updated across the 11:31-11:48 sessions
25. ~~D4: TODO_LIST.md reconciliation (mark B3/C2/C4/D5 done).~~ done — rows flipped across the 20:26/23:xx dispatches
26. ~~D4: ROADMAP.md graduation note for the export API.~~ done — 2026-10-12 rebuild graduated the shipped ideas
27. ~~D4: CHANGELOG entry for this window.~~ done — serve-era entries landed
28. ~~Full-mode `buildflow` (not just dev) green.~~ done — first green full-mode gate 2026-10-12 00:45 (01:16 report)
29. Prod re-ingest dry run (CDR + webhook + telnyx) once the sibling MOS column is decided. **→ TODO_LIST blocked row (needs prod deploy path)**
30. ~~Add `nix run .#test` / `.#lint` aliases if the flake lacks them.~~ done — module-generated apps.test/apps.lint deliberately kept (AGENTS.md)
31. ~~Add a CI lane matching the devShell gate (blocked on owner Q1 remote).~~ done — landed 2026-10-11; green on main
32. ~~Decide CI visibility/push (owner Q1).~~ done — public; pushed (10:28 session)
33. ~~Ratify `call.corrected` supersession vs conflict-rejection (owner Q2).~~ done — supersession doctrine documented (AGENTS.md gotchas)
34. ~~Confirm the §30 strike in `~/projects/pbx-artmann/TODO_LIST.md` (owner Q3).~~ **Won't implement — owner-gated cross-repo edit; pbx-lens TODO_LIST is the canonical backlog.**
35. ~~Add an integration test that runs the poller against a real sqlite DSN (not memory).~~ done — poller + age-gate tests run on real sqlite
36. Add `/api/freshness` to the dashboard header (reuse the JSON endpoint client-side). **→ ROADMAP theme 2**
37. Export: add an explicit `-limit`/paging doc for very large dumps. **→ ROADMAP theme 1**
38. ~~Export: decide whether JSONL/CSV should stream (bounded memory) instead of buffering.~~ done (docs-health pass 2026-10-12)
39. ~~Backup: add a `-verify-only` mode for the nightly staging check.~~ done (docs-health pass 2026-10-12)
40. ~~Backup: wire `backup` into the sibling nightly sync (m48).~~ done — module backup timer landed (11:48 C3)
41. ~~Search: add a test for the FTS rebuild after a poller pass on a real sqlite file.~~ done — poller tests assert the rebuild
42. ~~Reduce AGENTS.md; move the cqrs-lint policy to `docs/lint-policy.md`.~~ done — landed (P1, 19:37 dispatch)
43. ~~Add a `scripts/gate.sh` wrapper that always enters the devShell.~~ **Won't implement — doctrine documented in AGENTS.md/buildflow-dev-gate.md instead.**
44. Remove the em-dash from any remaining source strings (sweep). **→ convention enforced going forward (no-em-dash rule)**
45. Add a golden test for `/api/export.csv` header stability. **→ ROADMAP theme 2**
46. ~~Add `mnd`-clean constants for all dashboard magic numbers.~~ done — landed (dayFormat/utcDay consts)
47. ~~Re-check `go.mod` after every dev-gate run (guard against the downgrade).~~ done (docs-health pass 2026-10-12)
48. ~~Document the MOS column assumption + sibling requirement in AGENTS.md.~~ done — documented in FEATURES MOS rows + this report
49. ~~Add a test that a second `backup` run onto an existing path replaces atomically.~~ done — backup tests cover overwrite + unique staging (20:26)
50. ~~Write the resume checklist into `docs/planning/` so the next session starts grounded.~~ done — this report series is the checklist; plans in docs/planning/

---

## g) QUESTIONS I CANNOT ANSWER MYSELF

1. ~~**Do the two parallel sessions now conflict on scope?** The dashboard (D3) was landed by another session while I was mid-flight on MOS. Should I stop touching `internal/web/dashboard.go` and hand the MOS panel to that session, or take it over? I cannot see the other session's live intent.~~ done — converged cleanly; both sessions' work landed and is verified in the tree
2. **Is `rtp_audio_in_mos` actually configured on the live PBX**, and should I add it to the sibling `cdr_csv` template now (C3 territory), or leave the CDR parser’s MOS path dormant until you confirm the deployment change? I cannot verify the live PBX state. **→ owner/deploy-side decision (sibling cdr_csv template)**
3. ~~**Does the owner want the index-disposable rehash accepted** if MOS ever enters ContentHash for existing rows? I kept `MOS == 0` hash-stable deliberately, but the general “schema change = one-time re-derive” policy is yours to ratify.~~ done — ratified: disposable-DB rebuild doctrine (AGENTS.md)

---

_Generated 2026-10-11 09:39 CEST. This session touched: `internal/cli/poller_test.go`, `internal/cli/poller.go`, `internal/web/auth.go`, `internal/web/auth_test.go`, `internal/web/export.go`, `internal/web/export_test.go`, `internal/web/queries.go`, `internal/web/server.go`, `internal/backup/*`, `internal/calls/domain.go`, `internal/calls/projection.go`, `internal/ingest/cdr/cdr.go`, `internal/ingest/cdr/cdr_test.go`, `cmd/pbx-lens/main.go`, `.golangci.yml`, `AGENTS.md`._
