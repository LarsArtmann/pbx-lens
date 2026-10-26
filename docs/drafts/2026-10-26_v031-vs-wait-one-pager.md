<!-- docs/drafts/2026-10-26_v031-vs-wait-one-pager.md
     Owner decision one-pager (plan T3, 2026-10-26): cut v0.3.1 to repair
     the misprinted v0.3.0 tag, or wait for v0.4.0. Decision box unchecked
     by design — the owner owns the tag namespace (17-29 §g territory). -->

# v0.3.1 vs wait — the misprinted v0.3.0 tag (owner decision, 2-minute read)

## What actually happened

The `v0.3.0` tag was cut from a tree where `cmd/pbx-lens/main.go` still
declared `version = "0.2.0"`. Consequences, precisely scoped:

| Surface                                                              | State       |
| -------------------------------------------------------------------- | ----------- |
| Module proxy `go install github.com/larsartmann/pbx-lens/cmd/pbx-lens@v0.3.0` | Correct (proxy stamps the tag, not the const) |
| Binary built FROM the tag (`nix build` at the tag, release artifact) | Prints `0.2.0` on `version` and `/version` |
| Main branch today                                                    | Fixed — const is `0.3.0` (`f5631f2`) |
| Recurrence risk                                                      | Closed — `scripts/cut-release.sh` refuses a tag whose requested version ≠ the const |

No data, no index, no API behavior is affected — the misprint is one
string in the binary's self-report.

## Option A — cut v0.3.1 now

- **For:** any v0.3.0-tagged binary self-identifies correctly; the release
  ledger gets a clean latest tag.
- **Against:** a version number spent on a misprint (the changelog then
  carries two entries for one change-set); another full tag cycle
  (cut-release + CI + release object) for zero behavioral change.

## Option B — wait for v0.4.0 (recommended)

- **For:** zero churn; this repo has NO external Go consumers and both
  deploy hosts build from source (the NixOS module builds the flake, not
  the proxy tag) — nobody consumes a v0.3.0 binary's self-report; the
  misprint is already documented (CHANGELOG Unreleased "Fixed" + FEATURES).
- **Against:** the v0.3.0 tag artifact remains misprinted forever (a
  cosmetic scar visible only to someone who builds that exact tag).

## Decision box

- [ ] **A — cut v0.3.1** (command: `bash scripts/cut-release.sh v0.3.1`; the parity guard passes — const is 0.3.0, so cut-release needs the const bumped to 0.3.1 FIRST, which it will refuse otherwise — by design)
- [x] **B — wait for v0.4.0** (default; the misprint ships no wrong data)

Either way, the release OBJECT publish (`gh release create v0.3.0 …`,
notes paste-ready in `2026-10-26_v0.3.0-release-notes-paste-ready.md`) is
independent of this decision and unblocks the moment the gh token lives.
