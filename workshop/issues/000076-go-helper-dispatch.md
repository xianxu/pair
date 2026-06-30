---
id: 000076
status: done
deps: [000074]
github_issue:
created: 2026-06-26
updated: 2026-06-30
estimate_hours: 2.86
started: 2026-06-30T11:58:44-07:00
actual_hours: 0.54
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

- [x] Dispatcher can invoke selected existing Go helpers through `pair-go <subcommand>`.
- [x] Existing helper binary names still build and work.
- [x] Tests prove dispatch and legacy command paths reach the same behavior for at least one representative helper.
- [x] No zellij/nvim keybinding breaks.
- [x] Pair remains usable after merge.

## Plan

- [x] Choose the first helper set based on #73.
- [x] Extract reusable run functions for `pair-context` and `pair-scrollback-render`.
- [x] Add dispatcher routes for `context` and `scrollback-render`.
- [x] Preserve legacy binary names.
- [x] Run helper-specific and full relevant integration tests.

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
- 2026-06-30: closed — go test ./cmd/internal/contextcmd ./cmd/internal/scrollbackcmd ./cmd/pair-context ./cmd/pair-scrollback-render ./cmd/internal/dispatcher ./cmd/pair-go -count=1; go test ./cmd/pair-go -run TestPairGoContextMatchesLegacyPairContext -count=1; make pair-context pair-scrollback-render pair-go; make -B pair-context pair-scrollback-render pair-go; go test ./... -count=1; git diff -- zellij nvim bin/pair bin/pair-dev bin/pair-title.sh bin/pair-scrollback-open bin/pair-changelog-open empty; rg atlas helper dispatch; git diff --check; review verdict: SHIP

Claimed after #75 landed. Narrowed the first helper dispatch slice to `context` and `scrollback-render`: they are useful enough to prove the dispatcher path, but low-risk enough to avoid long-running PTY, model, git commit/push, or public launcher behavior. Existing zellij/nvim/shell callers stay on legacy binary names for this milestone (`ARCH-DRY`, `ARCH-PURE`, `ARCH-PURPOSE`).

Extracted `cmd/internal/contextcmd` and `cmd/internal/scrollbackcmd` runners so legacy binaries and dispatcher routes share implementation (`ARCH-DRY`, `ARCH-PURE`). Added `pair-go context` and `pair-go scrollback-render` dispatcher routes plus a process-level equivalence test proving `pair-go context` matches `pair-context` stdout/stderr/exit code on the same fixture. Updated atlas to record the current helper-dispatch state and unchanged live shell/Lua callers.

Verification before close: `go test ./cmd/internal/contextcmd ./cmd/internal/scrollbackcmd ./cmd/pair-context ./cmd/pair-scrollback-render ./cmd/internal/dispatcher ./cmd/pair-go -count=1`; `go test ./cmd/pair-go -run TestPairGoContextMatchesLegacyPairContext -count=1`; `make pair-context pair-scrollback-render pair-go`; `make -B pair-context pair-scrollback-render pair-go`; `go test ./... -count=1`; `git diff -- zellij nvim bin/pair bin/pair-dev bin/pair-title.sh bin/pair-scrollback-open bin/pair-changelog-open` empty; atlas grep found `pair-go context`, `pair-go scrollback-render`, and helper dispatch; `git diff --check`.
