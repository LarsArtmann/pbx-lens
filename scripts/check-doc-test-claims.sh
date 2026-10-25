#!/usr/bin/env bash
# check-doc-test-claims.sh — every Go test a living doc claims must exist.
#
# The 2026-10-25 phantom incident: docs/http-stack.md promised two recording
# timing tests for seven days; neither had ever been written, and the claim
# was discovered only by a failing assertion. This gate extracts backticked
# `Test…` names from every living Markdown file and asserts each names a real
# Go test function in the tree. A doc that claims a test ships the test — or
# the gate goes red naming the file and the name.
#
# Scope: living docs only. docs/status/archived/ and docs/planning/archived/
# are CLOSED history — their citations describe what was true when written,
# and rewriting them to chase a renamed test would falsify the record.
#
# Options: --self-test (prove red on a seeded phantom claim, green on the
# tree, and that fenced code is ignored).
set -euo pipefail

root="$(git rev-parse --show-toplevel)"

cd "$root"

# living_docs lists the Markdown files the gate owns (same scope as
# check-doc-links.sh).
living_docs() {
	find . -name '*.md' -not -path './docs/status/archived/*' \
		-not -path './docs/planning/archived/*' -not -path './.git/*' \
		-not -path './.crush/*' -not -path './work/*' -not -path './node_modules/*' \
		-not -path './result*' | sort
}

# claims_of prints one backticked Test name per line, skipping fenced code
# blocks (scratch snippets there are not claims). The name must start with
# Test and continue with identifier characters — prose like "Testing" alone
# does not match, a backticked `TestFoo_bar` does.
claims_of() {
	awk '
		/^[[:space:]]*```/ { fence = !fence; next }
		fence { next }
		{ print }
	' "$1" | grep -oE '`Test[A-Za-z0-9_]+`' | tr -d '`' | sort -u
}

# test_exists answers whether name is a Go test function in the tree.
test_exists() {
	local name=$1
	grep -qsE "func (${name})\(" --include='*.go' -r . \
		--exclude='*_templ.go' --exclude-dir=.git --exclude-dir=result
}

check_all() {
	local failures=0 checked=0 file name
	while IFS= read -r file; do
		while IFS= read -r name; do
			[ -n "$name" ] || continue

			checked=$((checked + 1))
			if ! test_exists "$name"; then
				echo "FAIL: $file claims \`$name\` but no such Go test exists in the tree"
				failures=$((failures + 1))
			fi
		done < <(claims_of "$file")
	done < <(living_docs)

	echo "doc test claims: $checked name(s) checked, $failures phantom"

	return "$failures"
}

self_test() {
	local dir failures=0
	dir=$(mktemp -d)
	trap 'rm -rf "${dir:-}"' EXIT

	if ! check_all >/dev/null 2>&1; then
		echo "FAIL: the tree itself has phantom claims; fix it before trusting the self-test" >&2
		exit 1
	fi
	echo "PASS: current tree is green"

	# Red: a doc claiming a test that does not exist.
	printf 'pinned by `TestRecordingProxyNoSuchThing` (never written)\n' >"$dir/phantom.md"
	if claims_of "$dir/phantom.md" | grep -q 'TestRecordingProxyNoSuchThing'; then
		echo "PASS: extraction finds the seeded phantom claim"
	else
		echo "FAIL: extraction missed the seeded phantom claim" >&2
		failures=$((failures + 1))
	fi

	if test_exists "$dir/phantom.md" 2>/dev/null; then
		echo "FAIL: test_exists matched a Markdown file" >&2
		failures=$((failures + 1))
	else
		echo "PASS: test_exists only answers on Go sources"
	fi

	# Fenced code blocks are scratch, not claims.
	printf '```\n`TestAlsoNoSuchThing`(scratch)\n```\n' >"$dir/fenced.md"
	if [ -n "$(claims_of "$dir/fenced.md")" ]; then
		echo "FAIL: extraction leaked a fenced claim" >&2
		failures=$((failures + 1))
	else
		echo "PASS: fenced code is ignored"
	fi

	if [ "$failures" -ne 0 ]; then
		echo "check-doc-test-claims self-test: $failures failure(s)" >&2
		exit 1
	fi

	echo "check-doc-test-claims self-test: all checks passed"
}

if [ "${1:-}" = "--self-test" ]; then
	self_test
	exit 0
fi

check_all
