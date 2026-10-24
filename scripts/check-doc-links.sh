#!/usr/bin/env bash
# check-doc-links.sh — every RELATIVE link in the living docs must resolve.
#
# The docs cross-reference each other heavily (plans → reports → source), and a
# moved file leaves a 404 that only a human clicking it would notice. This gate
# walks every living Markdown file, skips fenced code blocks, extracts the
# `](target)` links, drops absolute/URLs/anchors, and resolves the rest against
# the linking file's directory.
#
# Scope: living docs only. docs/status/archived/ and docs/planning/archived/
# are CLOSED history — their citations are annotated snapshots, and rewriting
# them to chase a moved file would falsify the record.
#
# Options: --self-test (prove red on a seeded broken link, green on the tree).
set -euo pipefail

root="$(git rev-parse --show-toplevel)"

cd "$root"

# living_docs lists the Markdown files the gate owns.
living_docs() {
	find . -name '*.md' -not -path './docs/status/archived/*' \
		-not -path './docs/planning/archived/*' -not -path './.git/*' \
		-not -path './.crush/*' -not -path './work/*' -not -path './node_modules/*' \
		-not -path './result*' | sort
}

# links_of prints one link target per line, skipping fenced code blocks (where
# `](…)` is usually an example, not a link).
links_of() {
	awk '
		/^[[:space:]]*```/ { fence = !fence; next }
		fence { next }
		{ print }
	' "$1" | grep -oE '\]\([^)]*\)' | sed -E 's/^\]\((.*)\)$/\1/'
}

# resolve_target normalizes one raw target: strips a title, an anchor and a
# query, and returns nothing for links the gate does not own.
resolve_target() {
	local target=$1

	case "$target" in
	"" | \#* | http://* | https://* | mailto:* | tel:* | /*) return ;;
	esac

	# Drop an optional "title" suffix and any anchor/query.
	target=${target%% *}
	target=${target%%#*}
	target=${target%%\?*}

	printf '%s\n' "$target"
}

check_all() {
	local failures=0 checked=0 file dir target resolved
	while IFS= read -r file; do
		dir=$(dirname "$file")
		while IFS= read -r target; do
			resolved=$(resolve_target "$target")
			[ -n "$resolved" ] || continue

			# URL-decode the few escapes that appear in practice.
			resolved=${resolved//%20/ }

			checked=$((checked + 1))
			if [ ! -e "$dir/$resolved" ]; then
				echo "FAIL: $file → $target (resolved: $dir/$resolved)"
				failures=$((failures + 1))
			fi
		done < <(links_of "$file")
	done < <(living_docs)

	echo "doc links: $checked relative link(s) checked, $failures broken"

	return "$failures"
}

self_test() {
	local dir failures=0
	dir=$(mktemp -d)
	trap 'rm -rf "${dir:-}"' EXIT

	if ! check_all >/dev/null 2>&1; then
		echo "FAIL: the tree itself has broken links; fix it before trusting the self-test" >&2
		exit 1
	fi
	echo "PASS: current tree is green"

	# Red: a link to a file that does not exist.
	printf 'see [nowhere](./does-not-exist.md)\n' >"$dir/broken.md"
	if links_of "$dir/broken.md" | grep -q 'does-not-exist'; then
		echo "PASS: link extraction finds the seeded target"
	else
		echo "FAIL: link extraction missed the seeded target" >&2
		failures=$((failures + 1))
	fi

	# Extraction must ignore fenced code blocks and absolute/URL links.
	printf '```\n[example](./nope.md)\n```\n[u](https://example.com) [a](#anchor) [r](/abs)\n' >"$dir/fenced.md"
	local extracted
	extracted=$(links_of "$dir/fenced.md")
	if [ -n "$(printf '%s\n' "$extracted" | sed -n 's/^\.\///p' | grep -v '^http' | grep -v '^#' | grep -v '^/')" ]; then
		echo "FAIL: extraction leaked a fenced/absolute link: $extracted" >&2
		failures=$((failures + 1))
	else
		echo "PASS: fenced, URL and absolute links are skipped"
	fi

	if [ "$failures" -ne 0 ]; then
		echo "check-doc-links self-test: $failures failure(s)" >&2
		exit 1
	fi

	echo "check-doc-links self-test: all checks passed"
}

if [ "${1:-}" = "--self-test" ]; then
	self_test
	exit 0
fi

check_all
