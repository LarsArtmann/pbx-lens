# Runbook: first deploy with native login (identity posture)

**Status:** READY — every step is code-verified (`cmd/pbx-lens/serve.go`,
`internal/web/userauth.go`, sibling `modules/telephony/lens.nix`); the
POSTURE decisions live in
[the identity answer sheet](../decisions/2026-11-12_identity-posture-answer-sheet.md)
and gate steps 4–6. The identity DB is NOT disposable: unlike index.db it
is never re-derivable, so its backup staging (step 7) is part of the
deploy, not an optional extra.

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

2. **Deploy.** The unit boots with `-auth-db /var/lib/pbx-lens/identity.db`
   and logs the bootstrap witness: *"the first visitor to /login becomes
   the operator (registration closes after the first account)"*.

3. **Register immediately.** Open `<publicOrigin>/login` in YOUR browser
   and complete the registration BEFORE anyone else can — the first
   completed signup owns the instance and closes the door (`MaxUsers: 1`).
   The journal (journalctl -u pbx-lens) records the registration as an
   event; a later audit can read who registered when.

4. **(Posture A) stay password-only over LAN http.** Nothing more to do;
   passkeys stay dormant until TLS fronts the lens.

5. **(Posture B) TLS reverse proxy.** Front the lens with the host proxy
   (vhost for `lens.<domain>` → loopback `cfg.address`), THEN the cookie
   should ride `Secure` and passkey enrollment works at
   `<publicOrigin>/login`. The proxy owns certs + HSTS; the lens stays
   plain http behind it.

6. **(Optional) enroll a passkey.** With the secure context in place,
   WebAuthn ceremonies work from the login page; the operator account
   keeps the password factor as fallback.

7. **Verify backups staged.** After the first night (or a manual
   `systemctl start pbx-lens-backup`), confirm
   `/var/lib/pbx-lens/backups/identity.db` exists beside `index.db` and
   passes the host backup tool's usual checks — the identity DB is the
   one file on this host that cannot be re-derived.

## Lockout and recovery

Five consecutive bad passwords lock the account for 15 minutes (per
account; distinct from the per-IP rate limit of 10 register/30 login
attempts per minute). The honest recovery path for a LOST operator
identity: stop the unit, remove `<stateDir>/identity.db`, restart, and
re-register — the identity DB holds nothing else, and the index DB is
untouched. This is also why the identity DB rides the nightly backup
(step 7): restoring it beats re-registering on a shared host.
