---
id: 000076
status: working
deps: [000074]
github_issue:
created: 2026-06-26
updated: 2026-06-30
estimate_hours: 2.86
started: 2026-06-30T11:58:44-07:00
---

# pair Go helper dispatch

## Problem

Pair already has several Go helpers, but packaging still exposes them as separate binaries in `bin/`. A single-primary-binary architecture should route those helpers through `pair` without copying code or breaking existing callers.

## Spec

Fold the first low-risk Go helper commands behind the dispatcher introduced by #74. This milestone proves the shared-runner pattern without moving live zellij/nvim call sites yet.

- `pair context`
- `pair scrollback-render`

The old binary names must continue to build and work during this milestone. Existing zellij/nvim/script callers should not change yet; `bin/pair-title.sh`, `bin/pair-scrollback-open`, `bin/pair-changelog-open`, and `nvim/scrollback.lua` keep invoking the legacy command names.

Implementation should extract shared run functions from the existing `package main` commands so both the legacy binaries and `pair-go <subcommand>` routes call the same behavior (`ARCH-DRY`). The dispatcher remains the only process-facing result abstraction for in-process helper routes; helper packages should write to injected stdout/stderr and return integer exit codes or errors rather than calling `os.Exit` in reusable code (`ARCH-PURE`).

Out of scope for this milestone: `pair wrap`, `pair slug`, `pair changelog`, `pair continuation`, `pair scribe`, public `pair` entrypoint changes, install alias changes, and zellij/nvim call-site rewrites. Those remain candidates for later issues once the low-risk dispatch pattern is proven.

## Done when

- [ ] Dispatcher can invoke selected existing Go helpers through `pair-go <subcommand>`.
- [ ] Existing helper binary names still build and work.
- [ ] Tests prove dispatch and legacy command paths reach the same behavior for at least one representative helper.
- [ ] No zellij/nvim keybinding breaks.
- [ ] Pair remains usable after merge.

## Plan

- [x] Choose the first helper set based on #73.
- [ ] Extract reusable run functions for `pair-context` and `pair-scrollback-render`.
- [ ] Add dispatcher routes for `context` and `scrollback-render`.
- [ ] Preserve legacy binary names.
- [ ] Run helper-specific and full relevant integration tests.

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.*

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module design=0.60 impl=0.45
item: skill-or-dispatcher design=0.30 impl=0.30
item: smaller-go-module design=0.40 impl=0.25
item: atlas-docs design=0.10 impl=0.05
item: milestone-review design=0.00 impl=0.20
design-buffer: 0.15
total: 2.86
```

## Log

### 2026-06-26

Created from #72. This milestone reduces packaging surface while preserving current command names.

### 2026-06-30

Claimed after #75 landed. Narrowed the first helper dispatch slice to `context` and `scrollback-render`: they are useful enough to prove the dispatcher path, but low-risk enough to avoid long-running PTY, model, git commit/push, or public launcher behavior. Existing zellij/nvim/shell callers stay on legacy binary names for this milestone (`ARCH-DRY`, `ARCH-PURE`, `ARCH-PURPOSE`).
