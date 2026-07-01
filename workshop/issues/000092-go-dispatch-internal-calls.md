---
id: 000092
status: working
deps: [000090]
github_issue:
created: 2026-07-01
updated: 2026-07-01
estimate_hours:
started: 2026-07-01T09:40:38-07:00
---

# route internal calls through Go dispatcher

Tracking: #91 (native single binary) — step 2 of 5. Depends on #90.

## Problem

The runtime tree still ships several standalone Go helper binaries
(`pair-wrap`, `pair-slug`, `pair-changelog`, `pair-continuation`, `pair-context`,
`pair-scribe`, `pair-scrollback-render`, `pair-session-watch`) that generated
internal call-sites (nvim Lua, shell hooks, `pair-wrap` turn-end) invoke by
name. #76 proved the pattern — `pair-go context` and `pair-go scrollback-render`
dispatch through a shared internal runner while the legacy binary names stay
live — but it stopped there (`pair slug` was explicitly left as a later
candidate). As long as every helper is its own binary that callers hardcode,
the runtime bundle can't shrink toward a single executable.

## Spec

Fold the remaining Pair-owned helper binaries behind the Go dispatcher so the
canonical invocation is `pair <subcommand>` (e.g. `pair wrap`, `pair slug`,
`pair changelog`, `pair continuation`), reusing the existing internal runner
packages rather than duplicating logic (`ARCH-DRY`).

- Keep the standalone `bin/pair-<name>` binaries only as **thin compatibility
  shims** that re-exec `pair <name>` (or are dropped where no external caller
  depends on the name). Do not maintain two divergent implementations.
- Migrate generated internal call-sites (nvim Lua, shell orchestrators,
  `pair-wrap` hooks) to the dispatcher form where the caller is Pair-owned.
- Preserve every helper's current CLI contract (flags, env, stdin/exit codes) so
  callers and tests are unaffected; changes are invocation-path only.
- Scope is dispatch routing + shims **only**. Porting stateful shell
  orchestrators to Go is #93; this issue must not change shell lifecycle
  behavior.

Merge-safe: after this lands, `pair`, `pair-dev`, keybindings, scrollback,
changelog, continuation, and review flows all still work; Pair remains usable.

Architecture: `ARCH-DRY` (one implementation behind dispatch, shims not forks),
`ARCH-PURE` (dispatch/arg parsing stays unit-testable).

## Done when

- [ ] Each remaining Pair-owned helper is invocable as `pair <subcommand>`
      through the dispatcher, reusing its existing internal runner package.
- [ ] Legacy `bin/pair-<name>` binaries are thin shims (re-exec) or removed where
      no external caller needs the name; no duplicated helper logic remains.
- [ ] Generated Pair-owned call-sites use the dispatcher form.
- [ ] Every helper's CLI contract (flags/env/stdin/exit) is unchanged, with
      tests covering the dispatch path and unsupported-subcommand errors.
- [ ] Pair remains usable after merge through the existing `pair` entrypoint.

## Plan

- [ ] Inventory the helper binaries + their internal runner packages and every
      Pair-owned call-site (grep nvim Lua, shell, hooks) that invokes them by name.
- [ ] Route each remaining helper through the dispatcher (`pair <subcommand>`),
      reusing the internal runner.
- [ ] Reduce `bin/pair-<name>` to shims (re-exec) or drop where safe; update the
      runtime manifest.
- [ ] Repoint Pair-owned generated call-sites to the dispatcher form.
- [ ] Tests: dispatch parsing, per-helper contract parity, unsupported-subcommand
      errors; run the shell/nvim integration suites.

## Log

### 2026-07-01

Created as step 2 of the native-single-binary tracker (#91). Continues the
helper-dispatch pattern #76 began (`pair-go context`, `pair-go
scrollback-render`); `pair slug` and the remaining helpers are the concrete
targets.
