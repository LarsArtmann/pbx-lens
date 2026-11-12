# Owner Answer Sheet — identity posture (2026-11-12, three answers)

- **Created:** 2026-11-12 (SUPERB plan P01,
  `docs/planning/2026-11-12_05-44_SUPERB-identity-hardening-fleet-currency.md`).
  The native login stack is LANDED and green
  (`internal/web/userauth.go`, cqrs-htmx usermgmt); these three decisions
  gate the deploy path and six downstream rows. Everything else in the
  plan executes without them.
- **How to answer:** write the choice ID (or free-form) after
  **→ ANSWER:** on each question. On arrival, outcomes are recorded into
  TODO_LIST + AGENTS.md stack decisions (plan task P01.3) and the gated
  rows flip to executable.
- **Code truth per question:** `cmd/pbx-lens/serve.go` (`deriveAuthDefaults`,
  `openServeIdentity`), `internal/web/userauth.go` (`MaxUsers: 1`,
  `publicAuthPath`, session middleware), docs/DOMAIN_LANGUAGE.md §
  "Identity and access (2026-11-12)".

---

## 1. TLS for passkeys → ANSWER:

**Current truth (corrected 2026-11-12 during the deploy-receipt wiring):**
the lens serves plain http on the LAN, and **login is PASSKEY-ONLY** —
registration is email-only and usermgmt ships no password login at all.
WebAuthn needs a secure context (localhost, or TLS): over plain LAN http
the login ceremony CANNOT run, so after the first registration the
operator account exists but nobody can log into it — and `MaxUsers: 1`
has already closed the door. The session cookie follows the origin
scheme (`allOriginsHTTPS`): non-Secure over http, Secure once TLS fronts
the lens.

**Options:**
- **A — hold http** is NOT VIABLE for production: it means an instance
  nobody can log into (registration closes on the first signup, whose
  only login path is a ceremony the insecure context refuses). Fine only
  for localhost development.
- **B — TLS reverse proxy in front** (caddy/nginx on the NixOS host).
  Lens stays plain http behind the proxy; the proxy owns certs + HSTS.
  Unlocks: the login ceremony, `Secure` cookie, `-auth-origin
  https://lens.<domain>` (which also seeds the RPID). The vhost snippet
  belongs in the sibling module (`nix-international-telephony`
  `modules/telephony/lens.nix`). Deploy order matters: proxy FIRST, then
  the first registration.
- **C — Go TLS termination in the lens process** (`-cert`/`-key` flags,
  new config surface in serve). No proxy dependency, but cert renewal,
  key handling, and port 443 privileges move into the unit — more ops
  surface for no user-visible gain over B.

**Recommendation: B — REQUIRED, not optional.** The standard posture,
least lens-side code, and the sibling module can grow the vhost in one
commit when you say go. P05's CDP ceremony (P05.6/P05.7) runs on the
localhost posture and is answer-independent.

## 2. Bootstrap posture → ANSWER:

**Current truth:** first-visitor-becomes-operator. On a fresh identity
DB, the first completed `/auth/register` creates the operator and
`MaxUsers: 1` closes the door (every later registration refused —
tested). There is no pre-provisioned account; the boot log witnesses
the open door (`openServeIdentity` logs "the first visitor to /login
becomes the operator").

**Options:**
- **A — keep first-visitor** (implemented, zero new code). Mitigation is
  procedural: register immediately after first deploy, before the
  listener is broadly reachable. Risk: on a shared LAN, whoever hits
  /login first on a fresh DB owns the instance.
- **B — one-shot provision CLI** (`pbx-lens auth provision -auth-db …`
  creating the operator + printing a one-time enrollment URL; serve
  starts with registration closed). The operator identity exists BEFORE
  the first /login hit. Cost: a new CLI surface + a registration-closed
  mode (~a package of work), and a live identity DB migration if you
  flip AFTER the first production deploy.

**Recommendation: A** for the first deploy (trusted LAN, single
operator, door-closing already tested and logged), with B as a named
flip-trigger: any second human needing access, or any deploy on a
network you don't fully trust. Switching after production is possible
but means migrating a live identity DB — cheap today, expensive later,
so decide BEFORE the first real deploy.

## 3. The stale pins — bump now or hold → ANSWER:

**Current truth:** go-appkit v0.9.0 → v0.12.0 (v0.10.0 ships
Server-Timing as default outermost middleware — **deletes** our explicit
`web.ServerTimingMiddleware()` wiring — plus testkit helpers
`WaitForRunning`/`FreeAddr`/`HTTPGetURL`; v0.11.0's go floor 1.27.3 is
already satisfied; v0.12.0 recomposes Service on httputil.Server). The
cqrs-lite wave behind it: system v4.18.0, sqliteengine v4.8.0,
systemscenario v4.3.0. go-etag upstream v0.7.0 is root-only (server
module unchanged on the proxy — filing #2, nothing to bump); cmdguard
is not a pbx-lens dependency. Full migration steps:
docs/review/libs/go-appkit.md (forward findings 1–5), plan package P06.

**Options:**
- **A — hold, one variable at a time on the deploy path:** let the
  identity stack run in production ~1 week, THEN bump appkit + the
  cqrs-lite wave together on the next deps train. Cost: fold-in work
  accumulates slightly (though v0.10 deletes code — the hold is cheap).
- **B — bump now:** the migration is small and fully documented
  (P06.1–P06.6, ~60 min + ladder). Risk: the deploy path carries two
  changes at once (new identity stack + new service shell), so a
  production regression has two suspects instead of one.

**Recommendation: A** — hold ~1 week post-deploy, then execute P06 with
its documented steps. The bump is one decision away either way; P06.1
(the CHANGELOG preflight) is done regardless.

---

**Answered on (date):** ____________ · **Recorded into TODO_LIST + AGENTS by:** ____________

**Rows unblocked by each answer:** #1 TLS → passkey activation +
`Secure` cookie flip + sibling vhost; #2 bootstrap → the provision-CLI
package (if B); #3 pins → P06 (appkit v0.12) + the cqrs-lite bump
train.
