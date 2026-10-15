# Process prep checklists (plan 05-40 M20, F62–F65) — 2026-10-15

Three bounded prep items landed as one page; each is executable the moment
its external gate opens. F61 (dependabot) was verified separately.

## F61 — dependabot confirmation (VERIFIED 2026-10-15)

`.github/dependabot.yml` watches `github-actions` + `gomod`, weekly,
grouped. The newly-direct deps (`httputil`, `server_timing`,
`go-etag/server`) were promoted 2026-10-12; the weekly cycle proposes
bumps for anything newer than the pin — nothing to do beyond glancing at
the next dependabot PR wave. The watch row in TODO_LIST closes with this
note; the dependabot-core#16538 owner decision stays with the owner.

## F62 — >1 MiB transcript policy (one-pager, owner decides)

- **State:** the ingest scanner caps a single line at 1 MiB
  (`internal/ingest/ingest.go` NewScanner); the web render buffer warns
  via `OnBufferOverflow` past its limit; exports stream unbounded.
- **Option 1 (recommended):** accept untagged — a >1 MiB transcript line
  is a pathological capture (a stuck ASR loop), not data to preserve; the
  scanner error names the file+line, the operator re-ingests after
  trimming. No code.
- **Option 2:** raise the ceiling (e.g. 8 MiB) — costs bounded-memory
  guarantees everywhere the scanner is shared, to serve a shape no real
  Telnyx/FreeSWITCH source produces.
- **Owner answer needed:** none expected (default = option 1 stands
  unless a real capture ever trips it).

## F63 — production re-ingest prep (ContentHash v2 runbook)

Runs ONLY when the owner picks a hash-changing change (ContentHash v2 or
PII option A). Checklist:

1. Announce a maintenance window (the index is disposable; serve can stay
   up — ingest is idempotent and re-runs are safe).
2. `pbx-lens backup -db …` (verified snapshot first, always).
3. Stop pollers (`systemctl stop pbx-lens-ingest…` or the module timers).
4. Move the old DSN aside (`calls.db` → `calls.db.v1`), create fresh.
5. Re-run every ingest command against the sources (CDR Master.csv files,
   Telnyx `-since` deep window, webhook JSONL, transcripts dir).
6. Verify: row counts per source vs `/api/ops` counters; spot-check three
   known calls (deep links keep working — stream ids are source-native).
7. Re-point serve at the new DSN; `restore-drill` test is the CI-level
   proof the chain works (`internal/backup/restore_drill_test.go`).
8. Keep `calls.db.v1` for one backup rotation, then host-tool purge.

## F64 — host deploy prep (aiDids + module options)

Both hosts, current module (nix-international-telephony
`modules/telephony/lens.nix`):

1. Set `services.telephony.pbx-lens.aiDids = [ … ]` on the AI-facing
   host (empty = old behavior; the option is live since e7c0f8b).
2. Confirm `-max-age` / `-recording-timeout` module values (verified
   exposed 2026-10-12; `0` keeps everything).
3. Backup + purge timers: confirm `services.telephony.pbx-lens.backup.*`
   targets exist on the host's storage path.
4. `nixos-rebuild … switch` + `systemctl status pbx-lens` (Type=notify —
   READY under the manager) + one `/health/ready` probe.
5. LoadCredential path check: the htpasswd secret lands via
   `environment.etc` or sops — per the module's current contract.

## F65 — Telnyx 2027-01 re-verify

Calendar entry exists (owner calendar, m27.9; next window opens 2027-01).
The session bundle for that window: docs/telnyx-live-api.md cluster
re-verify + the MOS poller design (docs/planning/2026-10-15_mos-poller-
design.md) executes with live DTOs. Nothing else pre-staged.
