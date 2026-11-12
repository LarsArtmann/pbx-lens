# Session status — usermgmt adoption + fleet-audit refresh (2026-11-12 04:43)

**Session scope (operator order, 2026-11-11):** (1) fleet utilization audit
refresh across the first-party libs → `docs/review/libs/*.md` + `index.html`,
superb and at latest versions; (2) mid-session addition: **"I want native
usermgm via cqrs-htmx"** — execute the cqrs-htmx flip-trigger that the
2026-11-04 audit had parked as ADOPT-PARTIAL.

**Headline: the identity integration is LANDED and green end-to-end (build,
full test suite, lint 0, cqrs-lint 0, dedup gate 0, nix package build green
after vendor-hash repair, devshell repaired and green). The audit-refresh
half of the session is NOT started beyond research — the reports still say
2026-11-04.**

---

## a) FULLY DONE

1. **Research phase.** library-deep-dive + cqrs-htmx skills loaded; upstream
   research subagent returned a citation-verified API report (ServiceConfig
   fields, SQL store constructors, session middleware, loginpage mount,
   WebAuthn config, bootstrap semantics, version floors, close ordering) —
   every file cited was tag-verified.
2. **Coordinated bump (adoption-map step 8).** templ v0.3.1070→v0.3.1100,
   templ-components v1.26.0→v1.27.0 (root/htmx/icons/utils), go 1.27.2→
   1.27.3. MVS riders: metaengine v4.22.0, catalog v4.14.0, id v4.7.5,
   kv v4.4.0, stack v4.5.1, go-sse v0.7.0, go-idempotency v0.3.2.
3. **Bump checklist executed:** corpus regenerated (loginpage module ADDED
   to `scripts/gen-tc-classes.js` with a second pin stamp), goldens
   refreshed with `-update` and the diffs REVIEWED, art-dupl baseline
   checked (0 groups), nonce threading fixed, styles.css rebuilt via
   `nix run .#update-styles`.
4. **Two real bump regressions found and fixed:**
   - templ ≥v0.3.1100 codegen emits the icon→text separator space inside
     anchors — the pager test helper's `>Text</a>` match broke (fixed the
     helper, documented the behavior).
   - The new tc forms validity-bridge singleton script (`tcValidity…`)
     rendered WITHOUT the nonce on `/`, `/search`, `/dashboard` — a real
     CSP regression. Fixed by threading `Nonce` into all three forms.Form
     sites (calls filterForm, dashboard rangeForm — signature changed,
     search form).
5. **Identity integration (the centerpiece).** `internal/web/userauth.go`:
   `OpenIdentity` → own SQLite identity DB (event store + SQL session
   store + SQL read models + checkpoint store with hand-applied schema +
   `OptimizeSQLiteDB` pragmas + `MaxOpenConns(1)`), WebAuthn provider,
   lockout, `MaxUsers: 1`; middleware chain session → CSRF → gate
   (public: `/login`, `/auth/*`, `/app.css`, `/robots.txt`; browsers 303
   to /login, API clients 401); loginpage panel at GET /login; identity
   projections ANDed into `/health/ready`. PATH B manual wiring — NO setup
   bundle (it would collide with our `/dashboard` and drag adminui/
   dashboardui in).
6. **serve wiring.** Flags `-auth-db` / `-auth-rpid` / `-auth-origin`
   (empty `-auth-db` = unauthenticated localhost dev — probes unaffected);
   `deriveAuthDefaults` maps IP/localhost listen addresses to RPID
   `localhost` (go-webauthn REFUSES an IP RPID at construction — discovered
   by test, not doc); identity opened/mounted/closed with correct defers.
   htpasswd deleted from the tree (auth.go trashed, flag gone).
7. **Tests.** auth_test.go fully rewritten: gate contract (303/401/HX),
   public paths, first-registration-works-then-door-closes (no session
   issued on the refusal), **identity DB survives reopen** (SQL sessions),
   readiness after drain, and the full flow test re-authored on the new
   gate. The CSRF dance required replicating a real browser: Host + Origin
   + Referer + `X-Csrf-Token` + cookie pair (nosurf same-origin
   attestation) — pinned in helpers for future tests.
8. **Sibling deploy contract updated in lockstep.**
   `nix-international-telephony/modules/telephony/lens.nix`:
   `htpasswdFile` option → `auth.enable` + `auth.publicOrigin` (eval-time
   assertion), `-auth-db ${stateDir}/identity.db -auth-origin …` on
   ExecStart, and — because identity.db is NOT re-derivable — the nightly
   backup unit now stages a verified identity.db snapshot beside the
   index's.
9. **Doctrine docs.** AGENTS.md: cqrs-htmx adoption block, UI-trio bump
   block (incl. the two codegen behavior changes), serve/UI auth rewrite,
   flake repair records, fleet-audit line updated; docs/http-stack.md auth
   section rewritten (chain position, gate semantics, session posture,
   CSRF notes, dev posture).
10. **Nix chain repaired (two PRE-EXISTING breakages found on the way).**
    - The package preBuild hook ran the bare `templ` binary = pkgs.templ,
      which has drifted ahead of the go.mod pin — its codegen emitted
      `templ.ResolveAttributeValue` calls the pinned v0.3.1100 runtime does
      not have → every sandbox build died at compile. Fixed: the hook now
      runs `GOEXPERIMENT=jsonv2 go tool templ generate` (module-pinned),
      the doctrine the flake's own treefmt comment already carried.
    - cqrs-lint's `fetchFromGitHub` source hash was an ALL-ZEROS
      placeholder and its vendorHash pointed at the wrong module set — the
      devshell has been broken since before this session. Re-landed both
      (source `sha256-BH3IGPq…`, modules `sha256-jTwZn/T…`); devshell now
      builds (go 1.27.3, golangci-lint 2.15.0).
    - `vendorHash.nix` repaired via the sanctioned app on a clean tree:
      `sha256-lu4M2ZhNfmf9YVde4MtmLpeYkSy6MgRP09En3qeUu/0=`, **green
      package build verified** by the app itself.
11. **Gates.** `go test ./...` green; `go vet` clean; golangci-lint 0
    issues on `./internal/web/...` + `./cmd/pbx-lens/...` (cyclop
    refactors: openServeIndex/openServeIdentity/wireServeSources
    extractions; unparam, mnd, canonicalheader, errorlint, wrapcheck,
    gochecknoglobals, nilnil all resolved); cqrs-lint exit 0; dedup gate
    0 groups.

## b) PARTIALLY DONE

1. **The fleet-audit refresh (half the original order) — reports untouched.**
   Version-currency sweep DONE (22 repos, latest tags recorded); the
   research is in hand; the WRITING is not started. In detail:
   - `docs/review/libs/cqrs-htmx.md` still says **NOT USED /
     ADOPT-PARTIAL** — factually wrong as of this session; needs the full
     rewrite to ADOPTED with a re-derived adoption score.
   - The other 21 reports still carry the 2026-11-04 snapshot (4 pins are
     stale in the reports: go-etag v0.7.0, go-sse v0.7.0, appkit v0.12.0,
     templ-components v1.27.0 — all now current or MVS-landed).
   - `index.html` not rebuilt; `index-screenshot.png` not re-rendered.
2. **templ-components adoption map** (`docs/templ-components-adoption.md`):
   checklist steps 1/5/6/8 executed; steps 2 (re-audit ABSENT/rejected
   rows against the new module cache), 3 (hand-rolled thin pieces
   re-check), 4 (open filings #48–#51 re-read), 7 (filings-ledger state
   check) NOT run — plus new rows owed: the icon-text space, the duplicate
   `</script>` (upstream-filing candidate), the validity-bridge nonce
   threading.
3. **Browser probe lane** (`scripts/verify-all.sh`): not re-run after the
   integration. Go-level coverage is green, but the four probe suites are
   the contract for the UI.
4. **Login-page styling verification**: corpus carries loginpage classes
   and styles.css rebuilt, but my class-presence check was inconclusive
   (noisy extraction + a broken shell-escaping loop on my side). A rendered
   login page (probe or screenshot) is the honest check.
5. **Sibling repo state**: lens.nix edited; its commit/review left to the
   host repo's own flow (I did not run its gate or eval-test the module).

## c) NOT STARTED

1. cqrs-htmx.md report rewrite (ADOPTED + score).
2. Refresh of the remaining 21 reports + rebuild of index.html.
3. `docs/research/README.md` index row for the refreshed audit (the
   deep-dive done-bar's Index item).
4. `docs/DOMAIN_LANGUAGE.md`: identity-domain terms (operator account,
   passkey enrollment, session) — the serve section references rules that
   should live there.
5. TODO_LIST entry: **logout affordance** — there is no Sign-out button in
   the UI; logout exists only as `POST /auth/logout` (needs the CSRF token,
   so it is scriptable but not clickable). Deliberately deferred to keep
   the pass shippable; must not be forgotten.
6. Browser-level WebAuthn ceremony probe (CDP virtual authenticator) —
   enrollment/login are covered upstream + by the live-registry contract,
   but nothing in THIS repo drives a real ceremony yet.
7. CHANGELOG.md entry for the auth migration (breaking: `-htpasswd` flag
   removed).
8. `nix flake check` after the flake edits.
9. A written threat-model note for the registration-open-while-empty
   posture (first-visitor-to-/login on a trusted LAN becomes the operator;
   `MaxUsers: 1` closes the door; the boot log says so).
10. SECURITY.md / README quick-start update for the new flags (deploy
    operators currently follow the htpasswd recipe).

## d) TOTALLY FUCKED UP (own goals this session)

1. **RPID default that failed at startup**: I derived `-auth-rpid` from the
   listen host (127.0.0.1) without checking go-webauthn's validation —
   it refuses IP RPID at construction. Five identity tests failed en
   masse. Reworked via `deriveAuthDefaults`; the lesson (read the
   dependency's validation rules BEFORE writing defaults) is now recorded.
2. **Close() wrapped a nil error**: my first `%w` fix would have made a
   successful shutdown return an error. Caught on re-read; final shape is
   `errors.Join(service.Close(), db.Close())`.
3. **Self-inflicted test churn**: a careless edit clipped
   TestOneFlow's redirect assertion; I also briefly committed a
   placeholder `loginChallenge` that just called `t.Fatal`. Both fixed;
   both were avoidable with slower, whole-function edits.
4. **Invented data in a verification loop**: I checked "missing" CSS
   classes I had GUESSED (`place-items-center`, `text-pretty` — loginpage
   uses neither), then burned cycles on a shell-escaping bug in the same
   loop. The honest check (grep the loginpage sources) showed every real
   class present. Verification by fabrication is worse than no check.
5. **The `rg -r/--replace` hook trap, three strikes** — blocked three
   times this session for the exact same misread. Fully my repetitions.
6. **Discovered pre-existing breakage late**: the zeroed cqrs-lint source
   hash was committed state — the devshell has been unbuildable since
   2026-11-0x and nobody noticed because `nix flake check` never builds
   devShells (the exact 2026-10-25 trap AGENTS.md already documents). The
   repair is landed, but the detection should have been a standing gate.

## e) WHAT WE SHOULD IMPROVE

1. Make `nix build .#devShells.x86_64-linux.default` (or an equivalent
   eval) part of a scheduled/CI gate so a dead devshell cannot ship again —
   `nix flake check` is proven blind to it.
2. Pin the sandbox codegen to the module toolchain EVERYWHERE (done for
   preBuild; the devshell still carries nixpkgs templ v0.3.1020 as the fmt
   oracle — fine for fmt, but it is a standing drift the flake comment
   already warns about).
3. A CDP virtual-authenticator probe for the full passkey ceremony in this
   repo's probe lane (upstream tests + my headless dance are not the same
   as Chrome driving `navigator.credentials`).
4. File upstream: duplicate `</script>` after singleton scripts (templ
   v0.3.1100 codegen), and the icon→text separator space (breaking for
   exact-match consumers) — both are templ-components/templ filing
   candidates with golden evidence in hand.
5. Report discipline: the audit half should have been written BEFORE the
   integration (the reports' version-currency half was independent of it);
   sequencing them serially left the session half-shipped.

## f) NEXT 50 (ordered by leverage)

**Finish the ordered work (1–10)**
1. Rewrite `docs/review/libs/cqrs-htmx.md` → ADOPTED (usermgmt/loginpage/
   webauthn wiring, scores, gotchas, verify blocks).
2. Refresh the remaining 21 lib reports (versions, deltas, watchlists).
3. Rebuild `docs/review/libs/index.html` (+ re-screenshot, in-repo).
4. Add the research/README index row for the refreshed audit.
5. Run `scripts/verify-all.sh` (four probe suites) on the new tree.
6. Render-verify the login page (screenshot into docs assets).
7. Update `docs/templ-components-adoption.md` (steps 2/3/4/7 + new rows).
8. `docs/DOMAIN_LANGUAGE.md` identity terms; CHANGELOG entry.
9. TODO_LIST: logout button; SECURITY.md/README flag migration notes.
10. `nix flake check` + final poll-gate receipt on the finished tree.

**Auth hardening/UX (11–20)**
11. Sign-out button in the shell nav (needs the nonce/token bridge).
12. Show session/user state ("signed in as …") somewhere honest.
13. CDP virtual-authenticator login probe.
14. Threat-model note for the open-while-empty registration posture.
15. Decide + document: pre-provisioned operator account vs first-visitor.
16. `-auth-cookie-name`? (not needed — skip; recorded as considered).
17. Consider TOTP submodule as the plain-HTTP fallback factor.
18. Identity metrics on /api/ops (sessions active, lockouts).
19. Session TTL decision (currently 24h default) for the deploy module.
20. Document the curl/script export recipe under session auth.

**Fleet-audit follow-ups (21–30)**
21. go-appkit v0.9.0→v0.12.0 bump (the v0.10 Server-Timing watchlist item
    triggers: delete the serve.go wiring on bump).
22. go-etag v0.6.1→v0.7.0 bump + refresh that report.
23. templ-components #48/#49/#51 filings state check (step 7 debt).
24. Re-audit the adoption map's ABSENT rows against the v1.27 cache.
25. go-retry flip-trigger review (DEFER; now v0.9.0 in-graph via MVS).
26. go-atomic-write DEFER re-check after the identity DB lands (atomic
    enrollment writes are event-sourced — likely stays REJECT).
27. go-idempotency: now DIRECT in the graph via usermgmt — refresh its
    report from indirect to consumed-by-dependency.
28. go-sse v0.7.0 direct rider — same treatment.
29. cmdguard v5.1.0: the CLI layer is still raw `flag` — a deliberate
    REJECT needs re-recording in its report with this session's context.
30. Refresh docs/review/libs screenshots for changed scores.

**Nix/deploy (31–40)**
31. Land the sibling nix-international-telephony module change through its
    own gate + VM test suite if one exists.
32. Schedule the devshell-build gate (item e.1).
33. Upstream the pkgs.templ drift problem to go-nix-helpers (preBuild
    should default to `go tool templ` when templGenerationPolicy=preBuild).
34. Verify `nix build` on a SECOND system (darwin) — platforms=unix.
35. Backup-restore drill for identity.db (VACUUM INTO → re-attach).
36. Document the credential-free deploy (no more LoadCredential for auth).
37. Confirm the monitoring scrape path stays unauthenticated-deliberate.
38. Add `-auth-origin` multi-origin docs for proxy + localhost access.
39. Guard: serve refuses `-auth-db` AND `-htpasswd` together — flag is
    gone, but a defensive error message for the old flag would be kind.
40. Cut a release (cut-release.sh parity) once probes are green.

**Session-hygiene/process (41–50)**
41. Write the LESSON: "derive defaults from validated constraints" into
    the project's gotchas (go-webauthn RPID incident).
42. Add the CSRF-dance helper documentation to docs/gotchas.md (Host +
    Origin + Referer + token pair).
43. Golden-diff review checklist entry for the icon-text space class of
    change (exact-match tests vs whitespace-emitting codegen).
44. Sweep the repo for other `>Text</a>`-style exact-match assertions that
    the new codegen may silently break later.
45. Consider `-race` run over the identity tests (sessions + sqlite
    concurrency).
46. Benchmark the gate's per-request overhead (session lookup + nosurf) —
    the http-stack doctrine carries numbers; add the auth row.
47. Re-run the poll-304 benchmark after the middleware grew a chain link.
48. Review `Identity.Register`'s rate-limit numbers against real ceremony
    traffic (2 req/login → 30/min is generous).
49. Write the upstream-filings ledger rows for the two templ candidates.
50. THEN WAIT-state review: nothing merged downstream (webphone deep-link,
    sibling deploy) until probes + release receipt are green.

## g) QUESTIONS ONLY YOU CAN ANSWER

1. **TLS posture:** does the production deploy terminate HTTPS in front of
   pbx-lens (so passkeys work from the LAN), or is the lens plain-HTTP on
   the LAN? This decides whether `auth.enable` should default true in the
   NixOS module and whether we need the TOTP fallback factor at all.
2. **Bootstrap posture:** is "first visitor to /login becomes the operator"
   acceptable on your LAN, or do you want a pre-provisioned operator
   (CLI-seeded account + registration permanently closed before first
   serve)?
3. **Reports vs bumps:** for the audit refresh, do you want me to ALSO
   bump the four stale pins (appkit → v0.12.0 with its Server-Timing
   watchlist deletion, go-etag → v0.7.0) in this effort, or strictly
   reports-now and bumps as separate ordered work?

— End of report. Waiting for instructions.
