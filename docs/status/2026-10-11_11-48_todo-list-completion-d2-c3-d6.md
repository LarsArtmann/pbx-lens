# 2026-10-11 11:48 — TODO-list completion session (D3 finish, D2, C3, D6, D4 docs)

> Continuation of `2026-10-11_09-39_backend-hardening-c2-c4-d5-mos.md` after the owner's
> "GET SHIT DONE — the WHOLE TODO LIST" mandate. A parallel session worked the same repo
> through the window (CI lane, templ v0.3.1070, C5/C6 enrich, D4 docs); coordination
> notes in §f.

## a. What is done now (this continuation)

| Item                                    | Evidence                                                                                                                                                                                                                                                                                                                                                                      | Verified how                                                                                                                                                                                                                                               |
| --------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **D3 finish (m65)** — real MOS card     | `dashboard.go` `MOSKnown/MOSSum` + `DayVolume.AvgMOS`; `dashboard.templ` `mosCard` (avg + per-day BarChart, honest placeholder when unscored, text note when scored-only-outside-window)                                                                                                                                                                                      | 4 tests (render scored, outside-window totals, query aggregation, placeholder preserved); `BarChart.EmptyMessage` only fires on EMPTY bars — all-zero bars render nothing, hence the explicit text branch                                                  |
| **D3 hash contract** — MOS hash golden  | `TestContentHashMOSOnlyWhenReported`: MOS==0 hash equals the pre-MOS algorithm byte-for-byte; reported MOS changes it; different values differ                                                                                                                                                                                                                                | pins the historical-stream-identity contract from the 09:39 session                                                                                                                                                                                        |
| **D2 (m62/m63, T23)** — retention + PII | `internal/retention` (`Policy.Admits` inclusive cutoff, `OldViews`, `Retire`, `Report`); `call.retired` tombstone (domain event + `metaengine.Remove[CallView]()` fold + `RetireCallCmd`); `pbx-lens purge -max-age [-dry-run]` (refuses without the flag; rebuilds FTS after); `?pii=hash` export mode (E164 → `sha256:<16hex>`, non-E164 → `[redacted]`, unknown pii → 400) | unit tests (cutoff boundary, tombstones-only-over-age, re-run convergence, unbounded no-op, **no-resurrection on re-poll**); CLI e2e: seed → dry-run lists → purge retires 1/1 → re-run 0 → `/search` finds nothing → export shows only hashed recent call |
| **C3** — NixOS module landed in sibling | `nix-international-telephony`: `modules/telephony/lens.nix` (`services.telephony.pbx-lens.*`: notify service + watchdog + credentials + hardening, backup + **retention purge** timers), wired in `default.nix` + `state.*`, flake input + package default, enabled on demo host `hosts/pbx`                                                                                  | both hosts eval green; `ExecStart` lines verified materialized; treefmt clean; sibling AGENTS.md + FEATURES.md updated                                                                                                                                     |
| **D6** — five upstream filings          | go-cqrs-lite **#62** (sqliteengine raw filter bind), **#63** (memory engine field-name column resolution), **#64** (cqrs-lint C009 len>4 + E005 system.RegisterCommand), go-nix-helpers **#7** (templ formatter ignores resolved goPkg), numtide/treefmt-nix **#545** comment (already filed upstream; corroborated with our repro)                                           | every diagnosis verified at source against pinned master revs (`f2c892b`, `ad423c8`); Gate-5 duplicate search (only #545 pre-existed); drafts in `docs/drafts/`, all passed the voice checker; review boxes left UNCHECKED for Lars                        |
| **D4 (docs)**                           | FEATURES.md: PII row → PARTIAL, purge row → DONE, dashboard/export rows note `?pii=hash` + MOS card, module-draft note → LANDED; TODO_LIST retention row → DONE (ingest-side `Policy.Admits` gate noted as open enhancement); `deploy/nixos/module-draft.nix` marked superseded; AGENTS.md gotchas + filings policy                                                           | this file + the rows themselves                                                                                                                                                                                                                            |

## b. Verification state

- Full `go test ./... -count=1` green (incl. parallel session's enrich + telnyx).
- `golangci-lint` 0 issues on all touched packages; `cqrs-lint` exit 0 (documented advisories only).
- Full gate: `nix develop -c env BUILDFLOW_NO_RESULT_CACHE=1 buildflow --build-mode dev` — running at report time; result recorded in the session log (§e).

## c. Design decision worth remembering: purges are tombstones, not surgery

metaengine exposes no public record delete; the sanctioned path is a registered remove fold.
`call.retired` keeps streams append-only, replays converge (tombstone is last in its stream),
and a re-poll of a purged call is a ContentHash no-op. Known accepted edge: a CORRECTION
arriving after retirement resurrects the view — the periodic purge pass removes it again
(convergence by policy, not by stream surgery). Full PII erasure remains the disposable-DB
move (drop + re-poll), as AGENTS.md always said.

## d. Paid-for this continuation

1. `templ` on PATH (v0.3.1020) trailed go.mod's v0.3.1070 and failed to parse valid
   syntax (`{ if … }` inside an expression is ALSO invalid templ — two distinct parse
   lessons); generate via `go run github.com/a-h/templ/cmd/templ@<module version>`.
2. `rg -r` is `--replace` — a `-rn "pattern"` silently rewrote output and misdirected a
   diagnosis for a minute.
3. Read-only NixOS options reject even `mkIf`-gated second definitions — `services.telephony.state`
   contributions must go through the central list in `default.nix`.
4. `BarChart.EmptyMessage` renders only when `Bars` is empty, not when all values are 0.

## e. Open items (honest residue)

- **Owner actions**: tick + timestamp the MANUALLY REVIEWED boxes on #62/#63/#64/go-nix-helpers#7;
  prod re-ingest + `pbx-prod` enablement decision (module is demo-enabled only); Telnyx
  live probe of the enrich path (no API key on this machine); SegmentBar swap when
  templ-components publishes it (unchanged).
- **Enhancement (not a gap)**: wire `retention.Policy.Admits` as an ingest-side cutoff in
  the poller so re-ingest converges to the retention window without a purge pass.
- Upstream fixes: when go-cqrs-lite lands #62/#63, the in-repo `rfc3339Nano` binding and
  the sqlite-only test doctrine get revisited; when treefmt-nix lands #545, the flake's
  templ formatter `mkForce` can be dropped.

## f. Parallel-session coordination (what worked)

Division held all session: they owned enrich (C5/C6: `cli/enrich.go`, `ingest/telnyx/enrich.go`,
insight + cost wired into `ingest-telnyx` with double-drain) and D4 docs; I owned D3 finish,
D2, C3, D6. Two collision-free rules carried it: (1) never edit a file another session just
committed to without re-reading it; (2) the daemon's auto-commits sweep BOTH sessions' work —
verify your own edits survive each sweep (`rg` for your symbols) instead of assuming.
