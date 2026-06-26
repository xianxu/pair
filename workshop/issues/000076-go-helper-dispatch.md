---
id: 000076
status: open
deps: [000074]
github_issue:
created: 2026-06-26
updated: 2026-06-26
estimate_hours:
---

# pair Go helper dispatch

## Problem

Pair already has several Go helpers, but packaging still exposes them as separate binaries in `bin/`. A single-primary-binary architecture should route those helpers through `pair` without copying code or breaking existing callers.

## Spec

Fold existing Go helper commands behind the dispatcher introduced by #74. Candidate dispatch modes:

- `pair wrap`
- `pair slug`
- `pair context`
- `pair scrollback-render`
- `pair changelog`
- `pair continuation`
- `pair scribe`

The old binary names must continue to work during this milestone, either as built aliases, tiny shims, or unchanged standalone binaries. Existing zellij/nvim/script callers should not need to change yet unless #73 identifies a safe, tested call site.

Implementation should reuse existing command packages or extract shared run functions from `package main` without duplicating behavior (`ARCH-DRY`).

## Done when

- [ ] Dispatcher can invoke selected existing Go helpers through `pair <subcommand>`.
- [ ] Existing helper binary names still build and work.
- [ ] Tests prove dispatch and legacy command paths reach the same behavior for at least one representative helper.
- [ ] No zellij/nvim keybinding breaks.
- [ ] Pair remains usable after merge.

## Plan

- [ ] Choose the first helper set based on #73.
- [ ] Extract reusable run functions where needed.
- [ ] Add dispatcher routes.
- [ ] Preserve legacy binary names.
- [ ] Run helper-specific and full relevant integration tests.

## Log

### 2026-06-26

Created from #72. This milestone reduces packaging surface while preserving current command names.
