# Runbook — operator lockout recovery (2026-11-12, plan A16)

The lens has ONE operator account (`MaxUsers: 1` — registration closes
after the first signup, see docs/DOMAIN_LANGUAGE.md § "Identity and
access"). There is no password to reset and no second account to fall
back on, so a lockout has exactly three exits, in escalating order. All
of them are safe for the index: the identity DB is a separate file
(`-auth-db`), and nothing in this runbook touches call data.

## Symptoms

- Login ceremonies refused with the account-locked message after five
  failed WebAuthn ceremonies (usermgmt `AccountLockout` default: 5
  attempts / 15 minutes).
- The browser passkey prompt works but every ceremony answers with the
  lock refusal — that is the lockout, not a broken authenticator.
- `/api/ops` → the identity card (when wired): `audit_actions` shows the
  failed ceremony shape; `sessions` tells you whether any session is
  still alive (if one is, use it — an alive session is unaffected by the
  lock; log in matters only for NEW sessions).

## Recovery ladder

### 1. Wait it out (preferred)

The lock self-expires after 15 minutes. Nothing to do; failed-attempt
counters also decay (stale attempts are evicted). This is the
zero-side-effect exit.

### 2. Restart serve (immediate, honest about its power)

Lockout state is IN-MEMORY upstream (`usermgmt.NewAccountLockout` —
"Lockout state is held in-memory and lost on process restart"). A serve
restart clears the lock AND the attempt counters.

Security note, stated plainly: restart rights already imply root on the
host, so this exit grants nothing an attacker with those rights couldn't
do anyway. It exists so the OPERATOR is never locked out with no path
home.

```console
sudo systemctl restart pbx-lens   # or the deploy unit's name
```

### 3. Re-bootstrap (nuclear: passkey lost AND device gone)

If the operator lost the passkey (device wiped, no surviving
authenticator), no ceremony can ever succeed. There is deliberately NO
reset CLI; the door reopens only by starting a fresh identity DB:

```console
sudo systemctl stop pbx-lens
mv /var/lib/pbx-lens/identity.db /var/lib/pbx-lens/identity.db.locked-$(date +%Y%m%d-%H%M)
sudo systemctl start pbx-lens     # same -auth-db path; a fresh file is created
# first visitor to /login re-registers as the operator (MaxUsers: 1),
# then enrolls a NEW passkey on the replacement device
```

- Move the file, never delete it (it holds the audit journal of the
  locked-out period).
- Every old session dies with the old DB — stale cookies stop resolving
  (the gate answers 401/303), which is the desired posture after an
  account replacement.
- The index (`-db calls.db`) is untouched by all of the above.

## Known limitation (found 2026-11-12, plan A23)

After `/login` succeeds, the loginpage redirects to `/` (fixed
`loginpage.Config{Redirect: "/"}`) — a DEEP LINK clicked while logged out
(like a webphone `/calls/{kind}/{id}` link) lands on `/login` and then
loses its target: the operator must re-click from the list page. Fixing
it needs redirect-target passthrough upstream (loginpage reading a
`?next=` param); tracked as a candidate, not wired locally.

## Deploy stanza (NixOS module docs — wiring itself is TLS-gated)

The sibling deploy module (`nix-international-telephony`,
`modules/telephony/lens.nix`) grows these options when the TLS answer
lands (TODO_LIST identity-posture row; module edit deferred, docs first):

```nix
services.pbx-lens = {
  enable = true;
  auth = {
    enable = true;
    db = "/var/lib/pbx-lens/identity.db";   # NOT the index db
    # TLS-first doctrine (docs/runbooks/identity-first-deploy.md):
    # WebAuthn needs a secure context — behind the reverse proxy the
    # origin is the https URL the browser sees:
    origin = "https://lens.pbx.artmann.tech";
    # rpid defaults to the origin's host; override only for subdomain layouts:
    # rpid = "pbx.artmann.tech";
  };
};
```

Server-side mapping (what the module passes through): `-auth-db <db>
-auth-origin <origin> [-auth-rpid <rpid>]`. The reverse proxy must
terminate TLS and forward Host unchanged (WebAuthn binds the origin
byte-exact). Bootstrap order on a fresh host: TLS up FIRST, then the
first `/login` visit — a registration completed before TLS exists owns
an account that cannot log in (secure-context rule), and the door is
already closed.
