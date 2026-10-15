#!/usr/bin/env bash
# golden-pretty-diff — a readable review of golden-page changes (plan M4/F12).
#
# Goldens are full rendered HTML pages; a plain `git diff` drowns one
# attribute change in hundreds of context lines, and during the UI audit the
# unreadable diff was a real review cost. This wraps `git diff --word-diff`
# over internal/web/testdata/ only, with a grep filter that keeps the
# changed words visible and drops the noise.
#
# Usage: bash scripts/golden-pretty-diff.sh [<commit-ish>]   (default: HEAD)
set -euo pipefail

REF="${1:-HEAD}"

cd "$(git rev-parse --show-toplevel)"

if git diff --quiet "$REF" -- internal/web/testdata/ 2>/dev/null; then
	echo "golden pages unchanged vs $REF"
	exit 0
fi

git diff --word-diff=plain --unified=0 "$REF" -- internal/web/testdata/ |
	grep -v '^index ' | grep -v '^--- ' | grep -v '^+++ '
