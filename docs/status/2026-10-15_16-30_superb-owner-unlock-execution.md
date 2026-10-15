# 05-40 SUPERB execution status — owner unlock, v0.2.0, hardening road

- **Session:** 2026-10-15 (16:2x close), executing
  `docs/planning/2026-10-15_05-40_SUPERB-owner-unlock-v020-and-hardening.md`
  alongside a concurrent session (the two landed disjoint halves of the
  same plan; every overlap below names the owner).

## a. Released

- **v0.2.0 cut and shipped.** CHANGELOG Unreleased → `## [v0.2.0] —
  2026-10-15`; version const 0.2.0; annotated tag pushed with main
  (`7d15c6f`); GitHub release published from
  `docs/drafts/2026-10-12_v0.2.0-release-notes.md` (updated with the
  10-15 items); module proxy resolves
  `github.com/larsartmann/pbx-lens@v0.2.0`; `nix build` prints
  `pbx-lens 0.2.0`.

## b. Filed upstream (verify-before-filing, voice-checked)

- **BuildFlow#55** — `nix-hash-fix` matches the stale hash by VALUE only,
  first lexical hit wins, so the corrected hash landed in the WRONG
  attribute (the cqrs-lint `fetchFromGitHub` hash) during the 04:06
  incident (corruption a61ba8c, restore 48a63df; same class 2026-10-12).
  Source-verified at `d114f28ce` (`stale_hash.go:53-63` value-only
  `strings.Contains`, no attribute awareness, no dirty-tree guard).
  Draft: `docs/drafts/2026-10-15_buildflow_nix-hash-fix_wrong-attribute.md`.
- **BuildFlow#56** — the gate's managed `go`/`govalid` tools execute
  binaries from BuildFlow's own closure (go 1.27.0), failing after the
  appkit bump moved the floor to 1.27.2 while every repo-side environment
  was already current; `doctor` even prints "go1.27.2 is supported" in
  the same run. Recorded with the receipt in
  `docs/buildflow-dev-gate.md`.
- **vendorHash.nix provenance comment** (F12): hash origin, the
  never-paste rule, and the BuildFlow#55 incident pointer.

## c. Landed code (this session)

- **M15 security bundle:** `/robots.txt` (Disallow all, public by
  design), `SEO.NoIndex` on every page (goldens refreshed — the diff is
  exactly the noindex meta), opt-in `CSP_REPORT_ONLY=1` dry-run mode
  (nonce-aware, same policy as CSP_NONCE), and an explicit
  recording-stream rate limit (2 starts/s, burst 60, loopback exempt,
  bounded 256-host FIFO) with 429 + Retry-After. Tests:
  `internal/web/security_test.go`.
- **M16 perf rails:** `BenchmarkListCalls10k` (36.3 ms/op, 95 KB,
  677 allocs @ load-74), `TestDetail500TurnTranscriptRenders` (far-end
  ?q= anchor lands, no duplication), `TestParseMalformedJSONLNeverPanics`
  + over-ceiling scanner error pin, `BenchmarkETagHashCost` (FNV-64a
  22 µs @ 25 KB / 104 µs @ 122 KB — thousands of pollers per core).
- **M13 hygiene:** AGENTS.md PNG-count wording made rot-proof, README
  teal-evidence line, TestSearchHintProse tripwire decision recorded,
  long-excerpt fixture folded into screenshot-probe as a permanent phase
  (41 PNGs now), smoke help verified adequate (the other session later
  polished it), README screenshots verified current (the changed
  surfaces — hover shade, dark bg — are invisible in static light
  captures).
- **Probe tooling fixes (three real bugs):** verify-staleness-live now
  resets its DB (a reused DB silently skipped the 48h seeding and both
  probe phases failed — pollution, NOT a product regression); phase
  servers in screenshot-probe get their own free ports (the drained
  previous serve held the port through its 5 s shutdown and the
  long-excerpt captures rendered the WRONG phase's DB); inputHash reads
  `static/app.css` at the real path (landed broken at 07:54, fixed
  within the hour).

## d. Landed by the concurrent session (same plan, disjoint tasks)

M9 `/ops` page + stale-pass age, M10 retention card, M11 cache-semantics
tests (+ the strconv break this session fixed on sight), M12
(`verify-all.sh`, `--only` filter, content-hash ensureCodegen), M19
alert-only pixel-diff, F55 print page-breaks, F57 backup opinion
(conclusion matches this session's independent draft), smoke help
wording.

## e. Gate receipt (F7/M6)

- Individual: `go build`, full `go test ./...` (15 packages), 
  `golangci-lint run ./...` (0 issues), `nix build` (exit 0) — all green
  on the tagged tree; all four browser probes ALL PASS
  (`verify-all.sh`); dark dashboard + long-excerpt + 768px tablet
  captures eyeballed (768px: zero horizontal overflow — F54 pass).
- Full buildflow gate: green EXCEPT the two tool-runner steps
  (go/govalid) — the BuildFlow#56 environment issue above, not a repo
  defect; documented per doctrine in buildflow-dev-gate.md.

## f. Docs batch

- `docs/planning/2026-10-15_mos-poller-design.md` (M7: types, fetch,
  idempotence, fixtures, wire points — the 2027-01 window is
  execution-only).
- `docs/decisions/2026-10-15_pii-enforcement-one-pager.md` (M4/F13:
  ingest-vs-read, recommendation = read boundary; owner answer gates
  implementation).
- `docs/process-prep-2026-10-15.md` (M20: dependabot verified, >1MiB
  policy default, re-ingest runbook, host deploy checklist, Telnyx
  window bundle).
- `docs/followons-2026-10-15.md` (M18: cost panel sketch, backup
  rotation opinion, direction-design note closed as landed).
- `docs/upstream-filings.md` quarterly fold-in watch table (M14/F44).

## g. Open / owner-gated

1. **D1–D5 answers** (pack:
   `docs/decisions/2026-10-15_ui-owner-decision-pack.md`) — gates M3
   (EU date format), F52/F53 (phone grouping, CSV headers behind D1).
2. **PII boundary** (F14; one-pager ready) — gates M4 implementation.
3. **Mobile card-vs-scroll taste** (M8/F21) — 375px excerpt visibility
   is the last real responsive gap; 768px is clean.
4. Host deploys + production re-ingest (runbooks ready, owner paths).
5. Telnyx 2027-01 window (calendar m27.9; MOS design pre-staged).
