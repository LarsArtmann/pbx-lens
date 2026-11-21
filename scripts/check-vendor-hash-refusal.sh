#!/usr/bin/env bash
# check-vendor-hash-refusal.sh — pin the update-vendor-hash app's
# entry-refusal on an already-poisoned vendorHash.nix.
#
# Why this exists (2026-11-21, webphone 07-44 report §e-3): the app's
# EXIT trap restores the pre-run state on ANY failure — when the
# committed pin already holds the sentinel placeholder, a failed probe
# run "restores" the poison and the log reads confusingly. The app now
# refuses at entry unless UPDATE_VENDOR_HASH_ALLOW_SENTINEL=1 (the
# deliberate-repair override that skips backup/restore entirely). This
# script proves the refusal still bites: it runs the REAL app binary
# against a fixture git repo whose committed vendorHash.nix holds the
# fake and asserts the refusal message plus a nonzero exit. The
# override arm is deliberately NOT exercised here — it proceeds to a
# real probe build, too heavy for a check; the refusal is the safety
# surface this pins.
set -euo pipefail

root="$(git rev-parse --show-toplevel)"

# The app is not a derivation attr, so `nix build` cannot target it —
# `nix run` realizes the writeShellApplication wrapper and executes it
# with the fixture repo as cwd, propagating its exit status.
app_ref="$root#update-vendor-hash"

dir="$(mktemp -d)"
trap 'rm -rf "${dir:-}"' EXIT

git -C "$dir" init -q
git -C "$dir" config user.email fixture@invalid
git -C "$dir" config user.name fixture
printf '"%s"\n' "sha256-AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=" >"$dir/vendorHash.nix"
printf 'module fixture\n' >"$dir/go.mod"
git -C "$dir" add vendorHash.nix go.mod
git -C "$dir" commit -qm "fixture: poisoned pin"

set +e
out="$(cd "$dir" && nix run "$app_ref" 2>&1)"
rc=$?
set -e

failures=0
if [ "$rc" -eq 0 ]; then
	echo "FAIL: the app accepted an already-poisoned vendorHash.nix (exit 0)" >&2
	failures=$((failures + 1))
elif ! grep -q "sentinel placeholder" <<<"$out"; then
	echo "FAIL: nonzero exit but the refusal message is missing; output:" >&2
	printf '%s\n' "$out" >&2
	failures=$((failures + 1))
else
	echo "PASS: poisoned pin refused at entry with the actionable message"
fi

if [ -e "$dir/vendorHash.nix.bak" ]; then
	echo "FAIL: the refusal path created a .bak (it must not touch anything before refusing)" >&2
	failures=$((failures + 1))
else
	echo "PASS: refusal path left the tree untouched"
fi

if [ "$failures" -ne 0 ]; then
	echo "check-vendor-hash-refusal: $failures failure(s)" >&2
	exit 1
fi
echo "check-vendor-hash-refusal: all checks passed"
