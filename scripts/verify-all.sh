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
