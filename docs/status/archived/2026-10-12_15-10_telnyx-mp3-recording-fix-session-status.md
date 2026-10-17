# Session Status — Telnyx MP3 Recording Fix — 2026-10-12 15:10 CEST

**Session:** one user question ("Why does upstream have no recording?") turned into a
diagnose → live-verify → fix → test → e2e → document cycle for the Telnyx recording
proxy.
**Trigger:** 404 on `/calls/telnyx-conversation/fa8f0917-a776-4a46-a4f4-00f2a0dd146c/recording`
(432 ms) against the serve on :8300.
**Verdict:** upstream HAS the recording — Telnyx stores AI-assistant calls
**MP3-only** (`download_urls.wav` empty on 9/9 completed recordings) and the proxy
hard-required WAV. Proxy now falls back to MP3; verified end-to-end.

**Format note:** `.md` written per explicit instruction; the status-report skill's
canonical format is styled HTML — flagged as a one-off override, not propagated into
the skill. No manual commit (harness forbids; auto-commit daemon picks the file up).

---

## a) FULLY DONE

| #   | Work                                                                                                                                                                                                                                                                                                                                                           | Evidence                                                                                      |
| --- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------- |
| a1  | Root-causing the 404: the 432 ms duration proved the request reached Telnyx (a missing `RecordingRef` answers in ~2 ms from sqlite), narrowing to the `download_urls.wav == ""` branch — the only 404 that happens _after_ an upstream fetch                                                                                                                   | `internal/web/recording.go:68-72` (fast local 404) vs `recording.go:120-125` (post-fetch 404) |
| a2  | Live-API verification with the real key (read-only): conversation `fa8f0917…` correlates to recording `82a1e992-6205-4cd6-95a9-c6993c3581f7` (`status=completed`, 26 640 ms, dual-channel, `source=call`, `initiated_by=OutboundAPI`), and **all 9/9 completed recordings on the assistant connection carry an empty `download_urls.wav` and a populated MP3** | probe output this session; probe source at `/tmp/telnyx-probe/` (ephemeral)                   |
| a3  | DTO shape verified against Telnyx docs: REST object is `download_urls {mp3, wav}` (our decode was already correct); `recording_urls` belongs to the `call.recording.saved` webhook, not the REST resource — a wrong-field-name bug was ruled out before blaming upstream                                                                                       | developers.telnyx.com call-recordings reference (fetched this session)                        |
| a4  | Fix: WAV preferred, **MP3 fallback rendition**; 404 only when both are empty; `copyRecordingResponse` content-type default now follows the chosen rendition (`audio/wav` vs `audio/mpeg`)                                                                                                                                                                      | `internal/web/recording.go` (`proxyTelnyxRecording`, `copyRecordingResponse`)                 |
| a5  | Regression tests: fake extended with `rec-mp3` (live shape: empty wav, mp3-only, upstream sends **no** Content-Type) and `rec-none` (both empty); new `TestRecordingProxyFallsBackToMP3` + `TestRecordingProxyWithoutAnyRenditionAnswers404`                                                                                                                   | `internal/web/recording_test.go`                                                              |
| a6  | Full test suite green: `go build ./...` + `go test ./...` → **14/14 tested packages ok**, no failures                                                                                                                                                                                                                                                          | session run (load 31.86 < 40 gate ceiling, so full run was allowed)                           |
| a7  | Lint green: `golangci-lint run ./internal/web/...` → **0 issues** (repo `.golangci.yml` incl. `mnd`, matching CI)                                                                                                                                                                                                                                              | session run                                                                                   |
| a8  | **E2E on a disposable copy of the live index** (`/tmp/lens-fixcheck.db` = copy of `/tmp/pbx-lens.db` + WAL, served on 127.0.0.1:8399): the exact URL that 404'd returned **200, `Content-Type: audio/mpeg`, real MP3 bytes** (`\xff\xe3` frame sync). Copy deleted, background serve killed afterwards                                                         | probe2 run this session                                                                       |
| a9  | Docs landed: `docs/telnyx-live-api.md` new section "Recordings are MP3-only on the AI assistant connection (live 2026-10-12)" + intro updated (recordings cluster now live-verified); CHANGELOG `Unreleased → Fixed` entry; AGENTS.md de-staled (see e1)                                                                                                       | the three files                                                                               |
| a10 | Secret hygiene: the API key was read from `~/.pbx-prod-secrets/telnyx_api_key`, never printed or logged; signed URLs presence-checked only (never emitted); the existing key-leak audit test still passes                                                                                                                                                      | probe source; `TestRecordingProxyStreamsTelnyxWav`                                            |

## b) PARTIALLY DONE

| #  | Item | Works                                                                                                                                                                        | Open                                                                                      | Blocker / effort                                                                                                                                                                                |
| -- | ---- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| ~~ | b1   | **The fix is not running anywhere yet.** The serve on :8300 (and any deployed host) still executes the pre-fix binary — the user-visible 404 persists until a restart/deploy | Code, tests, e2e                                                                          | Restart `serve` / deploy the rebuilt binary                                                                                                                                                     |
| ~~ | b2   | Recordings-cluster live verification                                                                                                                                         | Presence, correlation field, shape, status values, offset-less stamps                     | MP3 endpoint **Range support unverified** (does it answer 206? without it, `<audio>` seeking silently degrades to full downloads); WAV-default content-type path asserted only via the MP3 test |
| ~~ | b3   | Quarterly re-verify plan (docs/telnyx-live-api.md)                                                                                                                           | Recordings + core clusters now live-verified                                              | Enrichment endpoints (`conversations-insights`, `detail_records` call-control alias reality) still spec-verified only                                                                           |
| ~~ | b4   | Session bookkeeping                                                                                                                                                          | Report (this file), telnyx-live-api.md, CHANGELOG, AGENTS.md                              | Section (f) → `TODO_LIST.md`/`ROADMAP.md` harvest (docs-health HARVEST) — deferred per "then wait for instructions"                                                                             |
| ~~ | b5   | Probe tooling                                                                                                                                                                | `/tmp/telnyx-probe/` works (conversation metadata, recordings list, per-recording detail) | Lives in /tmp, uncommitted, dies on reboot; not parameterized for other checks                                                                                                                  |

## c) NOT STARTED

Nothing in this session's scope was left unstarted — the asked question is answered
and the fix is complete at the code level. Everything below is _newly noticed_ follow-up
work (full ranked list in f): server-side reason logging for recording-route 404s,
recording-proxy health in `/api/ops`, recordings-lookback audit, MP3 Range verification,
probe promotion, enrichment-endpoint live verification. None had code written before
this session; all are unstarted by design until you prioritize.

## d) TOTALLY FUCKED UP

| #  | What is broken                                                                                                                                                                                                                                                                                                                                                                                              | Severity                                                 | Root cause                                                                                                   | Mitigation                                                                                                        |
| -- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------ | ----------------------------------------------------------------------------------------------------------------- |
| d1 | **The recordings cluster shipped against an unverified upstream shape with a hard WAV assumption.** `docs/telnyx-live-api.md` literally recorded "live verify blocked on a key" — and the WAV-only proxy shipped anyway. The bill came due today: _every_ AI-assistant playback 404'd in production                                                                                                         | High (user-facing feature dead since launch of playback) | Media-format assumption (WAV) encoded without a fixture or live check; unit fakes only model the happy shape | Today's MP3 fallback; lesson recorded in telnyx-live-api.md. Class risk: sweep other unverified DTO fields (f-27) |
| d2 | **Recording-route 404s are operator-blind.** The serve log shows `status=404` only; the distinguishing bodies ("no recording reference indexed" / "no indexed call" / "no WAV or MP3 download") go to the client and nowhere server-side. Diagnosing today required an out-of-band live probe — every future "why no audio?" pays the same archaeology                                                      | Medium (debuggability)                                   | `http.Error` bodies are never logged                                                                         | f-2 (log a reason class; no signed URLs)                                                                          |
| d3 | **Fix/deploy gap:** the repaired binary exists but reaches no running process. The running serve is the user's hand-launched process; nothing restarts it                                                                                                                                                                                                                                                   | High until restarted                                     | No restart step in the session (harness shouldn't kill/restart the user's process unasked)                   | You restart; or the NixOS-module deploy path (TODO_LIST existing item)                                            |
| d4 | _(Suspected, unverified — could be silent data loss)_ **Recordings lookback cap (300)**: ingest correlates against only the 300 newest recordings; a conversation ingested after its recording fell out of that window silently indexes with **no** `RecordingRef` → permanent "no recording reference indexed" 404. 9 recordings exist today so the window is not the problem _yet_, but nothing audits it | Medium (latent)                                          | `DefaultRecordingsLookback = 300`, `internal/cli/telnyx.go`                                                  | f-5 audit: page deeper, or document the bound as a contract                                                       |
| d5 | _(Pre-existing wart, noticed all session)_ go.mod:82 gopls: `templ-components/utils should be direct` — every LSP diagnostic of the session carried it; nobody owns it                                                                                                                                                                                                                                      | Low (noise)                                              | Indirect promoted-but-not-declared dependency                                                                | f-11: `go mod tidy` + `nix run .#update-vendor-hash`                                                              |

## e) WHAT WE SHOULD IMPROVE — and the brutal self-review answers

**What did I forget?**

1. **AGENTS.md was not updated at discovery time** — the memory protocol says update the
   moment you learn something; I updated telnyx-live-api.md + CHANGELOG but left AGENTS.md
   claiming "live verify blocked on a key" and saying "recording WAV stream" — a fresh
   split brain created by my own session. **Fixed during this self-review** (both spots).
2. The MP3 test asserts playback but **not Range passthrough on the MP3 path**, and the
   WAV content-type _default_ (upstream omits the header) is no longer directly asserted
   anywhere — only the MP3 default is.
3. I did not check the **Compression × audio** interaction (the middleware chain is
   auth → Server-Timing → Compression → ETag; ETag skips `/recording`, but whether
   Compression would try to gzip an `audio/mpeg` stream is unverified).
4. The e2e copy instance **fired one immediate Telnyx poll pass** (PollLoop's first pass)
   — harmless (idempotent replays) but avoidable with the poll flags.
5. Follow-through: I told you to restart serve but did not verify any deploy path.

**What is stupid that we do anyway?**

- Shipping against **unverified upstream shapes** when the docs themselves say "unverified"
  (d1). The fixture-fakes encode our guesses, then the tests pass — fake-driven confidence.
- 404 diagnostics that only exist in the HTTP body (d2). Logs must answer "why".
- Ephemeral throwaway probes in /tmp for checks that recur quarterly — the same probe gets
  rewritten from scratch every quarter.

**What could I have done better?**

- Run the **Range check live in the same probe pass** (one extra request) instead of
  leaving it open in b2.
- Started the e2e instance with polling disabled instead of accepting the extra API pass.
- Checked `git status` before finishing: the pre-session untracked status file
  (`2026-10-12_14-16_css-mime-favicon-fix-session-status.md`) is still at the mercy of the
  auto-commit daemon; I never verified it landed.
- Written the AGENTS.md update when I wrote the telnyx-live-api.md section, not two steps
  later during review.

**What could I still improve?**

- The full quality gate (`buildflow --build-mode dev`) was not run; I ran build + tests +
  scoped golangci-lint instead. Deliberate (change surface is one package), but the gate is
  the contract — say so, don't silently skip.
- No `nix build` after the Go change (go.mod/go.sum untouched, no `.templ`/CSS change, so
  vendorHash and codegen are provably unaffected — but the flake build is the shipped
  artifact and should be run once before deploy).

**Did I lie to you?** No. Every number in this report comes from this session's tool
output: 432 ms (your paste), 9/9 recordings (live probe), 14/14 packages (test run),
0 lint issues, 200/audio-mpeg e2e (probe2). The one _suspected_ item (d4) is explicitly
labeled unverified.

**Ghost systems?** None created. Minor note: `fakeTelnyxState.authOnMetadata` is now set
in the `rec-mp3`/`rec-none` fake cases without dedicated assertions there — the existing
audit test covers the shared code path, so it is dead weight at worst, not a ghost.

**Split brains?** One existed (AGENTS.md vs telnyx-live-api.md on verification status) —
closed this session. The WAV/MP3 truth now has exactly one home (telnyx-live-api.md) with
AGENTS.md pointing at it.

**Tests?** Strong for the fallback (regression pinned at the unit level + e2e once by
hand). Gaps: MP3 Range, WAV default content-type, a serve-level (full middleware chain)
MP3 test, and the timeout tests still use budget+sleep pairs (pre-existing TODO_LIST item).

**Scope creep check:** held. One user question → one fix + its tests/docs. The follow-ups
are listed, not built.

**Something useful removed?** No removals this session; the only signature change
(`copyRecordingResponse`) has exactly one caller (verified by grep before editing).

## f) NEXT TASKS (ranked; 33 items — brainstorm, not commitment; HARVEST should route with rigor)

| #  | Task                                                                                                                                           | Impact                                                                                                                                            | Effort | Category |
| -- | ---------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------- | ------ | -------- |
| 1  | Restart serve / deploy the MP3-fallback binary wherever lens runs — the fix is worthless until then                                            | Critical                                                                                                                                          | S      | Deploy   |
| 2  | Log a reason class server-side on recording-route non-200s (no-ref / no-rendition / unknown-id / upstream-status; never signed URLs)           | High                                                                                                                                              | S      | Quality  |
| 3  | Live-verify MP3 endpoint Range support (expect 206 + Accept-Ranges); if absent, handle graceful degradation                                    | High                                                                                                                                              | S      | Verify   |
| 4  | Live-verify the enrichment endpoints with the now-available key (`conversations-insights` shapes, `detail_records` call-control alias reality) | High                                                                                                                                              | M      | Verify   |
| 5  | Audit the recordings lookback cap (300): what happens to conversations older than the window; page deeper or document the bound                | High                                                                                                                                              | M      | Verify   |
| 6  | Pin with a test that Compression never gzips `audio/*` on the recording route (add a skip if needed)                                           | Medium                                                                                                                                            | S      | Quality  |
| 7  | Add recording-proxy health (rendition mix, upstream errors) to `/api/ops` beside the ingest counters                                           | Medium                                                                                                                                            | M      | Feature  |
| 8  | `status != completed` → explicit 503 "recording still processing, try later" instead of empty-URLs 404                                         | Low                                                                                                                                               | S      | UX       |
| 9  | Promote `/tmp/telnyx-probe` into `scripts/telnyx-recording-probe` (read-only, key from file, quarterly aid)                                    | Medium                                                                                                                                            | S      | Quality  |
| ~~ | 10                                                                                                                                             | Harvest this report into `TODO_LIST.md`/`ROADMAP.md` (docs-health HARVEST)                                                                        | Medium | S        |
| ~~ | 11                                                                                                                                             | `go mod tidy` for the templ-components/utils direct warning + `nix run .#update-vendor-hash`                                                      | Low    | S        |
| ~~ | 12                                                                                                                                             | Recording correlation determinism: first-match vs newest when one `call_control_id` has multiple recordings — pick and pin                        | Medium | S        |
| ~~ | 13                                                                                                                                             | Serve-level regression test: MP3 fallback through the full middleware chain (auth/Server-Timing/Compression/ETag ordering)                        | Medium | S        |
| ~~ | 14                                                                                                                                             | Range-passthrough assertion for the MP3 path in recording_test                                                                                    | Low    | S        |
| ~~ | 15                                                                                                                                             | Direct assertion for the WAV content-type default (upstream omits header)                                                                         | Low    | S        |
| ~~ | 16                                                                                                                                             | Detail page: show rendition (MP3) and duration (mm:ss from `duration_millis`)                                                                     | Low    | S        |
| ~~ | 17                                                                                                                                             | Map upstream 404-on-recording-metadata (deleted upstream) to a friendly "recording no longer exists upstream" instead of raw 502                  | Low    | S        |
| ~~ | 18                                                                                                                                             | Scrub upstream error strings bubbled into 502 bodies (recording proxy + telnyx client) for account ids/URLs                                       | Medium | S        |
| ~~ | 19                                                                                                                                             | Observe one long in-progress Telnyx conversation across polls: confirm growth lands as `call.corrected`, never as mutated-rejection               | Medium | M        |
| ~~ | 20                                                                                                                                             | Sweep remaining DTO fields for always-empty assumptions like the WAV one (live-check against real payloads)                                       | Medium | M        |
| ~~ | 21                                                                                                                                             | Document the recording route's 404 taxonomy (no-ref / no-rendition / unknown-id / upstream-404) in AGENTS or docs                                 | Low    | S        |
| ~~ | 22                                                                                                                                             | Verify FEATURES.md/README carry no stale WAV-only claims ("audio" phrasing is already correct — confirm nothing else)                             | Low    | S        |
| ~~ | 23                                                                                                                                             | Run `nix build` once with the new Go sources (vendorHash unaffected — prove it)                                                                   | Low    | S        |
| ~~ | 24                                                                                                                                             | Run the full `buildflow --build-mode dev` gate once before deploy (was scoped-out this session, load allowed it)                                  | Low    | S        |
| ~~ | 25                                                                                                                                             | Local `govulncheck` run (no new deps this session; CI covers, cheap to confirm)                                                                   | Low    | S        |
| ~~ | 26                                                                                                                                             | Confirm auth covers `/calls/*/recording` in the prod htpasswd chain (loopback dev bypass masked it in the e2e)                                    | Medium | S        |
| ~~ | 27                                                                                                                                             | Turn the ad-hoc e2e probe into a Go test helper so "probe the UI" stops being a /tmp one-off                                                      | Low    | M        |
| ~~ | 28                                                                                                                                             | Address the startup WARN noise: `unconsumed event type` for call.corrected/enriched/observed/retired — consume, silence, or document as expected  | Low    | S        |
| ~~ | 29                                                                                                                                             | Document that `-recordings-dir` defaults to a prod path (`/var/lib/telephony/recordings`) that is wrong on a laptop                               | Low    | S        |
| ~~ | 30                                                                                                                                             | Note the WAL-copy caveat (db+wal copy worked in this session's e2e; checkpoint before copy is the safe form) in backup docs                       | Low    | S        |
| ~~ | 31                                                                                                                                             | Recording card could probe availability (cheap HEAD) instead of rendering a player that 404s on play                                              | Low    | M        |
| ~~ | 32                                                                                                                                             | Verify the pre-session untracked file `docs/status/2026-10-12_14-16_css-mime-favicon-fix-session-status.md` got committed by the daemon           | Low    | S        |
| ~~ | 33                                                                                                                                             | Pre-existing TODO_LIST item: harden recording-proxy timeout tests with injected deadlines (sleep budgets remain `-test.shuffle` flake candidates) | Medium | M        |

## g) QUESTIONS I CANNOT FIGURE OUT MYSELF

~~1. **Is MP3-only a fixed property of Telnyx AI-assistant recordings for this account, or~~ resolved-as-documented — telnyx-live-api.md records the MP3-only shape; re-verify 2027-01
is there a recording-format/storage setting** (on the assistant, the connection
`3048844541827221178`, or via custom storage credentials) **you could flip to get WAV —
and do you want WAV at all?** I tried the docs (no format toggle documented for
assistant recordings) and the live API (9/9 empty WAV); account-side settings are
invisible to me. If MP3 is fine forever, the fallback becomes the primary path.
~~2. **Should fetched audio be cached locally after first play** (bounded disk + TTL),~~ NOT-DO — product call never requested; per-play re-proxy is the steady state
or is per-play re-proxying of Telnyx the intended steady state? Every play re-downloads
from Telnyx today (signed URLs expire, `no-store`). A cache adds state to a deliberately
read-only, disposable-index system — product call, not a code call.
~~3. **Will any future Telnyx-recorded calls outside the AI assistant (trunk-side /~~ resolved-as-documented — same cluster (telnyx-live-api.md; 2027-01 re-verify)
FreeSWITCH-via-Telnyx / other connections) carry WAV you rely on?** My sample is one
connection's 9 assistant recordings. If WAV never matters, WAV-first is historical
politeness; if another connection records WAV, the preference order is load-bearing.
This is topology knowledge only you have.

---

**WAITING FOR INSTRUCTIONS.** No manual commit made (harness rule — the auto-commit
daemon picks this file up). Section (f) is HARVEST-ready for TODO_LIST/ROADMAP on your word.
