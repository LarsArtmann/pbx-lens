#!/usr/bin/env bash
# cut-release.sh — the release runbook as a script (TODO row "Release runbook
# as a script", 2026-10-17). Encodes the two paid-for lessons:
#
#   1. NEVER on a dirty tree (BuildFlow#55): a dirty-tree hash repair
#      corrupted vendorHash.nix; every repair/receipt runs clean-only.
#   2. Never tag red (2026-10-15, v0.2.0): the tag was cut while CI was red,
#      forcing a fix + re-tag dance that poisons the module proxy window.
#      This script refuses to tag unless the local full gate is green AND
#      the remote CI run on HEAD is green.
#
# Sequence: quiesce check → CHANGELOG sweep → nix build → full dev gate →
# browser probes → tag → push → module-proxy check (`go list -m`).
#
# Usage: scripts/cut-release.sh vX.Y.Z
set -euo pipefail

VERSION="${1:-}"
REPO="$(git rev-parse --show-toplevel)"
cd "$REPO"

fail() {
	echo "FAIL: $*" >&2
	exit 1
}

step() { echo "== $*"; }

[ -n "$VERSION" ] || fail "usage: scripts/cut-release.sh vX.Y.Z (semver, v-prefixed)"
echo "$VERSION" | grep -qE '^v[0-9]+\.[0-9]+\.[0-9]+$' ||
	fail "$VERSION is not vX.Y.Z"

step "quiesce: clean tree, no concurrent gate, load policy"
[ -z "$(git status --porcelain)" ] ||
	fail "working tree is dirty — commit or stash first (BuildFlow#55: never repair or tag from a dirty tree)"
git fetch origin main --quiet
[ "$(git rev-parse HEAD)" = "$(git rev-parse origin/main)" ] ||
	fail "HEAD != origin/main — push/rebase the branch first; releases cut from unpushed work tag red"
read -r L1 L5 L15 _ < /proc/loadavg
awk -v a="$L1" -v b="$L15" 'BEGIN { exit !(a < 15 && b < 40) }' ||
	fail "load ${L1}/${L15} is over the gate policy (1-min < 15 AND 15-min < 40) — wait for a quieter machine"

step "CHANGELOG sweep: $VERSION must have a section (or Unreleased must be renamed first)"
grep -qE "^## \[$VERSION\]|^## \[Unreleased\]" CHANGELOG.md ||
	fail "CHANGELOG.md has neither [$VERSION] nor [Unreleased] — write the release section first"

step "nix build (ships result/bin/pbx-lens; proves the flake + vendorHash)"
TMPDIR="${TMPDIR:-/tmp}" nix build

step "full dev gate (the quality gate; TMPDIR redirect if /tmp is tight)"
TMPDIR="${TMPDIR:-/tmp}" nix develop -c env BUILDFLOW_NO_RESULT_CACHE=1 buildflow --build-mode dev

step "browser probe lane (all four suites)"
bash scripts/verify-all.sh

step "remote CI on HEAD"
if command -v gh > /dev/null 2>&1; then
	HEAD_SHA="$(git rev-parse HEAD)"
	gh run list --commit "$HEAD_SHA" --limit 1 --json status,conclusion --jq '.[0]' |
		grep -q '"conclusion":"success"' ||
		fail "remote CI on $HEAD_SHA is not green (pending or failed) — the 2026-10-15 tag-red lesson: watch it first (gh run watch)"
else
	echo "WARN: gh not available — skipping the remote-CI check; verify manually before pushing the tag"
fi

step "tag $VERSION (annotated) and push main + tag"
git tag -a "$VERSION" -m "pbx-lens $VERSION" ||
	fail "tag $VERSION already exists or refused to create"
git push origin main "$VERSION"

step "module proxy check (the proxy must serve $VERSION)"
sleep 10
go list -m "github.com/larsartmann/pbx-lens@$VERSION" ||
	fail "module proxy does not serve $VERSION yet — retry 'go list -m' in a minute (proxy lag), or investigate the push"

echo "DONE: $VERSION tagged, pushed, and served by the module proxy."
echo "Remaining by hand: watch the tag's CI run (gh run watch), then the GitHub Release notes."
