# Provenance (2026-10-15): this hash landed via the vendor-hash repair after
# the 05-18 UI-contract session's go.mod/go.sum changes (`nix run
# .#update-vendor-hash` / buildflow nix-hash-fix reproduce it). NEVER paste a
# hash by hand (AGENTS.md Commands), and never run the hash fixer on a dirty
# tree: BuildFlow#55 (filed 2026-10-15) is the wrong-attribute write this
# repo lived through that day (corruption a61ba8c, restore 48a63df) — it
# RECURRED 18:33 via the stale binary 518739c on a docs-dirty tree
# (corruption 62bcca6; restored by hand from the probe log's `got:` value).
"sha256-gHfzvwf+/m1v7TUCJluZJMVY09mRSqy6TGqJkrvDuEA="
