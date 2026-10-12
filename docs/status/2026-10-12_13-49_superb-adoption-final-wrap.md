<!-- docs/status/2026-10-12_13-49_superb-adoption-final-wrap.md -->
<!-- Point-in-time FINAL wrap of the 2026-10-12 12:31→13:49 SUPERB session
     (HTTP-stack adoption + verification + upstream filings). Supersedes the
     13-28 report's open-items list with current states; that report stays
     as-is (already annotated once). Do not edit later — ANNOTATE instead. -->

# Status (final wrap): SUPERB HTTP-stack adoption — incident, verification, filings

**Session:** 2026-10-12 ~12:31 → 13:49 · **Report:** 13:49 · **Feature commit:** `fd0e082` (pushed; CI red on itself, green since `6bb7d17`) · **Prior reports:** 13-28 (mid-session), appendix 13:5x

**One-line verdict:** All adoption work is done, verified, documented, and filed upstream — and the session's defining failure is that I verified the _working tree_ but shipped the _commit_, putting one public red X on `main` that a concurrent session's commit, not mine, healed.

---

## a) FULLY DONE

| #  | Work                                                                                                                                                                                                                                                                                                                               | Evidence                                    |
| -- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------- |
| A1 | The full adoption set: ETag/304 + Skip policy, Server-Timing (8 sites), gzip (outer of ETag), `/version`, `/metrics`, `WriteTimeout: NoTimeout` stream fix, templ render buffering, 3 deps promoted direct                                                                                                                         | `fd0e082` on `main`                         |
| A2 | Verification, ultimately complete: 14/14 pkgs green, `golangci-lint` 0 issues, gofmt clean, **race detector green** (web+cmd, 13s), **CI full matrix green** on main HEAD post-`6bb7d17` (runs 38243761913, 38244128681), `nix build` green twice on the dirty shared tree, live probe (304s, 4–6x compression, per-route timings) | CI + local logs                             |
| A3 | Upstream filings landed + verified OPEN + ledgered: **go-etag#2** (Flush observability), **go-appkit#26** (testkit 5s drain/test); voice-checked 0 FAIL/0 WARN, banners unchecked for owner review                                                                                                                                 | `docs/upstream-filings.md`, `docs/drafts/*` |
| A4 | Docs complete: plan (12-47, Pareto + mermaid), AGENTS.md (deps + serve/UI), FEATURES.md (+6 rows), CHANGELOG.md, TODO_LIST.md (+2), status reports 13-28 (+ appendix)                                                                                                                                                              | committed / daemon-picked                   |
| A5 | Incident root-caused end-to-end (CI red on `fd0e082` → harness shipped unstaged-deleted → clean-runner `/tmp` write; healed `6bb7d17`) and documented in the 13-28 appendix                                                                                                                                                        | this file §d                                |

## b) PARTIALLY DONE

| #  | Item                              | State                                                                                                                                                                                                                                                                     |
| -- | --------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| B1 | Clean full local `buildflow` gate | Attempted once load < 40; 2 steps + 9 tools failed — tool-lock contention with the concurrent session's active work (BuildFlow#45 class). `nix build` green ×2 proves no real hash/compile issue. Needs a QUIET tree; CI's green matrix covered my commit in the meantime |
| B2 | Browser-level htmx revalidation   | Still nobody looked: 304s proven only with hand-set `If-None-Match` (Go client). Fragments also still lack explicit `Cache-Control: no-cache` — deterministic revalidation remains unimplemented                                                                          |
| B3 | Markdown hygiene of my doc edits  | FEATURES/TODO table rows column-unaligned; `buildflow -s markdown-lint` (on-demand) never ran on them                                                                                                                                                                     |

## c) NOT STARTED (all deliberate, tracked)

- `/metrics` scrape option in the sibling NixOS module (TODO row)
- HTML research report (TODO row; substance lives in plan + AGENTS.md)
- go-etag `On304`/`OnETagGenerated` → `/api/ops` cache-hit ratio; `OnBufferOverflow` logging
- `Cache-Control: no-cache` on fragment/page responses
- README ops notes for `/version`, `/metrics`, ETag behavior; `docs/gotchas.md` templ-flush entry
- Everything else on the 13-28 §f list that isn't marked done above

## d) TOTALLY FUCKED UP

1. **I verified the wrong artifact and pushed it.** I ran the suite on the shared working tree — where the concurrent session had already deleted `zz_dump_audit_test.go` — then committed WITHOUT staging their deletion. The commit therefore shipped a harness that writes `/tmp/pbxlens-baseline` without creating it → public CI FAILURE (`38242512527`) on `main`. I did not catch it, did not watch CI after pushing (the red X sat ~17 min until the user's "Is that all?"), and did not even heal it — the daemon committing their deletion (`6bb7d17`) did. My commit message even claimed "Full suite 14/14 packages," true for the tree, false for the commit. Class: verification-of-the-worktree ≠ verification-of-the-commit.
2. **Near-miss, same class:** the gate run I started while the second session was visibly mid-flight (their files in `git status`) burned minutes producing uninterpretable failures. I knew the precondition (quiet tree) and ran anyway.

## e) WHAT WE SHOULD IMPROVE

1. **Ship-state verification:** before pushing, test what HEAD will contain — `git stash -u` the unstaged foreign changes (or test in a clean worktree of the commit). `git status` entries are not noise; they are the delta between verified and shipped.
2. **Push ≠ done:** watch the CI run to completion after every push (first-run failures are literally documented as this repo's pattern).
3. **Run the cheap thing, don't TODO it:** the race check I deferred cost 13 seconds.
4. **Concurrency pre-flight:** before suite/gate/commit in this repo, check for foreign activity (fresh mtimes, `git status` entries I didn't author, seconds-old daemon commits) and either wait or scope to a worktree. Twice this session, the shared tree made my signals lie to me.
5. **Assert the hot path in the first test draft**, not a convenient stand-in (static asset ≠ templ fragment — the flush bug survived a green e2e).
6. **Check the probe before doubting the code** (missing `Accept-Encoding` almost became a "bug").
7. **Concurrent session reality:** this repo demonstrably runs two agents at once; a written protocol (claim header in AGENTS.md, or branch-per-session) is now justified, not paranoia.

## f) NEXT — up to 50 (state-carryover from 13-28 §f + delta; ✅ done, 🔶 partial, ⬜ open)

| #  | Task                                                                       | State                                                               |
| -- | -------------------------------------------------------------------------- | ------------------------------------------------------------------- |
| 1  | Race detector on changed packages                                          | ✅ done 13:45                                                       |
| 2  | Watch CI on `fd0e082`                                                      | ✅ done — RED, root-caused, healed by `6bb7d17`; matrix green since |
| 3  | Annotate 13-28 report with incident                                        | ✅ done                                                             |
| 4  | File upstream: go-etag Flush observability                                 | ✅ go-etag#2 OPEN                                                   |
| 5  | File upstream: testkit drain cost                                          | ✅ go-appkit#26 OPEN                                                |
| 6  | Clean full buildflow gate on a quiet tree                                  | 🔶 B1 — blocked on concurrent session landing                       |
| 7  | Browser check: htmx fragment polls really 304                              | ⬜ B2                                                               |
| 8  | `Cache-Control: no-cache` on fragments/pages                               | ⬜ B2 — small, safe, unstarted                                      |
| 9  | `buildflow -s markdown-lint` (+ align my table rows)                       | ⬜ B3                                                               |
| 10 | go-etag `On304`/`OnETagGenerated` → `/api/ops` hit ratio                   | ⬜                                                                  |
| 11 | Log `OnBufferOverflow` (>1MiB untagged streaming)                          | ⬜                                                                  |
| 12 | HTML research report (or retire the deliverable)                           | ⬜ TODO row                                                         |
| 13 | Sibling NixOS `/metrics` scrape option                                     | ⬜ TODO row                                                         |
| 14 | Real Prometheus/Gatus scrape test of `/metrics`                            | ⬜                                                                  |
| 15 | Verify `Vary: Accept-Encoding` on 304s                                     | ⬜                                                                  |
| 16 | Confirm `Content-Length` set on buffered renders                           | ⬜                                                                  |
| 17 | README ops notes (/version, /metrics, ETag)                                | ⬜                                                                  |
| 18 | `docs/gotchas.md`: templ-flush-kills-ETag entry                            | ⬜                                                                  |
| 19 | Annotate 12-47 plan: EXECUTED + `fd0e082`                                  | ⬜                                                                  |
| 20 | AGENTS.md 220-line budget check after my expansions                        | ⬜                                                                  |
| 21 | Multi-agent protocol for this repo (claim header / branch-per-session)     | ⬜ — now session-proven need                                        |
| 22 | pinned-version-vs-HEAD research trap → crush-config lessons.md             | ⬜                                                                  |
| 23 | Crash-resilient auto-commit daemon (stale index.lock retry)                | ⬜ crush-config                                                     |
| 24 | e2e suite runtime: trim testkit drain cost — **now upstream go-appkit#26** | 🔶 filed                                                            |
| 25 | Skip-policy tests: HEAD/query-string/trailing-slash variants               | ⬜                                                                  |
| 26 | Concurrent-poll hash-cost sanity probe                                     | ⬜                                                                  |
| 27 | Export gzip CPU check at prod index size                                   | ⬜ deferred                                                         |
| 28 | Marathon-transcript >1MiB: accept untagged or raise buffer                 | ⬜ pairs w/ #11                                                     |
| 29 | Dependabot policy note for the 3 newly-direct deps                         | ⬜                                                                  |
| 30 | Revisit errorpages/security when templ-components publishes errorpage      | ⬜ blocked upstream                                                 |
| 31 | Telnyx etag-client revisit (2027-01 live window)                           | ⬜ calendar                                                         |
| 32 | Data-fingerprint fast-304 if index grows ~100x                             | ⬜ deferred                                                         |
| 33 | `/api/ops` stale-pass warning                                              | ⬜ pre-existing row                                                 |
| 34 | Filters into staleness fragment links                                      | ⬜ pre-existing row                                                 |
| 35 | SECURITY.md: `/metrics` auth posture                                       | ⬜                                                                  |
| 36 | Static assets: hashed filenames + long max-age (measure first)             | ⬜ YAGNI-lean                                                       |
| 37 | `entitytag` direct only if handler-set tags ever land                      | ⬜ hold                                                             |
| 38 | Consolidate timing.go/caching.go if a 4th middleware arrives               | ⬜ hygiene                                                          |
| 39 | htmx DOM-churn check on cached-body swaps                                  | ⬜                                                                  |
| 40 | Dashboard page 304 rarity (banner age) — accepted, revisit on complaint    | ⬜ decided                                                          |
| 41 | Next release cut (CHANGELOG Unreleased is rich)                            | ⬜                                                                  |
| 42 | cqrs-lint on new files (never locally run; CI ran it green)                | ✅ via CI                                                           |
| 43 | Coverage pass on the two new test files                                    | ⬜                                                                  |
| 44 | etag/gzip ratios as appkit_* metrics (single surface)                      | ⬜ design                                                           |
| 45 | `/tmp/pbxlens-probe` cleanup                                               | ⬜ cosmetic                                                         |
| 46 | recording_test: assert timing hops don't perturb timeout budgets           | ⬜                                                                  |
| 47 | gopls stale `modernize` diagnostic — restart LSP to clear                  | ⬜ cosmetic                                                         |
| 48 | Watch go-etag#2 / go-appkit#26 for upstream movement; fold in when landed  | ⬜ ledger duty                                                      |
| 49 | Dependabot sweep: confirm server_timing v1.0.2 rides routine bumps         | ⬜                                                                  |
| 50 | HARVEST this list + 13-28 §f into TODO_LIST.md (only 2 rows exist there)   | ⬜ docs-health                                                      |

## g) THREE QUESTIONS I CANNOT FIGURE OUT MYSELF

1. **Sequencing vs the concurrent session:** their templ-components pass is mid-flight in the same tree. Should I hold ALL further pbx-lens edits (even small, non-colliding ones like `Cache-Control: no-cache` on fragments) until they land, or is last-writer-wins on disjoint files acceptable here?
2. **Browser verification:** will you do the one manual Chrome/Firefox check of the fragment polls revalidating to 304, or should I ship `Cache-Control: no-cache` + trust the server-side proof without it?
3. **Multi-agent protocol:** do you want a written coordination convention for this repo (session-claim line in AGENTS.md, or branch-per-agent with explicit merges), or keep the current daemon-committed free-for-all? Twice today the shared tree made two independent sessions lie to each other.

---

_Snapshot only. Next actions live in TODO_LIST.md after HARVEST; bring this file current only via ANNOTATE._

_Now waiting for instructions._
