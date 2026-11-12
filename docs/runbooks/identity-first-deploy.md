# Runbook: first deploy with native login (identity posture)

**Status:** READY — every step is code-verified (`cmd/pbx-lens/serve.go`,
`internal/web/userauth.go`, sibling `modules/telephony/lens.nix`); the
POSTURE decisions live in
[the identity answer sheet](../decisions/2026-11-12_identity-posture-answer-sheet.md)
and gate steps 4–6. Login is PASSKEY-ONLY (usermgmt ships no password
login), and WebAuthn needs a secure context — so **TLS-first is deploy
doctrine**: a fresh identity DB that registers over plain LAN http owns an
account that can never log in (the registration door closes on that
signup). The identity DB is NOT disposable: unlike index.db it is never
re-derivable, so its backup staging (step 7) is part of the deploy, not an
optional extra.

## The procedure (per fresh host)

1. **Set the module options** (`nix-international-telephony`
   `modules/telephony/lens.nix`):

   ```nix
   services.telephony.pbx-lens = {
     enable = true;
     auth = {
       enable = true;
       publicOrigin = "https://lens.<domain>";  # byte-matches the address bar
     };
   };
   ```

   Eval refuses `auth.enable` without `publicOrigin` (assertion, proven by
   the sibling's `telephony-eval` check). The WebAuthn RPID defaults to
   the publicOrigin's host (`deriveAuthDefaults`); pass `-auth-rpid` only
   for a subdomain RPID spanning multiple origins.

2. **Front the lens with TLS** (answer-sheet posture B): the host reverse
   proxy gets a vhost for `lens.<domain>` → loopback `cfg.address`; the
   proxy owns certs + HSTS and the lens stays plain http behind it. SKIP
   THIS STEP ONLY when the operator browses from the host itself
   (localhost is a secure context too).

3. **Deploy.** The unit boots with `-auth-db /var/lib/pbx-lens/identity.db`
   and logs the bootstrap witness: _"the first visitor to /login becomes
   the operator (registration closes after the first account)"_.

4. **Register through the TLS vhost, immediately.** Open
   `<publicOrigin>/login` in YOUR browser, click "Create one", and
   complete the email-only registration BEFORE anyone else can — the
   first completed signup owns the instance and closes the door
   (`MaxUsers: 1`). The journal (journalctl -u pbx-lens) records the
   registration as an event; a later audit can read who registered when.
   Registering over plain http (step 2 skipped) bricks the login: the
   account exists, the ceremony can never run.

5. **Enroll a passkey** from the logged-in session (the registration
   response carries the session that lets you reach the credentials
   surface). This is the ONLY login factor — there is no password
   fallback. Losing every passkey means the recovery path below.

6. **Verify the cookie posture.** Over the TLS vhost the session cookie
   rides `Secure` (`allOriginsHTTPS` derives the flag from
   `-auth-origin`'s scheme); over plain http it must NOT (browsers drop
   Secure cookies outside a secure context).

7. **Verify backups staged.** After the first night (or a manual
   `systemctl start pbx-lens-backup`), confirm
   `/var/lib/pbx-lens/backups/identity.db` exists beside `index.db` and
   passes the host backup tool's usual checks — the identity DB is the
   one file on this host that cannot be re-derived, and it is registered
   for the stack backups (`services.telephony.state.sqliteDatabases`)
   when `auth.enable` is set.

## Lockout and recovery

Five consecutive FAILED WEBAUTHN LOGIN CEREMONIES lock the account for
15 minutes (per account; there are no passwords to mistype). Distinct
from the per-IP ceremony rate limit (30 WebAuthn requests/min, 10
registrations/min). The honest recovery path for a LOST operator identity
(all passkeys gone): stop the unit, remove `<stateDir>/identity.db`,
restart, and re-register — the identity DB holds nothing else, and the
index DB is untouched. This is also why the identity DB rides the nightly
backup (step 7): restoring it beats re-registering on a shared host.
