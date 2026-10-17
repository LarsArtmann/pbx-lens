#!/usr/bin/env bash
# verify-all — the one-command browser-probe lane (plan 05-40 M12/F34).
#
# Runs every browser-level probe sequentially, each on its own free port.
# The three contract probes take no flags; extra args pass through to
# screenshot-probe (--out, --only, --themes). A failure stops the lane
# with that probe's exit code — first red wins, later probes don't run.
#
# --fast (2026-10-17): contracts-only — skips the 41-PNG screenshot sweep
# for a fast UI sanity check after logic changes; run the full lane before
# tagging or after any visual change.
#
# Usage: bash scripts/verify-all.sh [--fast | screenshot-probe args...]
#   e.g. bash scripts/verify-all.sh --only dashboard --themes light
set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

# statix no-op tripwire (gate-hardening, 2026-10-18): statix exits 0 on a
# config parse error and lints NOTHING — the repo's statix lane once read
# green for five days that way (docs/gotchas.md, CHANGELOG 2026-10-17).
# This lane therefore greps stderr for the no-op signature instead of
# trusting the exit code. Lint VERDICTS stay BuildFlow's job; this only
# asserts the check is not silently dead. Skips when statix is off PATH.
statix_tripwire() {
	if ! command -v statix >/dev/null 2>&1; then
		echo "statix-tripwire: statix not on PATH — skipped (nix develop provides it)"
		return 0
	fi
	local stderr_text
	stderr_text="$(statix check 2>&1 >/dev/null || true)"
	case "$stderr_text" in
	*"config error"*)
		echo "statix-tripwire: FAIL — statix printed 'config error' and exited 0:" >&2
		echo "$stderr_text" >&2
		echo "statix-tripwire: the statix lane is a silent no-op; fix statix.toml (docs/gotchas.md)" >&2
		return 1
		;;
	esac
	echo "statix-tripwire: config parses (no-op tripwire green)"
}
statix_tripwire

if [ "${1:-}" = "--fast" ]; then
	shift
	FAST=1
else
	FAST=0
fi

echo "=== verify-ui-contract ==="
node scripts/verify-ui-contract.js

echo "=== verify-style-contract ==="
node scripts/verify-style-contract.js

echo "=== verify-staleness-live ==="
node scripts/verify-staleness-live.js

if [ "$FAST" -eq 1 ]; then
	echo "verify-all: contracts passed (--fast: screenshot sweep skipped)"
	exit 0
fi

echo "=== screenshot-probe ==="
# Alert-only pixel diff hook: LENS_DIFF_BASELINE=<dir> compares the fresh
# captures against that baseline (never fails the lane). Practiced corpus:
# ~/lens-baselines/<gen>/ (save one with screenshot-probe --save-baseline).
if [ -n "${LENS_DIFF_BASELINE:-}" ]; then
	set -- "$@" --diff "$LENS_DIFF_BASELINE"
fi
node scripts/screenshot-probe.js "$@"

echo "verify-all: all probes passed"
