# Status: identity-hardening execution pass — comprehensive a–g (2026-11-12 12:06)

**Session scope:** this report covers MY execution run of the
[05-44 SUPERB plan](../planning/2026-11-12_05-44_SUPERB-identity-hardening-fleet-currency.md)
(~06:00–07:55 CET, both repos) plus the 12:06 state I found when writing
it. A concurrent session was active throughout and beyond it (its own
05-54 plan, the report-replay lane, gate-chain micro-benchmarks,
rate-limit arithmetic review — commits `5d952bd`/`22e841b`/`d11b51b`,
a 35-file daemon sweep at 08:06). I reconcile where our work touched;
I did not research their lane beyond that.
**Earlier report this day:** [07-43 execution status](2026-11-12_07-43_identity-hardening-execution-pass-status.md)
— this one is the honest a–g audit on top, including what the 07-43
report glossed over.

---

## a) FULLY DONE (verified, receipts banked)

1. **P01.1+P01.2** —
   docs/decisions/2026-11-12_identity-posture-answer-sheet.md (three
   questions, code-truth sections, recommendations); TODO_LIST row links
   it; ROADMAP gained the "Operator identity" raw-ideas block.
2. **P02 eval guards** — sibling tests/eval.nix `lensAuthCheck`
   (auth.enable → `-auth-db`/`-auth-origin` args + identity.db in
   `services.telephony.state.sqliteDatabases`) and
   `lensAuthNegativeCheck` (auth.enable without publicOrigin is rejected
   at eval time, proven via `builtins.tryEval`). The negative guard
   **caught identity.db missing from the backup registration** — fixed
   in modules/telephony/default.nix.
3. **P02 gate** — sibling `nix flake check` **EXIT 0** on the final
   tree (all checks incl. pre-commit's pinned nixfmt v1.5.0 bytes);
   receipt commit `4dbac5d` **pushed**.
4. **P02.5/P02.6** — purge unit comment pins identity.db's exclusion
   from retention; backup unit + timer descriptions fold the
   never-rotates/host-owns-retention opinion AND now name identity.db.
5. **P03** — nav logout form (`navRight`: hidden CSRF field + nonced
   fetch → /login, rendered ONLY when `web.Options.AuthEnabled`);
   `TestIdentityLogoutEndsTheSessionEverywhere` (server-side session
   invalidation + honest nav in both states); goldens refreshed (+37
   chars ×2, reviewed) and green without `-update`; screenshot-probe
   `login` phase (4 PNGs light/dark × mobile/desktop, all 200, styles
   verified applied).
6. **P05** — `scripts/verify-identity-e2e.js`: **15 contracts green**
   (gate 303/401/public paths, no-store, bootstrap 201 + cookie
   posture + /auth/me, door closed, WebAuthn enrollment + login
   ceremonies via CDP virtual authenticator, logout roundtrip, 5-failed-
   ceremonies lockout + its message); wired into verify-all +
   scripts/README.md. csp-sweep /login phase: zero violations, 7
   surfaces.
7. **Two deploy-blocking bug fixes + pins** — `allOriginsHTTPS` (cookie
   Secure follows origin scheme; browsers drop Secure cookies on LAN
   http) and origin-seeds-RPID in `deriveAuthDefaults`; pinned by
   `TestIdentityCookieSecureFollowsOriginScheme` /
   `TestAllOriginsHTTPS` / `TestDeriveAuthDefaults` (9 cases).
8. **Doctrine correction** — login is PASSKEY-ONLY (no password factor
   exists in usermgmt); the false "LAN password login" claims removed
   from DOMAIN_LANGUAGE, TODO_LIST, CHANGELOG, the answer sheet, the
   runbook; TLS-first written into SECURITY.md + the runbook as deploy
   doctrine (register-before-secure-context = bricked account).
9. **P04** — SECURITY.md "Identity and access" + "Threat model notes";
   README auth recipe (htpasswd commands gone); DOMAIN_LANGUAGE
   session-lifetime paragraph (fixed 24h, no sliding, upstream default);
   docs/runbooks/identity-first-deploy.md (7 steps + lockout/recovery).
10. **P10.1–P10.7** — concurrency.md rule 7 (re-stat before
    create/overwrite); verify-all `verify-all: PASS <gate>` verdict
    lines (13 in the final run); docs-gates-in-writing-loop convention
    (AGENTS.md commands); `TestSkipBodyCachingCoversTheRouteTable`
    (parses server.go mux literals, fails on drift AND vacuous parse);
    the drain→rebuild ordering-contract comment in PollOnce;
    staleness-live /health/ready parity; the ANNOTATE receipts block on
    the 2026-11-04 fleet plan (T01–T19 + T24); five TODO_LIST rows to
    DONE/PARTIAL.
11. **P09** — pin spot receipts (usermgmt v4.20.0 + tc v1.27.0 are
    ANNOTATED tags, checkouts ce01d9d3/97c49407; templ v0.3.1100 +
    storage v4.11.1 proxy-verified); filing #60 CLOSED upstream
    2026-11-07 post-tag (ledger row; Sparkline render joins the P07
    list); 2026-11-12 refresh stamps on go-datastar/cmdguard/
    httputil-endpoints; the go-datastar staged-delta examined post-hoc —
    benign (exactly the stamp line); htpasswd residue swept (one live-doc
    fix); FEATURES auth row sharpened.
12. **P08** — usermgmt `AuditLog` wired at construction; `IdentityOps`
    (enabled/RPID/users/sessions/audit per-action) in `/api/ops` +
    the /ops "Login & sessions" card; `TestIdentityOpsSnapshot` +
    `TestOpsPayloadIdentitySection`; coverage 72.5% (floor 70);
    `BenchmarkConcurrentPollIdentity` + the generation recorded in
    docs/http-stack.md (+165 allocs/op, +10.5 KB/op through the gate
    chain).
13. **P06 verification legs** — appkit v0.10–v0.12 CHANGELOG preflight
    (the executed bump matches the documented steps exactly); `nix
    build` EXIT 0 (vendorHash correct); `nix develop -c echo ok` EXIT 0.
    The bump itself landed via the concurrent session.
14. **Final verification** — full Go build/vet/test green; golangci-lint
    **0 issues**; `cqrs-lint .` **EXIT 0** (run at 12:06, see §c5);
    **verify-all EXIT 0 with 13 PASS verdict lines** (dedup gate, 3 docs
    gates, 5 contract probes incl. the new identity e2e, 54-PNG
    screenshot sweep); pbx-lens + sibling **both pushed**
    (`050a4ff` / `4dbac5d`).

## b) PARTIALLY DONE

1. **P02.4 backup staging** — verified by module READING (paths,
   shared StateDirectory, preStart mkdir) but the identity.db backup was
   never RUN against a booted auth-enabled serve (the sibling VM suite
   proves index.db's snapshot; identity.db's `VACUUM INTO` +
   integrity_check is proven only by code identity with the index path).
   A lens-VM extension would close it.
2. **P08.1 counter design** — the plan asked for
   sessions/lockouts/auth-failures/rate-limit-rejects; I shipped
   sessions/users/audit actions only. Lockout state and per-IP
   rate-limit rejects are NOT surfaced (usermgmt's lockout has no
   enumeration API; the rate limiter exposes nothing) — and I did not
   write that gap down as a decision until this report.
3. **P05.10 headers** — `/login` no-store asserted in the probe;
   `/auth/*` hop headers asserted only indirectly (upstream code +
   loginpage's own no-store); no direct HTTP-level assertion for the
   ceremony routes.
4. **P04.5 AGENTS cross-check** — I updated the auth/serve paragraphs
   but never did a systematic contradiction sweep of the whole serve/UI
   section against the new docs.
5. **P09.1 pin receipts** — appkit/httputil/go-etag tag-type checks
   outstanding (row kept PARTIAL in TODO_LIST).
6. **P06 residual docs** — docs/review/libs/go-appkit.md still carries
   the "bump pending" watchlist framing (score 73, WATCH lines); the
   fleet rule says refresh a report on its pin bump — not done (also §c).
7. **Plan-DoD receipt table** — the 05-44 plan says "the session report
   repeats the §2/§3 tables on completion"; the 07-43 report covers
   packages, not all 59 tasks individually. Deviation stands.

## c) NOT STARTED

1. **P07** (tc coordinated bump: retire 5 workarounds + #60 Sparkline
   render check) — gated on the next templ-components release.
2. **P01.3** (flip the gated rows on owner answers) — the answer sheet
   sits ready; TLS is now REQUIRED not optional.
3. **Quiet-box bench re-measure** — this pass's ns/op numbers were
   load-noise; the allocs stand, the µs comparisons await a
   poll-gate-verified window.
4. **Pixel-baseline refresh** — the corpus grew 50 → 54 PNGs (login
   surface); no new baseline saved; the next `--diff` against an old
   baseline will alert on every login PNG.
5. **docs/http-stack.md ordering doctrine rewrite** — the ExtraMiddlewares
   chain changed on the appkit bump (Server-Timing now appkit-default
   OUTERMOST, auth inside it); the doc's ordering section still
   describes the old chain. I edited the bench paragraph but NOT the
   ordering doctrine. Same staleness in AGENTS.md's chain line
   ("auth → SecurityHeaders → Server-Timing → Compression → ETag").
6. **screenshot-probe + verify-all README rows** — scripts/README.md
   still says "36 PNGs / 9 surfaces" and "three contract probes"; the
   corpus is 54 PNGs + the identity probe.
7. **Golden/render coverage for the identity ops card** — the card
   compiles and its snapshot is tested, but no test renders /ops with
   `Options{Identity}` wired; the card has zero render-level coverage.
8. **Answer-sheet Q3 staleness** — the pins recommendation says "hold"
   while appkit is ALREADY bumped (concurrent session); the sheet's
   question shrinks to the cqrs-lite wave but the text wasn't updated.
9. **The 14 owner-gated TODO_LIST rows** — untouched, gates unchanged.
10. **TODO_LIST sibling rows** — row 153 (backup-rotation fold) and row
    140 (deploy hosts) not annotated with their 2026-11-12 receipts.

## d) TOTALLY FUCKED UP (own failures, ranked)

1. **Piped-exit-code false green, TWICE in one session** —
   `nix build … | tail; echo $?` reported EXIT 0 while the eval check
   had FAILED, and the sibling-check echo pattern repeated it. The named
   2026-10-27 trap, repeated despite the handoff warning. Redirect-to-
   file + `$?` (or `${PIPESTATUS[0]}`) is now the only acceptable shape;
   the concurrency rule 7 I wrote covers re-stat, not this — the pipeline
   rule needs its own line.
2. **Destructive edit on auth_test.go** — replaced
   `TestAllOriginsHTTPS` with only its signature line (body deleted) by
   sloppy old_string choice; repaired two edits later, but the full
   function text was available and should have been used. No gate caught
   it (compile broke only on the NEXT edit) — luck, not process.
3. **Multiedit half-landed confusion** — an mtime-rejected multiedit
   that I misread as partially applied; I then made three speculative
   edits against a file state I hadn't re-verified. Cost: a detour of
   ~5 tool calls before a deterministic python-with-asserts rewrite.
   Rule: after ANY mtime rejection, re-View and re-plan, never
   chain-guess.
4. **Lint-dirty code shipped to lint** — my new code accumulated 16
   golangci findings (wsl ×5, err113 ×2, gci ×2, gocognit, gocritic,
   noctx, golines, dupword-adjacent) discovered only at the pre-commit
   ladder. All fixed, but the standard is lint-clean at write time.
5. **Botched commit message** — a broken heredoc produced a commit with
   the subject `:` (amended immediately since unpushed, but it shipped a
   garbage message to the reflog and cost a cycle).
6. **Almost duplicated the concurrent session's P06** — I started
   planning appkit bump steps before checking `git log` and finding it
   done. Caught in minutes, but the check belongs BEFORE reading the
   package, not after (now codified in concurrency rule 7's paired
   clause).
7. **`TestIdentityCookieSecureFollowsOriginScheme` nil-origins case was
   unreachable** — I wrote a case OpenIdentity can never reach
   (WebAuthn refuses empty RPOrigins at construction); the test failure
   taught me the boundary. Wasted a cycle; the fixture constraint should
   have been checked first.
8. **I published the concurrent session's in-flight docs edits** — my
   push included daemon-swept commits of THEIR review-doc edits I had
   not verified (docs-only, low risk, and their lane continued — but I
   pushed bytes I did not vouch for; the concurrency norm tolerates it
   for daemon commits, still worth naming).

## e) WHAT WE SHOULD IMPROVE

1. **A "full ladder" checklist exists but wasn't a checklist.** The
   plan's DoD lists build/vet/test + golangci + cqrs-lint + dedup +
   verify-all + nix; I ran build/vet/test/lint/dedup/verify-all and
   MISSED cqrs-lint until 12:06 (green, but missed). Make
   scripts/verify-all.sh carry the Go-side gates (or add a
   scripts/check-go-gates.sh) so "the ladder" is one command.
2. **Every lane that boots serve should boot `-auth-db` in at least one
   phase** — the Secure-cookie bug survived a week of green gates
   because every probe boots auth-free. The identity e2e now does; the
   screenshot corpus + staleness/ops probes still don't exercise the
   gate chain.
3. **Render-level golden coverage must accompany new templ cards** —
   the identity card shipped with snapshot+payload tests only.
4. **Update the docs IN THE SAME EDIT as the behavior** — three stale
   rows (screenshot README count, verify-all probe count, http-stack
   ordering) exist because the script/code changed in one session and
   the doc row in another. The docs-gates-in-writing-loop convention
   needs a "same-edit" corollary for count-bearing rows.
5. **Concurrency etiquette for pushes** — pushing daemon-swept commits
   of another session's edits is tolerated but unverifiable; a short
   `git log origin/main..HEAD --stat` scan before push (which I did for
   CONTENT, not for attribution) should become an explicit habit.
6. **The 59-task table → receipt mapping** — future plans should either
   drop the "repeat the tables in the report" DoD or generate the
   receipt table mechanically (the ANNOTATE block pattern works well
   for this).
7. **Fixtures-before-cases** — check what the constructor forbids
   BEFORE writing table cases (the nil-origins lesson generalized).

## f) Up to 50 things to do next (roughly in impact order; ★ = owner-gated)

1. ★ Answer the three identity-posture questions in
   docs/decisions/2026-11-12_identity-posture-answer-sheet.md (TLS is
   REQUIRED; TLS answer unblocks 1–4 below).
2. Rewrite docs/http-stack.md's ordering doctrine for the appkit v0.12
   chain (Server-Timing default outermost, auth inside; re-derive the
   two-hop budget statement) + fix AGENTS.md's chain line.
3. Refresh docs/review/libs/go-appkit.md for the v0.12 pin (WATCH lines
   → receipts, re-score) + the fleet index row.
4. Update scripts/README.md: screenshot-probe row (54 PNGs, login
   surface) + verify-all row (five probes, PASS verdict lines) +
   screenshot corpus count wherever FEATURES/docs claim it.
5. Save a fresh pixel baseline including the 4 login PNGs
   (screenshot-probe --save-baseline ~/lens-baselines/<gen>).
6. Add a render-level test for the /ops identity card (Options{Identity}
   wired, assert the card markup) + golden if the house pattern wants
   one.
7. Extend the sibling lens VM test (tests/lens.nix) with auth.enable:
   boot with `-auth-db`, run `pbx-lens backup` on identity.db, assert
   integrity_check — the P02.4 runtime proof.
8. Direct HTTP-level header assertions for `/auth/*` hops in
   verify-identity-e2e (register/logout/webauthn begin) — close P05.10.
9. Add the /health/ready parity to the remaining booting probes
   (csp-sweep's login phase, screenshot-probe login phase).
10. Surface lockout/rate-limit observability or file the upstream ask:
    AccountLockout enumeration + limiter counters (P08.1's honest gap).
11. ★ Bootstrap answer: if provision-CLI (B) — design + build
    `pbx-lens auth provision` (one-shot operator + registration-closed
    mode) as a new package.
12. ★ TLS answer: if proxy (B) — the vhost snippet in the sibling lens
    module (proxy choice + hostname + HSTS), then the runbook's step 2
    becomes copy-paste.
13. ★ Pins answer: cqrs-lite wave bump (system v4.18, sqliteengine
    v4.8, systemscenario v4.3) as the next deps train — with the appkit
    report refresh (item 3) folded in.
14. Run the ingest benchmarks on a quiet window (poll-gate-verified) and
    replace this pass's load-noise ns/op notes in docs/http-stack.md.
15. Annotate TODO_LIST row 153 (backup-rotation fold) + row 140 (deploy
    hosts) with their receipts; re-check the row-140 gate.
16. Verify the answer-sheet Q3 text against the appkit-already-bumped
    reality (shrink to the cqrs-lite wave).
17. CHANGELOG editorial: move the two deploy-blocking bug fixes from
    "Added b" into a "Fixed — 2026-11-12" section.
18. Systematic AGENTS.md ↔ new-docs contradiction sweep (finish P04.5
    properly).
19. scripts/README.md + verify-all header: document the identity probe's
    runtime cost so the lane budget is known.
20. Measure + record verify-all's total runtime pre/post identity probe.
21. Add cqrs-lint + govulncheck to the standard pre-push checklist doc
    (verify-all carries dedup only).
22. Probe the `deriveAuthDefaults` edge (origin without scheme) and
    either document it as unsupported or pin it with a test.
23. P07 when the tc release lands: retire the five workarounds +
    duplicate `</script>` fold + Sparkline #60 render check + adoption
    map steps 2–4+7 (the standing watchlist).
24. Extend verify-identity-e2e with a passkey DELETE ceremony (the
    credentials surface) — enrollment covered, removal not.
25. WebAuthn ceremony under the enforcing CSP (CSP_NONCE=1 without
    REPORT_ONLY) in csp-sweep's login phase — the report-only sweep
    proves the policy; enforcing proves the flip.
26. Add `--only login` corpus usage to the eyeball pass doc so the four
    login PNGs get eyeballed per bump, not once.
27. Eyeball the dark/mobile login PNGs explicitly (P03.2's drift check
    was desktop-light only).
28. Lockout UX prose review: the panel's lock message text — confirm it
    names the wait, not a user fault, in the PANEL rendering (the probe
    asserts the API body; the rendered message inherits it, but an
    explicit DOM assertion is cheap).
29. Rate-limit arithmetic review follow-up: the concurrent session's
    22e841b touched the lockout runbook — reconcile my runbook's
    lockout paragraph with theirs (one source of truth).
30. Consider surfacing `identity.Ready()` in /api/ops (readiness is
    gate-relevant; currently only /health/ready carries it).
31. Wire the ops page's identity card into ops-wired screenshot variant
    (an auth-boot ops capture) if operator-visible truth wants it.
32. Re-run `node scripts/replay-report-verify.sh` (the concurrent
    session's lane) over the review reports THIS pass edited
    (go-datastar/cmdguard/httputil-endpoints stamps) — verify my stamp
    lines' verify-commands still replay clean.
33. Check the login corpus PNGs against the palette/style contract
    (verify-style-contract covers app pages; /login is loginpage-owned —
    decide whether the teal-bridge contract applies there).
34. Docs: name the identity e2e probe in AGENTS.md's verify-all comment
    (it says "five probe suites" — matches now, keep in sync).
35. Sweep FEATURES.md for any remaining count-bearing rows the corpus
    growth invalidates (36→54 PNG claims).
36. Decide + record whether `login` should join the ops-wired-style
    seeded variants or stay first-visitor-fresh (current: fresh; a
    post-registration variant would show the passwordless prompt).
37. Upstream watchlist: verify the four pin receipts' `git tag --contains`
    form once more at the next deps train (T14 style, full sweep).
38. Filing #60 follow-through: when the tc release carries it, verify
    the Sparkline min/max sign fix against the spend card render.
39. Add the identity DB to the restore-drill test if one exists in the
    sibling backup suite (index.db is proven; identity.db restore is the
    P02.4 runtime twin).
40. ops.go `Wired` semantics: an auth-only boot (no poller/cache) shows
    "No counters are wired" empty state AND the identity card is skipped
    (card sits inside the Wired branch) — decide whether an auth-only
    boot should still show the identity card; currently it cannot.
41. Re-check `TestOpsPageBodyIsDeterministic` under AuthEnabled (the
    logout form's csrf token is per-session — the determinism harness
    boots auth-free; an authed determinism pin would need a frozen
    token).
42. Guard the `login` screenshot phase against the parallel-port race
    the empty phase paid for (it uses findFreePort — fine — but the
    doc-worthy trap note belongs in scripts/README.md).
43. Consider `-auth-rpid` passthrough in the sibling module (currently
    origin-derived only; a subdomain-RPID deploy needs the flag) —
    module option + eval guard.
44. Review deriveRPID against multi-origin subdomain setups (RPID must
    be the registrable suffix shared by ALL origins; first-origin host
    is a simplification — pin or document).
45. Add the identity DB file count/size to the backup verification step
    of the runbook (operator-checkable facts).
46. Write the "what a fresh deploy looks like" screenshot set (login
    first-visit → register → enrolled → authed nav) as a doc figure
    trail.
47. Check whether `opsSessionsText`'s unknown-state renders anywhere
    reachable (a failed COUNT is rare; a seeded test would pin the
    honesty path).
48. Keep scripts/verify-all.sh's `pass()` line inventory in sync with
    any new probe (a drift guard: count `===` headers vs PASS lines).
49. Reconcile the 07-43 report's §f with this report's §f (supersede
    explicitly in docs/status/README.md annotations).
50. Archive-queue check: three status reports now share 2026-11-12 —
    when docs-health next runs, decide which annotate vs archive.

## g) Three questions I cannot answer myself

1. **TLS deploy shape:** passkeys REQUIRE TLS — which proxy fronts the
   lens on the PBX host (the stack's existing nginx vs a new caddy),
   what hostname should the lens vhost bind (`lens.artmann.tech`? a
   pbx-prefixed subdomain?), and is ACME already the cert path there?
   This decides the sibling-module vhost commit AND the
   `-auth-origin`/RPID values.
2. **Bootstrap on the FIRST production deploy:** with TLS-first doctrine
   the registration window is one browser session behind the proxy — do
   you accept first-visitor (register immediately through the vhost, my
   recommendation), or do you want the `pbx-lens auth provision` CLI
   built BEFORE any real deploy so the operator identity exists pre-boot?
3. **cqrs-lite wave timing + report refresh scope:** the appkit bump
   landed today via the concurrent session; do you want the cqrs-lite
   wave (system v4.18, sqliteengine v4.8, systemscenario v4.3) on the
   next deps train now, or after the identity stack has run in
   production for a week (my original hold recommendation) — and should
   the appkit report refresh + fleet index re-score ride that same train
   or happen immediately?
