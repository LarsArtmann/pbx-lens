# Dependabot gomod: unrelated bump deletes sibling-module requires from go.mod

## TL;DR

Bumping ONE Go module can silently delete requires for completely unrelated
multi-module dependencies: our `a-h/templ` PR removed all four
`larsartmann/templ-components*` requires, which breaks the build on merge.
Ask: gomod updates should never prune requires outside the bumped module's
path (or at minimum fail loudly instead of emitting a broken manifest).

## Problem

Repo: https://github.com/LarsArtmann/pbx-lens (public, single Go module at
the root). Dependabot PR
https://github.com/LarsArtmann/pbx-lens/pull/1 — "bump github.com/a-h/templ
from 0.3.1020 to 0.3.1070" — proposes this go.mod diff:

```diff
 	github.com/a-h/templ v0.3.1020
+	github.com/a-h/templ v0.3.1070
-	github.com/larsartmann/templ-components v1.21.0
-	github.com/larsartmann/templ-components/htmx v1.21.0 // indirect
-	github.com/larsartmann/templ-components/icons v1.21.0 // indirect
-	github.com/larsartmann/templ-components/utils v1.21.0 // indirect
```

The bumped module is `a-h/templ`; every deleted require belongs to a
different module path entirely (`larsartmann/templ-components`, a
multi-module repo whose root publishes alongside `/htmx`, `/icons`, `/utils`
submodules). The consumer code imports all four — merging the PR cannot
compile. I bumped `a-h/templ` manually instead
(`40e1a49`, "dependabot mangles this go.mod") and closed the PR.

This looks like the same mechanism as #3617 (closed): the updater re-resolves
the manifest in a sandbox where it cannot see all consuming source, and
`go mod tidy` prunes requires that are alive. #3617 was monorepo-shaped
(nested modules inside ONE repo); this instance shows the pruning also hits
requires of EXTERNAL multi-module dependencies during an unrelated bump, so
the closed issue did not cover the whole class.

## Why it matters

The PR looks routine, CI may even pass on a shallow review diff (go.sum
rewrites dominate), and the broken require set only surfaces at build time
on merge. One wrong PR per multi-module dependency per security group is
enough to make teams pin dependabot off for gomod.

## Minimal repro shape

1. `go.mod` requires `example.com/root v1.0.0` plus its sibling module
   `example.com/root/sub v1.0.0` (separately tagged, both imported by the
   consumer), and an unrelated `example.com/other v1.0.0`.
2. Dependabot bumps `example.com/other`.
3. Observed class: the diff rewrites go.mod/go.sum and the `root/sub`
   (and sometimes `root`) requires disappear.

## Ask

Never remove a require whose module path is outside the bump target's path;
if the re-resolution genuinely cannot keep the manifest consistent, fail the
update instead of proposing one that cannot compile.

Context: seen 2026-10 on dependabot's gomod ecosystem, PR #1 of a fresh
public repo, dependabot config `package-ecosystem: gomod` with a
minor-and-patch group.

💘 Generated with Crush
