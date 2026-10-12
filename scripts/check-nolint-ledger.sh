#!/usr/bin/env bash
# check-nolint-ledger — every //nolint: in the Go sources must have a row in
# docs/nolint-ledger.md (matching by FILE and linter name; the ledger's line
# numbers are human context and may drift between the doc and the code).
# A nolint without a ledger row and without an inline why is a bug — fix one
# or both. Enforcement landed with plan T5.4 (2026-10-12); the rule itself
# is stated in the ledger header.
set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

LEDGER="docs/nolint-ledger.md"
fail=0

# Collect nolints from non-test, non-generated Go sources.
while IFS= read -r line; do
  file="${line%%:*}"
  linter="$(sed -n "${line%%:*}p" "$file" | grep -o '//nolint:[a-zA-Z_]*' | head -1 | cut -d: -f2)"
  base="$(basename "$file")"
  if ! grep -q "$base.*|.*${linter}\|\`${base}\`" "$LEDGER" 2>/dev/null && \
     ! grep -q "\`${base}" "$LEDGER"; then
    echo "MISSING LEDGER ROW: $file (${linter})"
    fail=1
  fi
done < <(grep -rn "//nolint:" internal/ cmd/ \
  --include="*.go" \
  | grep -v "_test.go:" \
  | grep -v "_templ.go:" \
  | cut -d: -f1-2 \
  | sort -u)

if [ "$fail" -ne 0 ]; then
  echo "nolint-ledger check FAILED: add a row to ${LEDGER} or drop the nolint"
  exit 1
fi

echo "nolint-ledger check OK"
