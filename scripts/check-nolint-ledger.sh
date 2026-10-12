#!/usr/bin/env bash
# check-nolint-ledger — every //nolint: in the Go sources must have a row in
# docs/nolint-ledger.md. Rows are matched by FILE BASENAME (the ledger cites
# file:line, but line numbers are human context that legitimately drifts; the
# file is the stable key — a file's first nolint must mint its row). A nolint
# without a ledger row and without an inline why is a bug — fix one or both.
# Enforcement landed with plan T5.4 (2026-10-12); the rule is stated in the
# ledger header. CI runs this as a named step; run it locally before pushing.
set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

LEDGER="docs/nolint-ledger.md"
fail=0

while IFS=: read -r file lineno; do
	base="$(basename "$file")"
	if ! grep -q "$base" "$LEDGER"; then
		echo "MISSING LEDGER ROW: $file:$lineno"
		fail=1
	fi
done < <(grep -rn "//nolint:" internal/ cmd/ \
	--include="*.go" |
	grep -v "_test.go:" |
	grep -v "_templ.go:" |
	cut -d: -f1-2 |
	sort -u)

if [ "$fail" -ne 0 ]; then
	echo "nolint-ledger check FAILED: add a row to ${LEDGER} or drop the nolint"
	exit 1
fi

echo "nolint-ledger check OK"
