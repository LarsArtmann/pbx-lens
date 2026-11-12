# Session status — fleet audit refresh (22 libs) + the cqrs-htmx adoption reconciled

**Session:** crush audit-refresh, 2026-11-12 ~04:00–05:26 CET · **Lane:** docs/review/libs fleet audit only (no code bumps, no identity work — that was the parallel session's lane)
**Scope line — what this report VERIFIED:** build/tidy/full-test-suite/corpus/guard-test outputs executed live in this session (`go test ./...` green, `go mod tidy -diff` clean, `TestTailwindCorpus*`, `TestServerTimingHeaderOnPlainRoute|TestETagMiddlewareWriteErrorCounts|TestFatalCodeFamilyMapping` green); every module version claim re-derived via `go list -m -versions` (proxy) and checkout `git tag --sort=-v:refname`; the refreshed `index.html` rendered in headless chromium and visually inspected (screenshot re-shot, 485 KB); go.mod line citations audited and corrected; verify-blocks of the touched reports replayed (one broken command found and fixed).
**Scope line — what this report did NOT verify:** the canonical BuildFlow poll-gate was NOT run (the tree never stood still — parallel identity session + auto-commit daemon were landing commits throughout; docs/concurrency.md rule 2 voids receipts from a moving tree — the component gates above are the evidence, the formal gate is not); the parallel session's deep claims (filings #48–#63 closure, its CSP-nonce fix, "master 88 commits ahead") were reconciled only where they overlapped my files, not independently re-derived; docs/http-stack.md, README.md, and the sibling NixOS module were NOT audited against the new identity chain; no browser-level login-ceremony test was run.

---

## a) FULLY DONE

1. **State diff vs the 2026-11-04 audit** — go.mod history, all 20 local checkouts, ~35 `go list -m -versions` proxy queries. The full behind/current matrix exists in this session's transcript and is baked into the reports.
2. **go-etag truth correction** — "upstream v0.7.0" disproven: the repo cut a ROOT tag, but the server submodule needs `server/vX` tags; the proxy rejects `server@v0.7.0` (verified: `go list -m ...@v0.7.0` → `unknown revision`). v0.6.1 is the latest consumable release; go-etag.md's currency section now says so.
3. **go-filewatcher mystery solved** — report's "v2.4.3" vs proxy "v0.3.0" was a module-path artifact (`/v2` module); real latest is v2.5.0 (health check + circuit breaker, 2026-11-06).
4. **Upstream delta research** — go-appkit v0.10–v0.12 (RegisterClassifications, Server-Timing default outermost, go 1.27.3 floor, httputil.Server composition), go-cqrs-lite's three waves, cqrs-htmx v4.16–v4.18.1, templ-components v1.27.0, go-health v0.10.0, health-dashboard v0.13.x, credentials-store v0.4.x — all read from checkouts and folded into reports.
5. **Reports refreshed (mine):** templ-components (v1.27.0 + D1-fold state fixed), go-cqrs-lite (14 direct pins, 4-behind matrix, storage/v4 row), go-appkit (v0.10–v0.12 deltas, D4 flip-trigger RELEASED status), go-error-family (census 23→29 files, +10 identity Rejection sites), httputil (CSRF moved N/A→USED via the identity adoption; lockout replaces the deleted htpasswd buckets), httputil-server-timing (identity-gate rationale, appkit interplay note), go-etag (currency correction).
6. **Three new reports:** go-datastar.md, cmdguard.md, httputil-endpoints.md — REJECT each, with recorded reasons and falsifiable flip-triggers. (The parallel session later touched their mtimes; content verified still mine, `cmdguard`/`endpoints` zero-graph claims replayed green.)
7. **index.html fleet refresh** — 22 libraries, 8 used / 14 not-imported-directly, adopted cqrs-htmx row (78/100), 3 new rows, scorecards 7/8 and 6/8, refresh callout, rewritten verify block, footer.
8. **Screenshot** — re-shot at 1680×2400, downscaled copy visually inspected: layout intact, score cards show the derived numbers, all new rows render.
9. **AGENTS.md memory** — review-libs paragraph repaired to the true arithmetic (14 not-imported-directly, 4 indirect-only riders, REJECT ×11 / DEFER ×2 / PATTERN-ONLY ×1) after the parallel session's stale "11 not imported" and my sed left a broken fragment (repaired via python with assert-in).
10. **vendorHash repair** — the identity session left the gate RED ("wip identity for vendor-hash repair"); ran the sanctioned `nix run .#update-vendor-hash` on the clean tree, green-build verified, `nix develop -c echo ok` green.
11. **Verification battery** — full `go test ./...` green (zero failures), tidy clean, corpus freshness green at HEAD, guard tests green, `rg -rn` typo found in go-datastar.md's verify block and fixed.
12. **Concurrency protocol** — claims written before my bulk edit (21 claim files), zero foreign work reverted, every overlap with the parallel session reconciled by verification instead of re-overwrite.
13. **Receipt commit** — `f462d12` (my delta after the daemon's sweeps) + the daemon's own mixed commits carry the rest.

## b) PARTIALLY DONE

1. **The canonical gate.** Component gates green, BuildFlow poll-gate not run (moving tree — see scope line). The refresh's claims hold at component level; the formal single-HEAD receipt is missing.
2. **cqrs-htmx.md authorship.** I wrote a full rewrite; the parallel session overwrote it with its own (better score breakdown). I ended up editor of their version (fixed their `go.mod:6-9` → `7-9`). The report is superb; my authorship didn't survive — process cost, not quality cost.
3. **Verify-block replay.** ~60% replayed (the blocks behind my edits + the index block). All 22 reports' blocks were not exhaustively looped.
4. **Pin currency itself.** Reports-only mandate: cqrs-lite (4 modules) and appkit (3 minors) stay behind, documented as scheduled holds with fold-in checklists. The deletable Server-Timing wiring in serve.go is still there.
5. **Overlap QA.** I reconciled the files we both touched, but did not re-derive the parallel session's independent deep claims (filings closure, CSP-nonce fix in forms, "master 88 ahead").
6. **Screenshot custody.** Both sessions claim a re-shot; mine (05:21) is the last write I verified. Whoever's PNG is current, it IS render-verified by one of us.

## c) NOT STARTED (noticed, deliberately out of my lane or deferred to wait-state)

1. **HARVEST of section (f) into TODO_LIST.md** — waiting per "then wait for instructions"; TODO_LIST is also the other session's active file right now.
2. **docs/http-stack.md vs the identity chain** — the doctrine doc almost certainly still carries htpasswd-era wording (auth gate source, the `When`-decision rationale, the two-hop budget's auth row). I updated the lib reports, not the doctrine doc.
3. **AGENTS.md-wide htpasswd sweep** — line 164 has the new Auth paragraph, but older sections ("serve htpasswd timing pad", "auth-failure throttling PER CLIENT ADDRESS", "padCost") still describe the DELETED gate — split-brain risk for the next session.
4. **README.md / FEATURES.md user-facing auth claims** — not checked at all.
5. **Sibling NixOS module** — `-auth-db/-auth-rpid/-auth-origin` deploy knobs not wired (also the other session's open question).
6. **Upstream-filings ledger rows for the two templ candidates** — the parallel session's item #49.
7. **CHANGELOG.md row for the 22-lib refresh** — not written by me.

## d) TOTALLY FUCKED UP (process damage only — the final tree is green and consistent)

1. **A full report rewrite wasted.** My cqrs-htmx.md rewrite was overwritten by the parallel session doing the same task. Root cause: I bulk-claimed files only AFTER my first write collision, and claims are advisory anyway — the other session never read them. ~1 report of duplicated effort; the collision was caught by the edit tool's freshness check, not by my process.
2. **Broken sentence shipped into AGENTS.md mid-flight.** My `sed` on the shared memory file left "(retry rides\n zero pbx-lens imports" — a mangled sentence in the most-read file in the repo, live for ~2 minutes until my python repair. Sed-without-assert on a shared file is the exact anti-pattern the concurrency doc exists to prevent.
3. **Blind-write attempt.** After the "file modified since read" refusal on cqrs-htmx.md, I first investigated correctly — but I had attempted the write in the first place without re-reading after the foreign 04:58 change landed.
4. **Two broken commands of my own shipped in reports:** the `rg -rn` flag typo in go-datastar.md (caught in replay, fixed) and a copy-paste `Pin v0.6.1` old_string that failed an edit round-trip on server_timing. Small, but both were self-inflicted.
5. **Wrong-needle panic.** A grep with a needle my own text didn't contain ("patches are values") made me briefly conclude my datastar report had been overwritten; one wasted verification round.
6. **Early greps used a broken `\t` escape** producing garbage warnings — re-run needed where one clean `--name-only`-style diff would have done.

## e) WHAT WE SHOULD IMPROVE

1. **Claim-before-first-write, in a shared place.** Per-file `.claims/` nobody reads is advisory theater. A single `.claims/ACTIVE.md` roster (session → file list) that both sessions append to would have prevented the cqrs-htmx double-work entirely.
2. **Proxy-first version checks.** `go list -m -versions` should be the FIRST query for every lib (checkout tags mislead twice tonight: filewatcher's /v2 path, go-etag's non-consumable root tag). Checkout tags are for verification, not discovery.
3. **Never sed shared memory files.** AGENTS.md edits go through read → python-replace with `assert old in t` → re-read. Tonight's broken fragment came from exactly that shortcut.
4. **Script the verify-block replay.** One loop extracting ```console blocks from docs/review/libs/*.md and running them would make the done-bar mechanical instead of ad-hoc (~60% coverage tonight).
5. **Moving-tree receipts.** When a lane must produce "gate green" evidence on a tree with a concurrent writer, either agree a quiet window or explicitly downgrade the claim to component gates — say which in the report header (done tonight, but it should be the reflex, not the correction).
6. **The wip-red-tree norm.** The identity session's "wip vendor-hash repair" left the gate red on a committed HEAD. A norm like "leave HEAD green or mark the commit `[red]` in its message" would let the next session trust `git log` again.

## f) Up to 50 things to get done next

_(Brainstorm inventory — routing rigor per docs-health HARVEST applies; most of the tail belongs in ROADMAP, the head in TODO_LIST.)_

1. Run the canonical BuildFlow poll-gate on a still tree; single-HEAD receipt for the 22-lib refresh.
2. Bump go-cqrs-lite's 4 trailing modules (system v4.18.0, metaengine v4.23.0, sqliteengine v4.8.0, systemscenario v4.3.0); re-run the fence tests + corpus.
3. Bump go-appkit v0.9.0 → v0.12.0; delete the now-duplicative `web.ServerTimingMiddleware()` wiring in serve.go; adopt `WaitForRunning`/`FreeAddr` in serve_test; decide on `RegisterClassifications()`.
4. Audit docs/http-stack.md against the identity chain (auth-gate source, `When` rationale, two-hop budget auth row, CSRF posture).
5. Sweep AGENTS.md for remaining htpasswd-era claims (timing-pad/padCost/throttle paragraphs vs the deleted gate).
6. Sweep README.md + FEATURES.md for user-facing auth claims (htpasswd → passkey identity).
7. HARVEST this list into TODO_LIST.md (docs-health HARVEST mode) once the wait-state lifts.
8. Wire `-auth-db/-auth-rpid/-auth-origin` into the sibling repo's `modules/telephony/lens.nix`.
9. `/ready` projection-worker readiness — the one standing cqrs-htmx health-bridge idea (audit's oldest open borrow).
10. Backup coverage for the identity SQLite DB (internal/backup owns the index DB — the identity DB is NEW durable state; verify it's in the backup path).
11. Retention/purge timers: prove they cannot touch the identity DB.
12. Identity DB health in `/health`/`/ready` (its own liveness signal).
13. Identity DB pragma tuning (WAL, busy_timeout) — same class as the index DB pragmas finding.
14. Session-cookie flag audit (Secure/HttpOnly/SameSite under plain-HTTP LAN posture).
15. /api/ops: add identity counters (lockouts, active sessions, registration-closed state).
16. Per-address 4xx count in /api/ops — the named "cheapest first instrument" of the httputil rate-limit Hold.
17. E2E probe suites (scripts/verify-all.sh): add an authenticated lane exercising the login ceremony.
18. Screenshot-probe baselines: confirm the login wall doesn't break the four probe suites.
19. Upstream-filings ledger rows for the two templ filing candidates (parallel session #49).
20. templ-components v1.28.0 watch: fold responsive-Sparkline + CopyButtonScript when released; retire those two workarounds.
21. Re-check cqrs-htmx pins on the next fleet sweep (≈30 commits/day upstream; v5 will remove re-export façades).
22. go-datastar flip-trigger re-check when templ-components ships its Datastar builders.
23. Reconcile the go-cqrs-lite report's versions.json story permanently (proxy-first rule now in the verify block — keep it canonical).
24. go-error-family: `WithContext` at source (standing "largest remaining win").
25. go-retry DEFER re-check if the Telnyx client grows per-request 429/5xx handling.
26. `-race` over the identity tests (sessions + sqlite concurrency — parallel session #45).
27. Benchmark the auth gate's per-request overhead; add the auth row to docs/http-stack.md numbers (#46).
28. Re-run BenchmarkConcurrentPoll after the middleware chain grew the session link (#47).
29. Review `Identity.Register` rate-limit numbers against real ceremony traffic (#48).
30. Sweep the repo for exact-match assertions the templ codegen change may break (#44).
31. TOTP fallback factor decision — contingent on the TLS answer (Q2 below).
32. robots.txt: verify it exists and is honored through publicAuthPath (M15/F46 reference).
33. CSRF posture note in docs/http-stack.md (CSRFConfig{} defaults now guard the ceremonies).
34. Loginpage assets under enforcing CSP: confirm nonce threading covers the login page, not just app forms.
35. Add the identity aggregate to the `catalog`/`-architecture` render if it counts as a second aggregate (E003 flat-hold trigger check!).
36. CHANGELOG.md row for the audit refresh (22 libs, cqrs-htmx verdict upgrade).
37. docs/research/README.md index: add/refresh the row pointing at the 2026-11-11/12 refresh state.
38. Record the answered "reports vs bumps" decision (Q3 below) in docs/decisions when made.
39. Record the vendorHash-wip incident as a docs/concurrency.md lesson (leave-HEAD-green norm).
40. go-etag release-engineering filing: the root `v0.7.0` tag is not consumable for its submodules — verify-before-filing, then propose `server/vX` tagging hygiene upstream.
41. Quarterly proxy-lag re-check habit: `go list -m -versions` claims can trail fresh pushes (the etag lesson generalizes) — put it in the refresh rules.
42. Sibling-repo deep-link: confirm the webphone read-only link still works with the login wall (no data dependency, but UX).
43. Ops page: session/lockout cards behind the identity gate (feed from /api/ops additions).
44. Consider surfacing `identity-model` events in the eventcatalog hub tree if usermgmt ships one.
45. Retention `-max-age` gate vs identity rows: `Policy.Admits` review for the new DB.
46. Docs: a short runbook for "operator locked out" (lockout recovery path).
47. Docs: NixOS module example stanza for the auth flags (after #8).
48. TODO_LIST row for the cqrs-htmx readiness idea so it stops living only in the audit report.
49. Fleet-review hygiene: schedule the next full audit trigger (pin bump OR flip-trigger) as an owner-visible cadence note.
50. Decide the dual-session protocol (Q1 below) and write it into docs/concurrency.md if double-dispatch will recur.

## g) Questions only the owner can answer

1. **Was the fleet audit deliberately split into two parallel sessions** (identity+audit vs audit-refresh), or was it accidental double-dispatch of the same prompt? This decides whether the 22-report state is final, whether a third sweep should be expected, and whether the `.claims/` roster improvement (e) is worth building.
2. **TLS posture in production:** does something terminate HTTPS in front of pbx-lens on the LAN (passkeys need it), or is the lens plain-HTTP? This gates the NixOS auth defaults, the cookie `Secure` question, and whether a TOTP fallback factor is ever needed.
3. **Reports vs bumps:** do you want the four stale go-cqrs-lite modules + go-appkit v0.12.0 bumped NOW as ordered work (with the Server-Timing wiring deletion and fence-test rerun), or do they wait for the next coordinated deps train? The reports document both postures; only you can pick.

— End of report. Waiting for instructions.
