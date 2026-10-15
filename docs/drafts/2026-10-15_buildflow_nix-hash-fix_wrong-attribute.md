# BuildFlow filing — nix-hash-fix wrong-attribute write

- **FILED: [BuildFlow#55](https://github.com/LarsArtmann/BuildFlow/issues/55)**
  (2026-10-15, verified OPEN; checker 0 FAIL/0 WARN before posting)

- **Repo:** LarsArtmann/BuildFlow (own repo, plan-solicited: 05-40 M5/F11)
- **Kind:** bug report body, AI-drafted (Crush footer)
- **Cite-at commit:** BuildFlow `d114f28ce`; go-checker-helpers `helpers.go:208`
  (module `../go-checker-helpers` replace in BuildFlow go.mod)
- **Status:** DRAFT — run checker, then `gh issue create --body-file` (never stdin)

---

## Title

`nix-hash-fix` patches the wrong `.nix` attribute when the stale hash text appears in more than one line (live incident: pbx-lens 2026-10-15)

## Body

`## Problem`

Today (2026-10-15 04:06 CEST) `buildflow -s nix-hash-fix --fix` on
`LarsArtmann/pbx-lens` wrote the corrected **Go vendor hash**
(`sha256-eKVzgUYm…`) into the **wrong attribute**: the `hash =` line of an
unrelated `fetchFromGitHub` input (`cqrs-lint`) inside `flake.nix`, instead
of the `vendorHash` the build error was about. The daemon committed the
corruption within minutes (a61ba8c); the correct input hash
(`sha256-BH3IGPq…`) had to be restored by hand (48a63df). Same class of
churn happened on 2026-10-12 (cc52158 wrote `Z7yp53…` into the same input
line, 8f3b9bc reverted it) — the repo has two live occurrences in four
days.

`## Root cause`

The stale-hash search matches by **hash value only, first hit wins** —
`modules/nix-checker/stale_hash.go:53-63` (at `d114f28ce`):

```go
for _, path := range nixFiles {            // lexical filepath.WalkDir order
    for lineIdx, line := range lines {
        if !strings.Contains(line, m.Specified) {   // value-only match
            continue
        }
        f := buildDirectFix(path, lineIdx+1, line, m)
        ...
        return []finding.Finding{f}         // FIRST line containing the value
    }
}
```

Two properties combine into the wrong-attribute write:

1. `filepath.WalkDir` visits `flake.nix` before `vendorHash.nix`
   (checker.go:69, lexical order), so any matching line in `flake.nix`
   wins over the intended `vendorHash` line.
2. The stale hash text genuinely CAN exist on two lines: on pbx-lens the
   10-12 churn had already put vendor-fix hashes into the
   `fetchFromGitHub` line, so the value-only guard found a "valid" target
   there and `replaceFirst` (go-checker-helpers `helpers.go:208`, count-1
   `strings.Replace`) patched it.

The module's own doc comment asserts the invariant that fails —
`modules/nix-checker/fixer.go:13-15`:

> The exact-match guard in BuildStaleHashFindings guarantees each
> BeforeCode uniquely identifies the line to patch.

Uniqueness holds only while the hash text appears on exactly one line.
After one wrong (or partially wrong) fix — or any copy-paste/template
history — it doesn't, and the fixer silently compounds the damage.

`## Secondary: no dirty-tree awareness`

The fixer reads via `fileindex.ContentCacheReadFile` and writes via
`ApplyTextFixes` with no git-state check between detection and write
(nix-checker/fixer.go:26-42). On pbx-lens the tree was dirty and the
auto-commit daemon swept the corrupted `flake.nix` into a commit before
any verification `nix build` could flag it. The existing
`scripts/vendorhash-tripwire.sh` guards a different case (uncommitted
dep-file changes vs a bumped vendorHash.nix), not fixer-vs-worktree races.

`## Suggested fix`

Minimal, in `BuildStaleHashFindings`:

- Collect ALL lines matching the stale value, not just the first.
- If more than one line matches: prefer a line that also names the
  attribute the mismatch is about (`vendorHash`/`cargoHash`/… for Go/Rust
  FODs — the mismatch error already knows which derivation failed), and
  emit a `warning`-severity finding naming the other matching
  file:line instead of silently picking one.
- If attribute-name disambiguation is not possible, FAIL LOUD (no fix
  emitted, "stale hash matches N lines in K files — resolve manually").
  A fix that lands in the wrong attribute costs far more than a skipped
  auto-fix.

This keeps the happy path identical (single match → same behavior today)
and converts the compounding case into a visible refusal.

Verified against: BuildFlow `d114f28ce` (modules/nix-checker), pbx-lens
commits a61ba8c/48a63df (2026-10-15) and cc52158/8f3b9bc (2026-10-12).

💘 Generated with Crush

---

## Post-file checklist

- [ ] checker run (`--kind body-issue --ai-drafted`)
- [ ] filed via `gh issue create --body-file docs/drafts/<file>` (extract fenced body first)
- [ ] landed: `gh issue view <n>`
- [ ] record in docs/upstream-filings.md
