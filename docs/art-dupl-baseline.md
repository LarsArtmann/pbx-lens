# art-dupl baseline — the duplication gate

The CI `duplication` lane is a **gate, not prose**: `scripts/art-dupl-baseline.json`
pins every accepted clone group, and `art-dupl check` fails the build on any
group whose hash is absent from the baseline — newly introduced duplication.
An enforced count cannot be relitigated; a prose note can (the go-etag
pattern).

Since 2026-10-24 (second pass) the baseline holds **zero groups**: the gate
contract is "no accepted clone groups at all — any new multi-line clone group
goes red."

## Gate mechanics

```console
art-dupl check     -t 4 --min-lines 2 . --baseline-path scripts/art-dupl-baseline.json  # CI (red on new groups)
art-dupl baseline  -t 4 --min-lines 2 . --baseline-path scripts/art-dupl-baseline.json  # refresh after a reviewed change
```

- **t=4** — the fleet production gate (duplicated statements).
- **--min-lines 2** — the calibration that drove the report to zero. Every
  1-line group art-dupl finds in this repo is a signature/type-expression
  match (parameter lists like `(v string)`, return types like
  `templ.Component`, two-field struct declarations) — shape coincidence
  between functions with different bodies, not duplication. The gate
  therefore only contracts on clones that span real lines; both directions
  are proven (clean tree → exit 0, seeded 12-statement cross-package clone →
  exit 2, 2026-10-24).
- **Directory arg (`.`) is load-bearing.** Explicit file lists shrink the
  detection universe: `.templ` files are source and must be scanned;
  `*_templ.go` generated code stays auto-excluded. The retired text-baseline
  lane never saw the .templ source at all.
- **The suppressed bucket is not the gate's blind spot — it was, once.** The
  default report hides two buckets ("non-actionable" test scaffolding and
  "filtered suppressed"); the 2026-10-24 second pass found four REAL
  production clone groups hiding in the suppressed bucket (invisible to
  `check`), extracted them, and spot-checked the rest (test scaffolding,
  self-contained tests by design). After any pin bump or suspicious green,
  audit the bucket once with
  `art-dupl --type-aware -t 4 --show-suppressed .`
  before trusting it.
- **Pin parity.** CI installs `art-dupl@93432fd` (the fork commit that
  generated this baseline — the v0.9.0 tag predates the `.templ` code-fence
  detection fixes and false-positived on 2-3 line markup fragments, CI run
  40165045608 on 2026-10-25); after a pin bump, regenerate the baseline in
  the same change (suppression buckets shift between versions, so a bump can
  surface old groups as "new" — that red is the pin asking for a review).
  **The binary must BE the pin, not merely exist** (2026-10-25): the devShell
  resolved to a stale user-local `~/go/bin/art-dupl` (version "dev") that
  missed a real 10-line clone AND flagged a 1-token comment-vs-struct
  "clone" (type-1, "1 token, 2 lines" — a distinctive word shared between a
  multi-line comment and a struct's field lines) that the pin does not see —
  wrong in both directions, and it cost a wasted extraction round. Verify
  with `art-dupl version | grep 93432fd`; `check-dedup-gate.sh` enforces
  this (mismatch fails with the install fix) and CI's Guard Self-Tests job
  installs the pin for the same reason.

## Zero-group state (2026-10-24, second pass)

The four production clone groups the first pass missed (all suppressed from
the default report, all in `internal/web`) were extracted:

| Was (tokens, lines)                                         | Now                                                                                                                                                                                   |
| ----------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `serveStyles`/`serveHtmx`/`serveFavicon` trio (3×~10 lines) | One `staticAsset` struct (path/contentType/missingMsg) + one `serve` method; the three assets are documented data values (`stylesAsset` etc.) wired straight into route registration. |
| `list` vs `stalenessFragment` bodies (2×19 lines)           | `queryListStaleness` returns a typed `listStaleness` value (filters/page/staleFor/stale — a struct, so the two trailing bools cannot transpose); both handlers parse-query-render.    |
| `dashboard` vs `dashboardFragment` bodies (2×15 lines)      | `queryDashboardStats` (stats/fragURL/ok); both handlers only render.                                                                                                                  |
| `exportJSONL` vs `exportCSV` format preamble (2×7 lines)    | `beginExportResponse(w, r, contentType, disposition)` absorbs the Content-Type/Disposition pair after the PII gate; call sites are one guarded call each.                             |

What deliberately remains, and why:

- **1-line signature/type matches (~21 groups, invisible at `--min-lines 2`)**:
  same signatures, different documented business rules — `mosBars` (empty
  slot stays, never a 0.0 dip) vs `volumeBars` (all-zero series suppressed);
  `noTranscriptTitle` vs `noTranscriptDescription` (distinct copy semantics);
  `percent` (ratio format) vs `mosGaugePercent` (gauge clamp). Reviewed body-by-body in the
  2026-10-24 audit; an abstraction would take more parameters than each
  duplicated line count.
  **Superseded 2026-10-24 (later the same day):** the templ-components deep
  dive (docs/research/2026-10-24_templ-components-deep-dive.html, F1) found
  `utils/format` at the pin ships `ClockDuration`, `CompactDuration`,
  `StringOrDash`, and `Percent` — so the morning's acceptance of
  `clockDuration`/`compactDuration` (and the local `percent`) as intentional
  shape coincidence was wrong: they were undetected dependency duplication.
  All three locals were deleted when the library package was adopted (T4 of
  docs/planning/2026-10-24_19-32_SUPERB-data-truth-and-ui-debt-plan.md);
  only `mosGaugePercent` remains, and it stays — it maps MOS onto the gauge
  scale (domain math), it does not format.
- **Test scaffolding (the suppressed/non-actionable buckets)**: per-test
  `OpenSystem`+`Fatalf` bootstraps, `ctx`/`t.Parallel`/fixture setup pairs,
  `newTestServer`/`openTestIndex` prologues. Self-contained tests are the
  design (audit item 19/20); real full-stack repetitions are extracted when
  they appear (`serveTestStack` in internal/web, 2026-10-24; `openPollHarness`
  in internal/cli, third pass — a verbatim 27-line fixture block invisible to
  `check` because test files are auto-excluded, yet a real drift risk).

If a group's code changes shape, its hash changes and the lane goes red even
though the count is unchanged — that is the desired strictness: a verdict
change deserves eyes, then a reviewed refresh.

## History

- 2026-10-18: `scripts/art-dupl-baseline.txt` (text diff, CI **warning-only**,
  file-list mode `.go`-only) — retired 2026-10-24: warnings could be ignored,
  the .templ source universe was invisible, and the text was generated from a
  different file universe than CI's installed pin could agree with.
- 2026-10-24 (first pass): JSON baseline + `check` gate pinning 23 accepted
  groups; the one harmful clone from that day's audit (`serveTestStack`
  extraction) verified gone before the baseline was recorded.
- 2026-10-24 (second pass): audited the suppressed bucket, extracted the four
  hidden production clone groups above, recalibrated with `--min-lines 2` —
  baseline now records zero groups; the gate means "no clones accepted."
- 2026-10-24 (third pass): a t-10 `--show-suppressed` audit surfaced a real
  27-line test clone in internal/cli (identical system + webhook-fixture +
  index harness copied across both poll-pass tests) — invisible to the gate
  (test files auto-excluded) but a genuine drift risk. Extracted as
  `openPollHarness` (+ `webhookHangupLine`), fixture bytes unchanged; the
  third inline literal in the counters test moved onto the shared
  `webhookPollLine` helper. Gate verified still at zero groups.
- 2026-10-24 (same-day correction): the utils/format adoption (plan T4)
  deleted the `clockDuration`/`compactDuration`/`percent` locals this doc's
  deliberate-remainder list had accepted hours earlier — grep the dependency
  graph BEFORE accepting a signature pair as shape coincidence (lesson
  recorded; see the annotated bullet above).
- 2026-10-25: the gate caught its own author — three pushes (session's four
  features) went red in CI because the gate ran nowhere locally (the FULL
  receipt predated the features; golangci-lint sees none of this). Fixed by
  extraction (`normalizedQuery` in internal/search; `firstMatchedTurn` via
  `slices.IndexFunc`), and the gate moved left: `check-dedup-gate.sh` now
  runs in `.githooks/pre-commit` (pin-enforcing, self-tested in CI's Guard
  Self-Tests), and the stale-dev-binary trap above was documented the hard
  way. Pinned-binary verification: pristine tree reproduces CI's exact
  group; fixed tree reports 0 groups; CI run 40221643859 green.
