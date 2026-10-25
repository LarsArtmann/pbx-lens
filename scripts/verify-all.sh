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
# --baseline <dir> (2026-10-18): alert-only pixel diff against that baseline
# directory (same hook as LENS_DIFF_BASELINE, as a lane flag). Both lane flags
# are parsed before the first probe, so a bad argument fails in a second.
#
# Before any probe, the lane runs the non-browser gates: the statix no-op
# tripwire, the docs-count gate and the doc-link gate (both self-tested).
#
# Usage: bash scripts/verify-all.sh [--fast | [--baseline <dir>] screenshot-probe args...]
#   e.g. bash scripts/verify-all.sh --only dashboard --themes light
set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

# Lane flags are parsed BEFORE any probe runs (2026-10-23, probe residue):
# a typo'd --baseline used to be discovered after four browser suites had
# already paid their setup. Everything else passes through to
# screenshot-probe untouched.
FAST=0
BASELINE_FLAG=""
while [ "$#" -gt 0 ]; do
	case "$1" in
	--fast)
		FAST=1
		shift
		;;
	--baseline)
		if [ -z "${2:-}" ]; then
			echo "verify-all: --baseline needs a directory argument" >&2
			exit 2
		fi

		BASELINE_FLAG="$2"
		shift 2
		;;
	*)
		break
		;;
	esac
done

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

# Docs-truth gates (plan T12, 2026-10-23): the corpus counts the living docs
# state, every relative link between them, and every backticked test name a
# doc claims. All three are subsecond and all catch a class the browser
# probes cannot see — a doc that lies about its own archive, a citation that
# no longer resolves, or a promised test that was never written.
echo "=== check-docs-counts ==="
bash scripts/check-docs-counts.sh

echo "=== check-doc-links ==="
bash scripts/check-doc-links.sh

echo "=== check-doc-test-claims ==="
bash scripts/check-doc-test-claims.sh

echo "=== verify-ui-contract ==="
node scripts/verify-ui-contract.js

echo "=== verify-style-contract ==="
node scripts/verify-style-contract.js

echo "=== verify-staleness-live ==="
node scripts/verify-staleness-live.js

echo "=== verify-ops-verdict ==="
node scripts/verify-ops-verdict.js

if [ "$FAST" -eq 1 ]; then
	echo "verify-all: contracts passed (--fast: screenshot sweep skipped)"
	exit 0
fi

echo "=== screenshot-probe ==="
# Alert-only pixel diff hook: LENS_DIFF_BASELINE=<dir> compares the fresh
# captures against that baseline (never fails the lane). Practiced corpus:
# ~/lens-baselines/<gen>/ (save one with screenshot-probe --save-baseline).
# --baseline <dir> is the same hook as a lane flag (T16): it wins over the
# env var when both are set, and everything after it still passes through
# to screenshot-probe.
if [ -n "$BASELINE_FLAG" ]; then
	DIFF_BASELINE="$BASELINE_FLAG"
elif [ -n "${LENS_DIFF_BASELINE:-}" ]; then
	DIFF_BASELINE="$LENS_DIFF_BASELINE"
	set -- "$@" --diff "$DIFF_BASELINE"
fi

if [ -n "${DIFF_BASELINE:-}" ] && [ -d "$DIFF_BASELINE" ]; then
	if find internal/web -newer "$DIFF_BASELINE" \( -name '*.templ' -o -name 'app.css' \) 2>/dev/null | grep -q .; then
		echo "verify-all: WARNING — internal/web markup is newer than baseline $DIFF_BASELINE;" >&2
		echo "  the pixel diff may flag drift that is really baseline staleness. Save a fresh" >&2
		echo "  baseline with: node scripts/screenshot-probe.js --save-baseline $DIFF_BASELINE" >&2
	fi
fi

if [ -n "$BASELINE_FLAG" ]; then
	set -- "$@" --diff "$BASELINE_FLAG"
fi
node scripts/screenshot-probe.js "$@"

echo "verify-all: all probes passed"
