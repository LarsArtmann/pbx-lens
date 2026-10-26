<!-- docs/drafts/2026-10-26_tag-ci-first-push-watch-checklist.md
     Plan T3 (2026-10-26): what to watch when the FIRST v* tag push proves
     the tag-CI trigger live. Watching is session-safe; CUTTING a probe tag
     stays owner-gated (17-29 §g1 — the tag namespace and module-proxy path
     are owner territory). gh-gated steps cite their dead-token date:
     nothing below has been watched live as of 2026-10-26. -->

# First `v*` tag push — CI watch checklist

## What should fire

- `.github/workflows/ci.yml` `on.push.tags: ["v*"]` (ci.yml:9, landed
  `cf6e393`, shape parse-verified with a YAML loader the same day). A tag
  push runs the SAME lanes as a main push: build + test(-race) + coverage +
  tidy, golangci-lint, govulncheck, art-dupl zero-group, templ generate
  first — no tag-specific shortcut lane exists.

## Watch steps (owner, once the gh token lives)

1. Confirm the trigger config is on the checked-out commit:
   `git show v0.4.0:.github/workflows/ci.yml | grep -A2 "tags:"` — a tag
   runs the workflow FILE AT THE TAG, so the trigger must predate the tag
   (it does for any tag cut after `cf6e393`).
2. Push the tag, then within a minute:
   `gh run list --workflow ci.yml` — the tag's run must appear with
   `event: push` and the tag name as branch ref.
3. `gh run watch <run-id>` to completion; expect the same green set as the
   HEAD push (a tag shares its commit's tree, so a red tag run on a green
   HEAD commit means a workflow-at-tag drift — investigate `git show
   <tag>:.github/workflows/ci.yml` first, not the lanes).
4. Record the receipt (run id + verdict) in the session report beside the
   `cf6e393` landing row — that closes the 19-53 §a1 "config-true, not
   live-proven" caveat for real.

## Known interactions

- **Double runs:** a tag on a commit that is also main's HEAD used to run
  the workflow twice (push + tags triggers, no shared group). The SHA-keyed
  concurrency group (2026-10-26, ci.yml) cancels the superseded twin — on
  the first tag push, expect ONE surviving run per commit.
- **cut-release.sh does the watching by hand for now:** its tail step says
  "watch the tag's CI run" — the 19-53 §f28 upgrade (a real `gh run watch`
  step in the script) lands after this checklist proves the trigger once.
- **Probe tag:** cutting `v0.0.0-ci-probe` just to watch the trigger is
  STILL owner-gated (§g1) — do not cut it session-side.
