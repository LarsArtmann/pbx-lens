#!/usr/bin/env bash
# replay-report-verify — mechanically re-derive the review reports' claims.
#
# Extracts every ```console fenced block from docs/review/libs/*.md and runs
# its lines one by one. A doc that claims "this command proves X" keeps that
# claim only while the command still runs and still prints what the report
# says — this loop is the mechanical half of that contract (plan A21, born
# from a report carrying a broken `rg -rn` command that no one ever ran).
#
# Lines are SKIP-listed, not run, when they are inherently unreplayable in
# this environment: host-mutating commands (sudo/mv/rm/cp), one-shot upstream
# actions (gh issue/pr create|edit|close), nix builds (the gate owns those),
# and module-cache paths pinned to OTHER versions than the current pin. A
# skipped line is reported as SKIP; a line that runs and exits non-zero is a
# FAIL (and fails the lane). Everything else is PASS.
#
# Usage:
#   bash scripts/replay-report-verify.sh                 # all review reports
#   bash scripts/replay-report-verify.sh --file <md>     # one report
#   bash scripts/replay-report-verify.sh --self-test     # seeded pass+skip+fail proof
set -uo pipefail

cd "$(git rev-parse --show-toplevel)"

# Skip prefixes (exact leading match). Ordered by the failure they prevent:
# host mutation first, then network writes, then heavy builds, then
# version-pinned module-cache reads that only verify a DIFFERENT pin.
skip_prefixes=(
	"sudo "
	"mv "
	"rm "
	"cp "
	"gh issue create"
	"gh issue edit"
	"gh issue close"
	"gh pr create"
	"gh pr edit"
	"gh release"
	"nix "
	"go work"
	"go doc github.com/larsartmann/go-atomic-write"
)

# Lines whose ${...} expansions reference a module-cache version that is NOT
# the current pin fail for boring reasons; the reports replay them through
# the pin-bump ritual instead. Kept narrow: only the GOMODCACHE@version read.
is_pinned_cache_read() {
	[[ "$1" == *"@v"* ]] && [[ "$1" == *"/go/pkg/mod/"* || "$1" == *"GOMODCACHE"* ]]
}

is_skippable() {
	local line="$1"
	local prefix

	for prefix in "${skip_prefixes[@]}"; do
		case "$line" in
			"$prefix"*) return 0 ;;
		esac
	done

	is_pinned_cache_read "$line"
}

# extract_blocks FILE — prints each console block's lines to stdout, blocks
# separated by a lone marker line. Only ```console fences count; ```text and
# plain fences are prose, not receipts.
extract_blocks() {
	local file="$1" in_block=0

	while IFS= read -r line; do
		if [[ $in_block -eq 0 && "$line" == '```console' ]]; then
			in_block=1

			continue
		fi

		if [[ $in_block -eq 1 && "$line" == '```' ]]; then
			in_block=0
			echo "---BLOCK-END---"

			continue
		fi

		if [[ $in_block -eq 1 ]]; then
			echo "$line"
		fi
	done < "$file"
}

replay_file() {
local file="$1" pass=0 skip=0 fail=0 line block_has_content=0 negative=0

	while IFS= read -r line; do
		if [[ "$line" == "---BLOCK-END---" ]]; then
			block_has_content=0

			continue
		fi

		# Blank lines and comment lines structure the block; not claims.
		[[ -z "$line" || "$line" == \#* ]] && continue

		block_has_content=1

		# Negative assertions flip the exit contract: a line whose comment
		# says the ABSENCE is the claim (`# zero hits`, `# expect-fail`,
		# `# gone from the tree`) passes when the command exits non-zero.
		negative=0
		cmd="$line"

		if grep -qiE '#.*\b(zero|absent|expect-fail|gone from)\b' <<<"$line"; then
			negative=1
			cmd="${line%%#*}"
		fi

		if is_skippable "$cmd"; then
			skip=$((skip + 1))
			printf '  SKIP  %s\n' "$line"

			continue
		fi

		# Strip truncation pipes before running: `| head` closes the pipe
		# early, the producer eats SIGPIPE, and pipefail (rightly) fails
		# the pipeline — truncation is display-only here, never part of
		# the claim's exit contract.
		cmd="${cmd%| head*}"
		cmd="${cmd%|head*}"

		# pipefail: a `rg … | wc -l` pipeline must NOT pass just because
		# wc ran — the rg half is the claim (paid 2026-11-12: a replay
		# passed lines grepping a file deleted from the tree because head
		# still exited 0).
		if bash -o pipefail -c "$cmd" >/dev/null 2>&1; then
			if [[ $negative -eq 1 ]]; then
				fail=$((fail + 1))
				printf '  FAIL  %s   # claimed absent, but it MATCHES\n' "$line"
			else
				pass=$((pass + 1))
				printf '  PASS  %s\n' "$line"
			fi
		else
			if [[ $negative -eq 1 ]]; then
				pass=$((pass + 1))
				printf '  PASS  %s   # absence confirmed\n' "$line"
			else
				fail=$((fail + 1))
				printf '  FAIL  %s\n' "$line"
			fi
		fi
	done < <(extract_blocks "$file")

	if [[ $block_has_content -eq 1 ]]; then
		echo "replay: $file ended inside a block (missing closing fence)" >&2

		return 1
	fi

	echo "replay: $file — $pass pass, $skip skip, $fail fail"

	return "$fail"
}

total_fail=0

if [[ "${1:-}" == "--self-test" ]]; then
	# Seeded proof: one block that must PASS, one SKIP-listed line, one
	# line that MUST FAIL — if the loop cannot catch a lying command, it
	# verifies nothing.
	tmp="$(mktemp -d)"
	trap 'rm -rf "$tmp"' EXIT

	cat > "$tmp/seed.md" <<'SEED'
# seed

```console
echo replay-self-test-pass
sudo rm -rf /would-be-host-mutation
false
```
SEED

	out="$(replay_file "$tmp/seed.md")"
	status=$?
	echo "$out"

	if [[ $status -ne 0 ]] && grep -q 'PASS  echo replay-self-test-pass' <<<"$out" \
		&& grep -q 'SKIP  sudo rm -rf' <<<"$out" \
		&& grep -q 'FAIL  false' <<<"$out"; then
		echo "replay: self-test green (pass+skip+fail all classified)"

		exit 0
	fi

	echo "replay: self-test RED — classification is wrong" >&2

	exit 1
fi

if [[ "${1:-}" == "--file" ]]; then
	[[ -z "${2:-}" ]] && {
		echo "replay: --file needs a path" >&2
		exit 2
	}
	replay_file "$2" || total_fail=$((total_fail + 1))
else
	shopt -s nullglob

	for md in docs/review/libs/*.md; do
		replay_file "$md" || total_fail=$((total_fail + 1))
	done
fi

if [[ $total_fail -gt 0 ]]; then
	echo "replay: $total_fail report(s) carry commands that no longer run — fix the doc or the world" >&2

	exit 1
fi

echo "replay: all report commands re-derived (skips are listed, not hidden)"

exit 0
