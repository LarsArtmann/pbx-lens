# Status Report — Dependency QA Session (templ-components question + CLI surface verification)

**Date:** 2026-10-11 01:05 CEST
**Session scope:** Q&A / investigation only. **Zero code written, zero files modified, zero tests or builds executed by this session.**
**Method:** 3 read-only investigations (go.mod, repo-wide grep for UI/HTTP/templ, TODO_LIST/FEATURES/ROADMAP cross-check, CLI `main.go` inspection).
**Series:** follows `2026-10-10_23-56_docs-health-audit-first-doc-set.md` (different scope: that one audited docs; this one answers a dependency question).
**Format note:** written as `.md` per explicit user instruction — the status-report skill's canonical format is a styled HTML dashboard; user instruction wins, flagged here so the divergence is visible.

---

## Self-Review (asked first: "What did you forget? What could you have done better? What could you still improve?")

### What did I forget?

1. **I forgot to RUN anything.** I asserted "Engine done, eyes not built" purely from reading code and docs. `go run ./cmd/pbx-lens version` and `smoke` are one command each and would have upgraded the claim from code-reading to evidence. Given that the docs (which I had just read) record the nix artifact path as broken, verifying the binary by hand was cheap insurance I skipped.
2. **I forgot to answer the question behind the question.** "Why don't we use templ-components??" plausibly signaled "should we start the UI now?" I answered factually and stopped. The exploration-mode answer deserved a proposal: "want me to scaffold T6 (serve + list page) on templ-components?" I never offered the next step.
3. **Precision slip:** my grep hit list included `internal/ingest/telnyx/telnyx.go` — that IS HTTP code, just an HTTP _client_ (Telnyx API), not a server. I said "no HTTP" loosely when I meant "no HTTP server/routes."
4. **Unverified future claim:** I called templ-components "the obvious pick" without checking its fit against this repo's constraints (templ codegen step vs buildflow/flake, Go 1.27 + `GOEXPERIMENT=jsonv2`, vendor-hash implications). The sentiment is probably right; the verification was not done and should not be treated as fact.

### What could I have done better?

- **Evidence over assertion.** One `go run` + one `go test` would have caught the overstatement below (d.2) before the user did.
- **Recommendation, not just inventory.** The dependency question was an opening to propose the highest-impact next work; I delivered a status answer instead of a plan.

### What could I still improve?

- Covered in section (e) — the short version: make run-the-thing the default close of every "what does this do" question.

**Did I lie to you?** No lie, but one **overstatement**: "Engine done." See (d.2). The engine has a documented High-severity correctness bug I had already read when I said it.

---

## a) FULLY DONE

_(This session produced no code, so "done" here means: verifiably established this session.)_

| #   | What                                                                                                                                                                                                | Evidence                                                                                                                      | Scope                  |
| --- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------- | ---------------------- |
| a.1 | Dependency-policy verification: direct deps are go-cqrs-lite modules + go-error-family ONLY — matches AGENTS.md policy; templ-components absent                                                     | `go.mod` require block read this session                                                                                      | `go.mod`               |
| a.2 | CLI surface inventory: `version \| smoke \| ingest-cdr \| ingest-telnyx`                                                                                                                            | `cmd/pbx-lens/main.go:27-36`                                                                                                  | `cmd/pbx-lens/main.go` |
| a.3 | Confirmed zero UI surface: no `.templ` files, no http server routes, no templ/HTMX/Tailwind imports anywhere                                                                                        | repo-wide grep (`templ\|http\.\|html\|ui`) — only hits are CLI usage text, FreeSWITCH parser, Telnyx HTTP client, domain code | whole repo             |
| a.4 | Corrected the user's "does nothing?!" with the accurate picture: working CLI index engine (CDR parse, Telnyx pull+recording correlation, ContentHash-deduped sqlite index, `call_views` read model) | FEATURES.md:37 (FreeSWITCH parser FULLY_FUNCTIONAL), AGENTS.md Telnyx live-API verification note, cmd inventory               | docs cross-check       |
| a.5 | Zero working-tree damage; no commits authored by this session                                                                                                                                       | git status clean at session start; only my writes were none (this report excepted)                                            | —                      |

## b) PARTIALLY DONE

| #   | What                                                                     | Works now                                                                                                                | Open                                                                                                                         | Effort  |
| --- | ------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------ | ---------------------------------------------------------------------------------------------------------------------------- | ------- |
| b.1 | **The ingest engine itself** (correction of my in-session "engine done") | CDR parser + Telnyx ingest + dedup + projection are shipped and tested at unit level (FEATURES.md, AGENTS.md test notes) | High-severity excerpt order-dependence bug (d.2); no e2e wiring test for the ingest-telnyx loop (TODO_LIST:35); no UI on top | see (f) |
| b.2 | This session's deliverable                                               | Two dependency/architecture questions answered with evidence                                                             | No artifact produced until this report; no follow-up action taken                                                            | —       |

## c) NOT STARTED

_(All read from TODO_LIST/FEATURES/ROADMAP/AGENTS.md this session; nothing beyond planning exists for any of these.)_

| #   | What                                                                                                       | Why not started                                                                     | Still wanted?                 |
| --- | ---------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------- | ----------------------------- |
| ~~c.1~~ | ~~T6 `serve` command + calls list page (templ-components is the planned but unadopted library)~~ done (docs-health pass 2026-10-12) | ~~CLI-first build order; UI is the "lens" half of the project, deferred~~ | ~~Yes — completes the 51% slice~~ |
| c.2 | T8 per-call detail page + recording streaming proxy **→ ROADMAP theme 1 (export streaming)** | depends on T6 routes                                                                | Yes                           |
| ~~c.3~~ | ~~T12 poller timers + freshness self-check / stale banner~~ done (docs-health pass 2026-10-12) | ~~ingest is CLI-invoked only; no scheduler code anywhere~~ | ~~Yes~~ |
| ~~c.4~~ | ~~T24 UI auth (operator htpasswd reuse)~~ done (docs-health pass 2026-10-12) | ~~depends on T6~~ | ~~Yes~~ |
| c.5 | webphone read-only deep link into a call view **→ ROADMAP theme 3 (sibling side)** | depends on T6/T8; boundary: webphone links in, never a data dependency (ROADMAP:47) | Yes                           |
| c.6 | PII policy: `E164` + `ParticipantHash` exist but are NOT enforced on stored fields (AGENTS.md domain note) **→ ROADMAP theme 4 (stored-row enforcement)** | policy decision pending                                                             | Yes (documented in TODO_LIST) |
| ~~c.7~~ | ~~T9 agent-transcript `SourceKind` ingest~~ done (docs-health pass 2026-10-12) | ~~fourth source kind, not begun~~ | ~~Yes~~ |
| ~~c.8~~ | ~~CI lane (GitHub Actions vs buildflow-only)~~ done — landed and green on main (2026-10-11) | ~~BLOCKED on owner decision (TODO_LIST:49)~~ | ~~Yes, pending decision~~ |
| ~~c.9~~ | ~~templ-components adoption work itself (go.mod entry, templ codegen in buildflow/flake, vendor hash)~~ done — adopted as the UI trio (go-appkit + templ-components + templ), 2026-10-11 | ~~no UI to render yet~~ | ~~Yes — this session's topic~~ |

## d) TOTALLY FUCKED UP

1. **`nix build` ships an EMPTY result.** Exits 0, produces no `bin/pbx-lens`; `nix run . -- version` fails. `flake.nix` even sets `mainProgram = "pbx-lens"` — the go-standard flakeModule is not wired to the `cmd/pbx-lens` binary. Severity: **blocks the entire deploy story** — the sibling repo's planned "thin NixOS deploy module" has nothing deployable. Workaround: `GOEXPERIMENT=jsonv2 go build ./...` locally. (Reproduced 2026-10-10 per FEATURES.md:69 / TODO_LIST:34 — documented, not re-reproduced by this session.) Root cause: flakeModule/package wiring; fix is TODO_LIST High 30-60min.
2. **My own claim "Engine done, eyes not built" was overstated — and the evidence was in the file I'd just read.** `internal/ingest/telnyx/map.go:70` `excerpt()` takes the API's return order (newest-first); indexed excerpts are order-dependent and can be silently wrong (TODO_LIST:30, High, report 2026-10-10 §d.1). A "lens" whose indexed content is order-dependent is not done; it is partially done with a known correctness bug. Severity: wrong data displayed once UI exists. Mitigation: none yet; semantics undecided (see question g.2).
3. **The documented lint workflow is not reproducible from the flake.** cqrs-lint is built to `/tmp` per AGENTS.md and dies at reboot (TODO_LIST:46). Severity: quality gate degrades silently for any fresh session/machine. Workaround: rebuild manually.
4. _(Session-level, honest)_: **This session itself produced nothing durable.** Two questions answered; if the user had stopped after message 2, the only residue would be chat scrollback. Not "fucked up" at project level — but zero deliverable-until-now is the truth of the ledger.

## e) WHAT WE SHOULD IMPROVE

| #   | Improvement                                                                                                                           | Impact                                      | Concrete fix                                                                                         |
| --- | ------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------- | ---------------------------------------------------------------------------------------------------- |
| e.1 | **Verify by running, not reading.** Reading code answers "what exists"; running answers "what works." This session conflated them.    | High — wrong confidence handed to the owner | Standing rule: any "this works/done" claim requires executing the binary or test in the same session |
| e.2 | **Answer the intent, not just the question.** "Why don't we use X?" often means "should we?"                                          | Medium — saves a round trip                 | Close exploration answers with one concrete recommendation + offer to execute                        |
| e.3 | **Keep FEATURES.md's honest UI section** ("UI (none shipped)") — it is exactly what let this session correct itself fast              | preserve                                    | —                                                                                                    |
| e.4 | Precision in language: HTTP client ≠ HTTP server; "no UI" ≠ "no HTTP"                                                                 | Low but compounding                         | —                                                                                                    |
| e.5 | When a High-severity bug is read in passing (excerpt bug), surface it immediately instead of leaving it to a status report to confess | Medium                                      | Mention known-broken things the moment they're touched in a session                                  |

## f) Top 40 things to get done next

_Ranked by impact. Items 1–10 are the commitment core; 11–40 are brainstorm fuel for docs-health HARVEST — most decompose the documented T6/T8/T12 plans or are verification steps. Per the status-report skill: a list this long is a brainstorm, not a commitment list._

| #  | Task                                                                                                                                       | Impact   | Effort       | Category      |
| -- | ------------------------------------------------------------------------------------------------------------------------------------------ | -------- | ------------ | ------------- |
| ~~1~~  | ~~Fix nix package: wire flakeModule output to `cmd/pbx-lens`; verify `nix run . -- version` works~~ done (docs-health pass 2026-10-12) | ~~Critical~~ | ~~M (30-60min)~~ | ~~Bug~~ |
| ~~2~~  | ~~Fix excerpt order-dependence: define semantics, implement, pin with test (`internal/ingest/telnyx/map.go:70`)~~ done (docs-health pass 2026-10-12) | ~~Critical~~ | ~~S (15min)~~ | ~~Bug~~ |
| ~~3~~  | ~~Run the verification battery this session skipped: `go build`, `go test`, binary `version`+`smoke`, artifact check~~ done (docs-health pass 2026-10-12) | ~~High~~ | ~~S~~ | ~~Quality~~ |
| 4  | ~~Add artifact-level guard: after `nix build`, assert `result/bin/pbx-lens` exists (kills the silent-empty-exit-0 class)~~ done — artifact checks are standard practice (AGENTS.md; 04:58+ sessions) | High     | S            | Quality       |
| ~~5~~  | ~~T6: implement `serve` command skeleton (http.Server, addr/db flags)~~ done (docs-health pass 2026-10-12) | ~~High~~ | ~~M~~ | ~~Feature~~ |
| ~~6~~  | ~~T6: adopt templ-components in go.mod; wire `templ generate` into buildflow/flake; update vendor hash via `buildflow -s nix-hash-fix --fix`~~ done (docs-health pass 2026-10-12) | ~~High~~ | ~~M~~ | ~~Feature~~ |
| ~~7~~  | ~~T6: calls list page — one table, all calls (templ-components)~~ done (docs-health pass 2026-10-12) | ~~High~~ | ~~M~~ | ~~Feature~~ |
| 8  | T6: AI badge + excerpt column **→ ROADMAP theme 2 (public presence)** | High     | S            | Feature       |
| ~~9~~  | ~~T6: direction/date filters (plan M18-M20)~~ done (docs-health pass 2026-10-12) | ~~High~~ | ~~S~~ | ~~Feature~~ |
| ~~10~~ | ~~T6: httptest smoke test for the list page~~ done (docs-health pass 2026-10-12) | ~~High~~ | ~~S~~ | ~~Quality~~ |
| ~~11~~ | ~~httptest end-to-end test for ingest-telnyx command loop (`main.go:172-206`)~~ done (docs-health pass 2026-10-12) | ~~High~~ | ~~M~~ | ~~Quality~~ |
| ~~12~~ | ~~T12: poller timers (cqrs-lite timers) so the index goes autonomous~~ done (docs-health pass 2026-10-12) | ~~High~~ | ~~M~~ | ~~Feature~~ |
| ~~13~~ | ~~T12: freshness self-check + stale-index UI banner (M33-M34)~~ done (docs-health pass 2026-10-12) | ~~High~~ | ~~M~~ | ~~Feature~~ |
| ~~14~~ | ~~T8: per-call detail page (M23-M25)~~ done (docs-health pass 2026-10-12) | ~~Med~~ | ~~M~~ | ~~Feature~~ |
| 15 | T8: recording streaming proxy; signed-URL refetch server-side **→ ROADMAP theme 1 (export streaming)** | Med      | L            | Feature       |
| ~~16~~ | ~~Decide excerpt semantics (asc-by-created_at vs newest-first) — unblocks #2~~ done (docs-health pass 2026-10-12) | ~~High~~ | ~~S~~ | ~~Decision~~ |
| ~~17~~ | ~~Decide CI story: GitHub Actions lane vs buildflow-only, then wire~~ done (docs-health pass 2026-10-12) | ~~Med~~ | ~~M~~ | ~~Decision~~ |
| ~~18~~ | ~~Add cqrs-lint to flake devShell — kill the /tmp binary dependency~~ done (docs-health pass 2026-10-12) | ~~Med~~ | ~~S~~ | ~~Cleanup~~ |
| ~~19~~ | ~~T24: UI auth via operator htpasswd reuse~~ done (docs-health pass 2026-10-12) | ~~Med~~ | ~~M~~ | ~~Feature~~ |
| 20 | webphone read-only deep link contract (link-in, never data-dependency) **→ ROADMAP theme 3 (sibling side)** | Med      | M            | Feature       |
| 21 | PII policy: enforce `E164` + `ParticipantHash` on stored observations **→ ROADMAP theme 4 (stored-row enforcement)** | Med      | M            | Feature       |
| ~~22~~ | ~~T9: agent-transcript SourceKind ingest path~~ done (docs-health pass 2026-10-12) | ~~Med~~ | ~~L~~ | ~~Feature~~ |
| 23 | Inbound AI calls: DID interception path; revisit `-direction` default **→ ROADMAP theme 4** | Med      | L            | Feature       |
| 24 | Schedule the quarterly Telnyx live-API re-verification cluster (due ~2027-01) **→ TODO_LIST blocked row (needs a real API key)** | Med      | S            | Maintenance   |
| 25 | E003 owner decision: split `internal/calls` or keep flat domain (revisit when T9 lands) **→ documented HOLD (AGENTS.md, docs/cqrs-lint-policy.md)** | Med      | S            | Decision      |
| ~~26~~ | ~~Record F005 obligation: schema upcasters become mandatory work at v2~~ done (docs-health pass 2026-10-12) | ~~Low~~ | ~~S~~ | ~~Documentation~~ |
| 27 | ~~Decide templ+Tailwind asset pipeline details (codegen, CSP-safety, dark mode) before T6 coding starts~~ done — decided by the 01:32 frontend plan + the F4 pipeline (templ preBuild + tailwindcss_4) | Med      | M            | Decision      |
| ~~28~~ | ~~Verify ContentHash replay idempotency is covered across BOTH ingest paths (cdr + telnyx), not just one~~ done (docs-health pass 2026-10-12) | ~~Med~~ | ~~S~~ | ~~Quality~~ |
| ~~29~~ | ~~Verify `FlexTime` regression test covers the offset-less (`2006-01-02T15:04:05`) decode case~~ done (docs-health pass 2026-10-12) | ~~Med~~ | ~~S~~ | ~~Quality~~ |
| ~~30~~ | ~~Re-run `smoke` after the nix fix to keep the CLI path green end-to-end~~ done (docs-health pass 2026-10-12) | ~~Med~~ | ~~S~~ | ~~Quality~~ |
| ~~31~~ | ~~Check C003 fold totality still holds when T9 event types land (unknown-event → Corruption)~~ done (docs-health pass 2026-10-12) | ~~Med~~ | ~~S~~ | ~~Quality~~ |
| ~~32~~ | ~~Decide list-page → detail route shape (`/calls/{id}`)~~ done (docs-health pass 2026-10-12) | ~~Low~~ | ~~S~~ | ~~Decision~~ |
| ~~33~~ | ~~Decide list-page pagination strategy~~ done (docs-health pass 2026-10-12) | ~~Low~~ | ~~S~~ | ~~Decision~~ |
| ~~34~~ | ~~Decide serve-mode DSN/deploy layout (memory vs sqlite path for the server)~~ done (docs-health pass 2026-10-12) | ~~Med~~ | ~~M~~ | ~~Decision~~ |
| ~~35~~ | ~~Update AGENTS.md stack decisions the moment templ-components lands (dependency policy says "ONLY" three families)~~ done (docs-health pass 2026-10-12) | ~~Low~~ | ~~S~~ | ~~Documentation~~ |
| ~~36~~ | ~~Write T6 UI copy in domain language (Observation, CallView, SourceKind)~~ done (docs-health pass 2026-10-12) | ~~Low~~ | ~~S~~ | ~~Documentation~~ |
| ~~37~~ | ~~Watch Telnyx recordings-list filter docs for server-side `RecordingByCallControl` replacement~~ done (docs-health pass 2026-10-12) | ~~Low~~ | ~~S~~ | ~~Maintenance~~ |
| ~~38~~ | ~~Verify CDR parser coverage claim (13- and 18-column templates) against fixtures~~ done (docs-health pass 2026-10-12) | ~~Low~~ | ~~S~~ | ~~Quality~~ |
| ~~39~~ | ~~README: add CLI usage section (`version \~~ done (docs-health pass 2026-10-12) | ~~smoke \~~ | ~~ingest-cdr \~~ | ~~ingest-telnyx`) — it currently undersells the working engine~~ | ~~Med~~ | ~~S~~ | ~~Documentation~~ |
| ~~40~~ | ~~After this report: docs-health HARVEST — route (f) items into TODO_LIST.md / ROADMAP.md so they don't die in this timestamped file~~ done (docs-health pass 2026-10-12) | ~~High~~ | ~~M~~ | ~~Process~~ |

## g) Three questions I cannot figure out myself

1. ~~**Priority intent:** Was the templ-components question a signal to build T6 (serve + list page) **now**, or should the nix-package fix (#1) and excerpt bug (#2) go first? I can read the docs' priorities, but not your intent behind the question — and it decides the next session's shape.~~ done — answered: the UI was built on templ-components the same day (F3/F4)
2. ~~**Excerpt semantics (explicitly marked "define semantics" in TODO_LIST:30):** sort messages ascending by `created_at` (= call opening) or pin newest-first? This is a domain call about what an operator wants to see at a glance; it blocks the Critical fix #2.~~ done — decided: first user turn by created_at (call opening); A1 fix + regression test
3. ~~**CI story (explicitly marked open in TODO_LIST:49):** GitHub Actions lane like pbx-artmann, or buildflow-only? This gates item #17 and every future PR's safety net.~~ done — decided: GitHub Actions lane, green on main

---

_End of report. Per status-report skill step 4: WAITING FOR FURTHER INSTRUCTIONS._
