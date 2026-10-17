#!/usr/bin/env bash
# AGENTS.md line-budget check (the ≤219 wc -l rule; see
# docs/buildflow-dev-gate.md — the lint tool counts a trailing \n as a
# phantom +1 line, so the file must stay at ≤219 by wc -l).
# Usable standalone, as a pre-commit hook (.githooks/), or in CI.
set -euo pipefail

budget=219
file="${1:-AGENTS.md}"

if [[ ! -f $file ]]; then
  echo "agents-budget: $file not found" >&2
  exit 1
fi

lines=$(wc -l < "$file")
if (( lines > budget )); then
  echo "agents-budget: $file is $lines lines (budget $budget)." >&2
  echo "  Extract, don't append: deep context belongs in docs/, status in docs/status/." >&2
  exit 1
fi

echo "agents-budget: $file at $lines/$budget lines"
