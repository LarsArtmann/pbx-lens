# Upstream filing drafts — staged 2026-11-20 (Pareto plan T22, F60–F62)

Four dependency-hygiene filings staged as drafts; none filed yet. The
fleet policy (how-to-golang) bans these libraries; each arrives in
pbx-lens transitively, so the ask goes to the PARENT module's
maintainers. File one PR/issue per repo, only after re-verifying the
parent's current main (the versions below are the pins this repo sees).

---

## 1. a-h/templ — drop `github.com/fsnotify/fsnotify`

- **Where:** `cmd/templ` watch mode (`templ generate --watch`).
- **Evidence at v0.3.1100:** module graph
  `github.com/a-h/templ@v0.3.1100 → github.com/fsnotify/fsnotify@v1.10.1`.
- **Ask:** evaluate stdlib `os`-based polling or a smaller watcher; the
  fleet policy treats fsnotify's syscall surface as not worth a
  transitive graph slot in a codegen tool.
- **Counter-argument to respect:** watch mode is a developer-only
  feature; fsnotify is the de-facto standard, maintained, and v1.10 is
  a clean module. If upstream declines, this filing dies as won't-fix —
  fine; the ask is cheap.

## 2. ThreeDotsLabs/watermill — drop `github.com/pkg/errors`

- **Where:** watermill core (`github.com/ThreeDotsLabs/watermill@v1.5.3`).
- **Evidence:** `watermill@v1.5.3 → github.com/pkg/errors@v0.9.1`;
  reaches pbx-lens via cqrs-htmx loginpage/usermgmt.
- **Ask:** mechanical replace with stdlib `errors` wrapping
  (`fmt.Errorf` + `%w`); pkg/errors has been in maintenance mode with a
  deprecation notice recommending stdlib since Go 1.13.

## 3. go-appkit (OWNER ACTION, not upstream) — charm v1 → charm.land v2

- **Where:** `go-appkit` v0.12.0 and `go-appkit/systemd` v0.1.0 import
  `github.com/charmbracelet/log@v1.0.0`.
- **Ask-to-self:** the fleet replacement for charm v1 libraries is the
  `charm.land/*/v2` vanity line (how-to-golang policy row). Move both
  modules to `charm.land/log/v2` in their next minor (v0.13 / v0.3),
  then pbx-lens's T23-style bump retires the v1 graph slot.
- **Why staged here:** the dependency leaves pbx-lens's graph only when
  the fleet module moves; the filing ledger records owner-work the same
  way as upstream asks.

## 4. erraudit — `--enforce-*` flags silently no-op under `lint` (REFRESH)

- **Status:** staged draft EXISTS —
  `docs/drafts/2026-10-27_erraudit-lint-enforce-flags-noop.md` (source
  verified against erraudit 61a744c on 2026-10-27).
- **2026-11-20 refresh:** re-verify against erraudit's current main
  before filing; the tool has shipped releases since 61a744c and the
  gap may have closed. If fixed upstream, close the draft as resolved
  and record it in docs/upstream-filings.md.
