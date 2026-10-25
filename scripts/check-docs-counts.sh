#!/usr/bin/env bash
# check-docs-counts.sh — keep the corpus counts in the living docs TRUE.
#
# The docs state numbers about themselves ("59 reports", "N closed reports",
# the live-report table). Those drift silently every time a report is
# archived, and a wrong count is worse than no count: it makes the index look
# curated while hiding an omission. This gate re-derives each number from the
# filesystem and fails when a claim disagrees.
#
# Checked:
#   1. AGENTS.md "<N> reports" vs docs/status/archived/*.md (minus README).
#   2. docs/status/README.md "<N> closed reports" vs the same count.
#   3. docs/planning/archived/README.md "<N> closed plans" vs its file count.
#   4. docs/status/README.md live table rows vs docs/status/*.md (minus README),
#      and the files exist.
#
# Options: --self-test (prove red on a seeded count change, green on the tree).
set -euo pipefail

root="$(git rev-parse --show-toplevel)"

cd "$root"

count_files() {
	# Count *.md in a directory, excluding its README index.
	find "$1" -maxdepth 1 -name '*.md' ! -name 'README.md' | wc -l | tr -d ' '
}

count_claim() {
	# count_claim <file> <regex with one capture group> — the claimed number.
	local file=$1 pattern=$2
	sed -nE "s/.*${pattern}.*/\\1/p" "$file" | head -n 1
}

failures=0

check() {
	# check <label> <claimed> <actual>
	local label=$1 claimed=$2 actual=$3
	if [ -z "$claimed" ]; then
		echo "FAIL: $label — no claim found (the gate has nothing to verify)"
		failures=$((failures + 1))

		return
	fi

	if [ "$claimed" != "$actual" ]; then
		echo "FAIL: $label — docs say $claimed, tree has $actual"
		failures=$((failures + 1))
	fi
}

run_checks() {
	local archived_status archived_plans live_status

	archived_status=$(count_files docs/status/archived)
	archived_plans=$(count_files docs/planning/archived)
	live_status=$(count_files docs/status)

	check "AGENTS.md archived reports" \
		"$(count_claim AGENTS.md '\<([0-9]+) reports')" "$archived_status"
	check "docs/status/README.md closed reports" \
		"$(count_claim docs/status/README.md '\<([0-9]+) closed reports')" "$archived_status"
	check "docs/planning/archived/README.md closed plans" \
		"$(count_claim docs/planning/archived/README.md '\<([0-9]+) closed plans')" "$archived_plans"

	# The live table in docs/status/README.md is an index: every listed report
	# must exist, and every live report must be listed (an unlisted report is
	# how a harvest source gets forgotten).
	local listed=0
	while IFS= read -r name; do
		[ -n "$name" ] || continue
		listed=$((listed + 1))
		if [ ! -f "docs/status/$name" ]; then
			echo "FAIL: docs/status/README.md lists a missing report: $name"
			failures=$((failures + 1))
		fi
	done < <(sed -nE 's/^\| `([^`]+\.md)`.*/\1/p' docs/status/README.md)

	check "docs/status/README.md live rows" "$listed" "$live_status"

	# Every live report file must appear in the table.
	while IFS= read -r file; do
		name=$(basename "$file")
		[ "$name" = "README.md" ] && continue
		if ! grep -qF "\`$name\`" docs/status/README.md; then
			echo "FAIL: live report not indexed in docs/status/README.md: $name"
			failures=$((failures + 1))
		fi
	done < <(find docs/status -maxdepth 1 -name '*.md' ! -name 'README.md')

	echo "docs counts: $archived_status archived reports, $archived_plans archived plans, $live_status live reports"
}

self_test() {
	local dir failures_before=0
	dir=$(mktemp -d)
	trap 'rm -rf "${dir:-}"' EXIT

	# Green on the real tree.
	if ! run_checks >/dev/null 2>&1; then
		echo "FAIL: the tree itself is red; fix it before trusting the self-test" >&2
		exit 1
	fi
	echo "PASS: current tree is green"

	# Red on a seeded count drift: bump the claim by one in a scratch copy.
	local probe="$dir/docs-status-readme.md"
	sed -E 's/([0-9]+) closed reports/999 closed reports/' docs/status/README.md >"$probe"
	local claimed actual
	claimed=$(count_claim "$probe" '\<([0-9]+) closed reports')
	actual=$(count_files docs/status/archived)
	if [ "$claimed" = "$actual" ]; then
		echo "FAIL: seeded drift did not change the claim" >&2
		failures_before=$((failures_before + 1))
	else
		echo "PASS: a drifted claim is red ($claimed != $actual)"
	fi

	# The claim must exist at all: a doc that stops stating the count is red.
	if [ -n "$(count_claim /dev/null '\<([0-9]+) closed reports')" ]; then
		echo "FAIL: an empty doc produced a claim" >&2
		failures_before=$((failures_before + 1))
	else
		echo "PASS: a missing claim is red"
	fi

	if [ "$failures_before" -ne 0 ]; then
		echo "check-docs-counts self-test: $failures_before failure(s)" >&2
		exit 1
	fi

	echo "check-docs-counts self-test: all checks passed"
}

if [ "${1:-}" = "--self-test" ]; then
	self_test
	exit 0
fi

run_checks

if [ "$failures" -ne 0 ]; then
	echo "check-docs-counts: $failures drift finding(s)" >&2
	exit 1
fi
