#!/usr/bin/env bash
# check-js-lint.sh — oxlint (no-undef + no-unused-vars) over the probe scripts.
#
# Why this exists: the 2026-10-23 probe-bootstrap refactor shipped a
# ReferenceError (`bin is not defined`) into a shared module, and nothing
# caught it until a browser run — dprint/oxfmt format JS but never check
# references. This gate is the missing half: it reads the same files and
# fails when a name is not defined. The rule set lives in .oxlintrc.json
# (browser + node globals, because probe callbacks run in the page).
#
# Tool philosophy: like the statix tripwire, a missing tool WARNS and skips —
# the lane must not become unusable on a machine without oxlint. Run
# `--self-test` to prove the gate still catches the class it exists for.
set -euo pipefail

root="$(git rev-parse --show-toplevel)"

if ! command -v oxlint >/dev/null 2>&1; then
	echo "check-js-lint: oxlint not on PATH; skipping (install oxlint to arm this gate)" >&2
	exit 0
fi

self_test() {
	local dir failures=0
	dir=$(mktemp -d)
	trap 'rm -rf "${dir:-}"' EXIT

	printf 'console.log(bin);\n' >"$dir/undefined.js"
	if (cd "$root" && oxlint "$dir/undefined.js" >/dev/null 2>&1); then
		echo "FAIL: an undefined reference slipped through no-undef" >&2
		failures=$((failures + 1))
	else
		echo "PASS: undefined reference is red"
	fi

	printf 'console.log("ok");\n' >"$dir/defined.js"
	if ! (cd "$root" && oxlint "$dir/defined.js" >/dev/null 2>&1); then
		echo "FAIL: a clean file was flagged" >&2
		failures=$((failures + 1))
	else
		echo "PASS: clean file is green"
	fi

	if [ "$failures" -ne 0 ]; then
		echo "check-js-lint self-test: $failures failure(s)" >&2
		exit 1
	fi

	echo "check-js-lint self-test: all checks passed"
}

if [ "${1:-}" = "--self-test" ]; then
	self_test
	exit 0
fi

oxlint "$root/scripts"
