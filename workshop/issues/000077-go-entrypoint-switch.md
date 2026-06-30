---
id: 000077
status: working
deps: [000074, 000075, 000076]
github_issue:
created: 2026-06-26
updated: 2026-06-30
estimate_hours: 2.6
started: 2026-06-30T12:42:11-07:00
---

# pair Go entrypoint switch

## Problem

At some point the public `pair` command must become Go-owned. The next safe step is to make the Go-owned `pair-go launch ...` path exercise the real launcher contract while leaving the existing `pair` and `pair-dev` entrypoints stable.

## Spec

Make `pair-go launch ...` a meaningful Go entrypoint by having it hand off to the existing `bin/pair` launcher with `pair`-compatible arguments. `pair-go launch claude` should behave like `pair claude`; `pair-go launch resume <tag>`, `pair-go launch continue ...`, `pair-go launch list`, and `pair-go launch rename ...` should all pass through to the same shell-owned implementation for this migration window.

Keep `pair` and `pair-dev` working exactly as they do today. The Go command has no separate `-dev` variant: a developer shell sourced from `../ariadne/construct/dev-aliases.sh` already discovers `cmd/pair-go`, rebuilds `bin/pair-go` on every invocation, and then runs it from the caller's cwd. If `pair-go launch` cannot find the launcher beside the built binary, its diagnostic should point to `make build` / `make install` and the dev-alias path instead of failing with a bare exec error.

This deliberately keeps real zellij lifecycle, prompt/fzf UI, restart/quit cleanup, continuation, rename/list, and dev rebuild behavior shell-owned until later migration issues. `ARCH-PURPOSE`: #77's purpose is now the first meaningful Go-owned launch entrypoint without destabilizing the existing public command; full public `pair` replacement remains a later cutover once shell stateful glue is reduced.

## Done when

- [ ] `pair-go launch ...` uses Go process code first and then invokes the real launcher with `pair`-compatible argv.
- [ ] Existing `pair` remains the stable public entrypoint for one migration window.
- [ ] `pair-dev` still rebuilds and launches the working tree behavior.
- [ ] Existing create, attach, resume, continue, rename/list, quit, and restart flows are preserved through the `bin/pair` fallback.
- [ ] The dev workflow is documented: `cmd/pair-go` is rebuilt by `../ariadne/construct/dev-aliases.sh`; no `pair-go-dev` command is needed.
- [ ] Pair remains usable after merge; no keybinding workflow regresses.

## Plan

- [ ] Confirm prerequisites from earlier Go migration issues.
- [ ] Add tests for `pair-go launch` argv/env handoff to `bin/pair`.
- [ ] Add stale/missing launcher diagnostics.
- [ ] Implement the thin Go handoff while keeping dispatcher helper routes intact.
- [ ] Verify `pair`, `pair-dev`, and `pair-go launch` behavior with process fakes and targeted builds.
- [ ] Update README/atlas packaging notes.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.25 impl=0.15
item: greenfield-go-module design=0.35 impl=0.40
item: skill-or-dispatcher design=0.30 impl=0.35
item: atlas-docs design=0.20 impl=0.25
item: milestone-review design=0.00 impl=0.20
design-buffer: 0.15
total: 2.62
```

## Log

### 2026-06-26

Created from #72 as the public switch milestone. This should not be claimed until the earlier dispatcher/helper/launcher milestones have landed.

### 2026-06-30

Re-scoped after operator guidance: keep `cmd/pair-go` as the Go entrypoint under test, leave `pair` / `pair-dev` stable, and rely on `../ariadne/construct/dev-aliases.sh` to rebuild `cmd/pair-go` in developer shells. `ARCH-DRY`: reuse the existing launcher for real zellij behavior instead of duplicating shell-owned lifecycle paths in Go. `ARCH-PURE`: keep launch path selection testable with a pure path/argv decision plus a thin exec boundary.

Plan-quality gate returned FAILURE because the plan promised argv/env handoff but did not explicitly test env propagation, and because `pair-dev --help` under-proved the dev rebuild acceptance criterion. Updated the durable plan to require an inherited-env fake-runner assertion and `make test-dev-rebuild` verification. `ARCH-PURPOSE`: compatibility claims must be pinned by tests, not implied by the shell fallback.
