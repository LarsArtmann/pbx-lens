# Security Policy

pbx-lens indexes call metadata (numbers, transcripts, recordings references)
for a live PBX — treat the index database as sensitive. The repo itself is
public, but call data never is: the index database, secrets, and any real
call content stay local and are never committed (fixtures use synthetic
numbers only).

## Reporting a vulnerability

Contact the owner directly: lars@artmann.tech (or GitHub private
vulnerability reporting on this repository). Do not open public issues.

Response time: single-maintainer project; reports are triaged as fast as
practical, typically within a few days.

## Supported Versions

Live version: the `main` branch. Tagged releases (starting v0.1.0) are cut
from it and supported until the next tag supersedes them.

## Security Practices

- READ-ONLY over every source: pbx-lens never writes back to the PBX
  runtime, receiver logs, or telephony service units.
- The Telnyx API key is read from `~/.pbx-prod-secrets/telnyx_api_key`
  (0600) and never logged; no secret ever reaches the index database.
- The index DB is disposable: delete and rebuild from sources at any time.
- Participant hashes (sha256-truncated) exist for the future PII policy;
  raw numbers are stored locally only, never exported.
- `/metrics` answers WITHOUT basic auth by deliberate opt-in
  (`AllowUnauthenticated` in `cmd/pbx-lens/serve.go`): appkit mounts the
  scrape endpoint outside the pbx-lens auth middleware chain and refuses
  to start unless auth comes from somewhere — so the deployment provides
  the perimeter instead (loopback bind in dev, firewalled LAN under the
  NixOS module, or an authenticating reverse proxy). The surface carries
  build info and index-shape counters, never call content. Do not expose
  the port beyond a trusted perimeter; if you must, set appkit's
  `Metrics.BasicAuthUser`/`BasicAuthPass` instead of removing the opt-in.

## Identity and access (2026-11-12)

The UI is gated by a native identity stack (`internal/web/userauth.go`,
cqrs-htmx usermgmt) over its own SQLite database (`-auth-db`; the INDEX
db is disposable, the identity DB is NOT — it holds the operator account,
passkeys, and sessions). Every route behind the gate needs a session;
browsers get 303 → /login, API callers a plain 401. `/login`, `/auth/*`,
`/app.css`, and `/robots.txt` are the only public paths.

- **Login is PASSKEY-ONLY.** Registration is email-only; login is a
  WebAuthn ceremony (`-auth-rpid`/`-auth-origin`; with an origin set, the
  RPID defaults to its host). There is no password login and no password
  recovery.
- **WebAuthn needs a secure context.** Over plain http the ceremony runs
  ONLY on localhost — on a LAN deployment there is NO working login until
  TLS fronts the lens. TLS-first is deploy doctrine; see
  docs/runbooks/identity-first-deploy.md and
  docs/decisions/2026-11-12_identity-posture-answer-sheet.md.
- **First-run warning — the first person to complete /login on a fresh
  identity DB OWNS the instance.** Registration closes the moment the
  first account exists (`MaxUsers: 1`); every later registration is
  refused. On a trusted LAN this is the designed bootstrap; the boot log
  witnesses the open door ("the first visitor to /login becomes the
  operator"). Register through the TLS vhost immediately after first
  boot, before the listener is broadly reachable.
- **Brute-force defense:** five consecutive FAILED WebAuthn login
  ceremonies lock the account for 15 minutes (per account; there are no
  passwords to mistype); the ceremonies additionally carry per-IP rate
  limits (10 registrations/min, 30 WebAuthn requests/min). The serve
  auth-failure throttle (per client address) was removed with htpasswd.
- **Session posture:** server-side session rows addressed by an
  HttpOnly SameSite=Strict cookie (`session_token`), Max-Age 24h
  (upstream default, recorded as chosen-for-now in
  docs/DOMAIN_LANGUAGE.md). The cookie rides `Secure` exactly when the
  WebAuthn origin fronts TLS (`allOriginsHTTPS`). Every auth page
  answers `Cache-Control: no-store`; app pages revalidate via
  `Cache-Control: no-cache` + ETag. Logout invalidates the session
  server-side (the nav's Sign out form posts the CSRF-token'd ceremony).
- **Recovery path:** all passkeys lost = stop the unit, delete the
  identity DB, restart, re-register. The identity DB rides the nightly
  backup staging beside the index snapshot.

## Threat model notes (LAN single-operator)

The identity stack is designed for ONE operator on a network you
control, not for hostile tenants:

- **Registration-open-while-empty** is the bootstrap mechanism. The door
  closes at `MaxUsers: 1`, but the window exists: on a shared LAN,
  whoever completes the first registration after a fresh boot owns the
  instance. Mitigations: register immediately (the runbook's step
  order), keep the listener firewalled until you have, or — once the
  posture decision lands — a pre-provisioned one-shot CLI replaces the
  window entirely
  (docs/decisions/2026-11-12_identity-posture-answer-sheet.md,
  question 2).
- **The boot-log witness:** enabling `-auth-db` logs the bootstrap
  posture at startup, and every registration lands as an event in the
  identity journal (auditable: who registered, when).
- **CSRF:** every mutating ceremony (/auth/*) rides the CSRF middleware
  (double-submit cookie + X-Csrf-Token; nosurf underneath) plus
  same-origin attestation checks.
- **Out of scope by design:** multi-user authorization (one operator,
  no roles), password storage (there are none), and public-internet
  exposure (front it with an authenticating proxy if you must).
