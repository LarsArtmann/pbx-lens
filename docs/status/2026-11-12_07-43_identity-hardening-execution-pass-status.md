# Status: identity-hardening execution pass (plan P01–P10) — 2026-11-12 07:43

**Session order:** execute the whole
[05-44 SUPERB plan](../planning/2026-11-12_05-44_SUPERB-identity-hardening-fleet-currency.md)
(gate-free lanes only): the identity decision pack, the sibling module's
own gate, the login surface, the deploy docs, the identity e2e suite, the
process debts, the receipts hygiene, and identity observability. A
concurrent session executed P06 (the appkit v0.12 bump) mid-pass —
reconciled, not duplicated (§d4).

## a) Executed (package by package)

- **P01 — decision pack (done).**
  docs/decisions/2026-11-12_identity-posture-answer-sheet.md: the three
  questions (TLS/bootstrap/pins) with code-truth "current" sections,
  consequence matrices, and recommendations (TLS-first REQUIRED, keep
  first-visitor with a named flip-trigger, HOLD pins ~1 week
  post-deploy). Linked from TODO_LIST row (answer inline there) and
  ROADMAP's new "Operator identity" raw-ideas block. P01.3 stays
  owner-gated.
- **P02 — sibling verification (done; receipts in ITS repo).**
  `nix flake check` EXIT 0 (twice — the first run failed ONLY on
  treefmt bytes, fixed by `nix fmt`). Added two lasting eval guards to
  tests/eval.nix (`lensAuthCheck`, `lensAuthNegativeCheck`): auth.enable
  wires `-auth-db`/`-auth-origin` + registers identity.db for the stack
  backups; auth.enable without publicOrigin is REJECTED at eval time.
  **The eval check caught a real module gap:** default.nix registered
  index.db but never identity.db in `services.telephony.state.sqliteDatabases`
  — fixed (+ comment on the purge unit pinning identity.db's exclusion
  from retention). Backup-rotation opinion folded into the backup
  unit/timer descriptions (they now name identity.db and "never
  rotates"). Runbook: docs/runbooks/identity-first-deploy.md.
  **P02.7 receipt commit pending the re-run gate (job 2E6).**
- **P03 — login surface (done).** /login rendered via the new
  screenshot-probe `login` phase (own `-auth-db` serve; 4 PNGs, all 200,
  tc teal visible — eyeballed). Logout implemented: navRight templ
  (hidden CSRF field + nonced fetch → redirect /login; renders only when
  `AuthEnabled`), goldens refreshed (nav wrapper div, +37 chars ×2) and
  green without -update; `TestIdentityLogoutEndsTheSessionEverywhere`
  proves server-side invalidation + honest nav in both states.
- **P04 — deploy docs (done).** SECURITY.md: "Identity and access" +
  "Threat model notes" sections; README auth recipe replaced the
  htpasswd commands; DOMAIN_LANGUAGE gained the session-lifetime
  paragraph (fixed 24h, no sliding, upstream default chosen-for-now);
  AGENTS.md auth/serve paragraphs synced with the corrected truth.
- **P05 — identity e2e (done).** New `scripts/verify-identity-e2e.js`
  (wired into verify-all after verify-ops-verdict; scripts/README.md
  row): **15 contracts green** — gate 303/401/public paths, no-store on
  /login, bootstrap 201 + non-Secure cookie + /auth/me, door closed,
  enrollment + login WebAuthn ceremonies through a CDP virtual
  authenticator (exact loginpage sequence), logout roundtrip, 5-failed-
  ceremony lockout + message. csp-sweep gained an auth-boot /login
  phase: zero violations across 7 surfaces.
- **P10 — process repairs (done).** docs/concurrency.md rule 7
  (re-stat before create/overwrite); verify-all `verify-all: PASS
  <gate>` verdict lines; docs-gates-in-writing-loop convention
  (AGENTS.md commands + verify-all header note); U2
  `TestSkipBodyCachingCoversTheRouteTable` (parses server.go mux
  literals; fails on a streaming route without a skip rule AND on a
  vacuous parse); U3 the ordering-contract comment on
  DrainProjections→RebuildOpenIndex in PollOnce; U4 /health/ready parity
  in verify-staleness-live; U1 the ANNOTATE receipts block inline in the
  2026-11-04 fleet plan (T01–T19 + T24, each cited to its artifact);
  TODO_LIST rows updated (five to DONE).
- **P09 — receipts & residue (done).** P09.1 pin spot receipts:
  cqrs-htmx usermgmt/v4.20.0 + templ-components/v1.27.0 are ANNOTATED
  tags (objecttype=tag; checkouts ce01d9d3 / 97c49407), templ v0.3.1100 +
  storage v4.11.1 proxy-verified via `go list -m`. P09.2 htpasswd
  residue: one live-doc fix (ROADMAP shipped-list marks the htpasswd
  wave SUPERSEDED); CHANGELOG/DOMAIN_LANGUAGE mentions are historical
  and stay. P09.3 refresh stamps on go-datastar/cmdguard/
  httputil-endpoints. P09.4 filing #60: CLOSED upstream 2026-11-07
  (post-tag; ledger row added; Sparkline render joins the P07
  re-verification list). P09.5 verdict: the staged-then-committed
  go-datastar.md delta was EXAMINED post-hoc (commit e09564b) — exactly
  the P09.3 stamp line, benign. P09.6 FEATURES auth row sharpened
  (passkey-only + TLS doctrine); ROADMAP done in P01.2.
- **P08 — observability (done).** usermgmt AuditLog wired at
  construction; `IdentityOps` (enabled/RPID/users/sessions*/audit
  totals + per-action) served as `/api/ops`'s new `identity` section
  (absent on auth-free boots) and rendered as the /ops "Login &
  sessions" card (sessions null on a failed COUNT — never fake zero).
  Tests: `TestIdentityOpsSnapshot`, `TestOpsPayloadIdentitySection`.
  Coverage: internal/web total **72.5%** (floor 70 — no gap-fill owed).
  `BenchmarkConcurrentPollIdentity` added; generation recorded in
  docs/http-stack.md (identity tax = **+165 allocs/op**, +10.5 KB/op;
  ns/op was load noise — the box carried the sibling nix build + the
  concurrent session).
- **P06 — executed by the CONCURRENT session** (appkit v0.9.0 →
  v0.12.0 + systemd v0.1.0 stays; Server-Timing wiring deleted;
  RegisterClassifications at boot; deriveAuthDefaults refactored into
  deriveRPID/deriveOrigin). This pass verified its remaining legs:
  P06.1 preflight (the v0.10–v0.12 CHANGELOG deltas match the executed
  steps exactly), `nix build` EXIT 0 against the bumped go.sum
  (vendorHash already correct), full ladder + lint 0 on the combined
  tree. Devshell check running (job 2DE).

## b) The pass caught four real bugs (all fixed + pinned)

1. **Secure-cookie deploy breaker** (P02/P05 verification): the session
   cookie rode upstream's always-true `Secure` default — browsers drop
   Secure cookies on plain http (localhost exempt), so EVERY login
   outside localhost would silently fail. Fixed: `allOriginsHTTPS`
   derives the flag from the RPOrigins' scheme; pinned by
   `TestIdentityCookieSecureFollowsOriginScheme` (3 cases through a real
   ceremony) + the e2e probe's cookie check. No lane had ever booted
   `-auth-db` before this pass — the bug was invisible to every gate.
2. **RPID/origin mismatch** (P02.2): the sibling module documents
   "RPID derives from publicOrigin's host" but the Go code derived RPID
   from the LISTEN address — an IP/loopback bind against a TLS origin
   breaks every WebAuthn ceremony. Fixed in `deriveAuthDefaults`
   (origin host wins); pinned by `TestDeriveAuthDefaults` (9 cases; the
   test caught a real parsing bug in my first implementation —
   multi-origin strings must split BEFORE url.Parse).
3. **LAN-http doctrine was fiction** (P03/P05 research): login is
   PASSKEY-ONLY — usermgmt ships no password login at all, so "over LAN
   http the password factor works" (DOMAIN_LANGUAGE, TODO_LIST, the
   answer sheet's first draft, CHANGELOG) was FALSE, and a fresh identity
   DB that registers before TLS exists owns an account that can never
   log in. All five docs corrected; the answer sheet's Q1
   recommendation flipped from "A is fine" to "B (TLS) is REQUIRED";
   SECURITY.md + runbook carry the brick warning.
4. **identity.db was invisible to stack backups** (P02.2 eval guard):
   the module staged its backup but never registered it in
   `services.telephony.state.sqliteDatabases` — the host backup tool's
   registry would never see it. Fixed in default.nix; the eval guard
   now pins it.

## c) Not done (gated or carried)

- P01.3 (flip the gated rows) — owner answers outstanding (the answer
  sheet is ready to answer inline).
- P06 ownership: the concurrent session executed the bump; the full
  receipt (goldens/pixel lane through the bumped chain) lands via this
  pass's verify-all run.
- P07 (tc coordinated bump retirements) — gated on the next
  templ-components release post-v1.27.0.
- The 14 owner-gated TODO_LIST rows (unchanged).

## d) Own fuckups this pass (§d norm)

1. **Piped-exit-code false green, AGAIN** (`nix build … | tail; echo $?`
   printed EXIT: 0 while the eval check had failed) — the named
   2026-10-27 trap, repeated despite the summary warning. `${PIPESTATUS[0]}`
   or a redirect-to-file + `$?` from now on.
2. **`edit`-without-fresh-read collisions with the daemon** — the
   auto-commit daemon's mtime touches rejected five `edit` calls before I
   learned to re-stat via View/git diff first (the rule the new
   concurrency.md §7 codifies was being learned live).
3. **A destructive edit on auth_test.go**: I replaced
   `TestAllOriginsHTTPS` with just its signature line (body deleted) —
   repaired two edits later, tests green, but the move was careless; the
   full-function old_string was available and should have been used.
4. **Concurrent-session surprise**: the interleaved session bumped
   appkit (P06) mid-pass, refactoring MY deriveAuthDefaults; my first
   response was to re-verify rather than re-write — correct call — but I
   briefly nearly duplicated P06 before checking `git log`. The
   re-stat-before-write rule now covers commits too.
5. **A multiedit half-landed** on serve_test.go (mtime rejection misread
   as success) — caught by vet (`undefined: filepath`) and by re-reading
   the region; fixed deterministically via python replace with asserts.
6. **golangci-lint drift**: my new code shipped 16 lint findings (wsl,
   err113, gocognit, noctx, gci) — `--fix` cleared the mechanical ones,
   the semantic ones were hand-fixed; the ladder-before-commit rule
   caught them before push, but the code should have been lint-clean at
   write time.

## e) Verification state (at report time)

- `go build ./...` + `go vet ./...` + `go test ./...` — ALL GREEN (full
  suite, combined P03/P05/P06/P08/P10 tree).
- golangci-lint: **0 issues**. cqrs-lint + dedup gate: run inside
  verify-all (below).
- Sibling: `nix flake check` re-run in flight on the formatted tree
  (job 2E6); the eval check itself is EXIT 0 (built standalone twice).
- pbx-lens `nix build` EXIT 0 (appkit v0.12 vendorHash verified);
  devshell check in flight (job 2DE).
- verify-all FULL lane in flight (job 2E7) — ops-verdict ALL PASS
  already visible; the identity e2e + screenshot sweep receipts land in
  /tmp/verify-all.log and the §f block below is finalized when it
  finishes.
- Docs gates: counts 33 live / links 0 broken / test-claims 0 phantom.

## f) Ranked next steps

1. Confirm job 2E6 (sibling flake check) EXIT 0 → receipt-grade sibling
   commit (P02.7) with the eval-guard + registration-fix message.
2. Confirm job 2E7 (verify-all) all-probes-passed → push main (the
   standing owner order) with the daemon-swept history intact.
3. Surface the three owner questions again (TLS/bootstrap/pins) — the
   answer sheet is answer-ready; TLS is now REQUIRED, not optional.
4. After the owner's pins answer: execute P06's residual docs steps
   (the watchlist lines in docs/review/libs/go-appkit.md marking the
   bump DONE) if the concurrent session has not already.
5. On the next templ-components release: P07 (retire the five
   workarounds + Sparkline #60 render check).

## g) Open owner questions (unchanged three, now sharper)

1. **TLS posture** — REQUIRED for any login (passkey-only stack); proxy
   (B) recommended, vhost lands in the sibling module.
2. **Bootstrap posture** — keep first-visitor (A) with the register-
   immediately-through-TLS discipline, or build the provision CLI (B)?
   Decide before the first production identity DB exists.
3. **Stale pins** — hold ~1 week post-deploy (recommendation), then bump
   appkit v0.12 + the cqrs-lite wave together. (The concurrent session
   already bumped appkit — the hold question now reduces to the
   cqrs-lite wave.)
