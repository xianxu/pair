---
id: 000416
status: done
deps: []
github_issue:
created: 2026-10-08
updated: 2026-10-08
estimate_hours:
card_mirror: 'eab10e0db351aeafba2876c67a59f240015d4867' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-08T17:35:52-07:00
claimant:
    operator: T
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "e03aca96", done: "087d4249"}
actual_hours: 0.27
---

# Scrolled-back pane swallows keystrokes: zellij 0.45 scroll_mode_sync enters an unbound Scroll mode

## Problem

Scroll the right terminal (or the agent pane) back with the wheel, then type:
every keystroke is dropped. The expected behavior, which every terminal has, is
that a keystroke snaps the view back to the bottom and reaches the program.

## Spec

Root cause, read from zellij v0.45.1 source:

- `scroll_mode_sync` (new in 0.45, default **true**): when the focused pane
  becomes scrolled, `Screen::sync_scroll_mode_on_focus` switches the client into
  `InputMode::Scroll`. This covers wheel scrolls and `pair term`'s
  `zellij action scroll-up`.
- `zellij-client/src/input_handler.rs:200`: an unbound key reaches the pane only
  in `Normal` or `Locked` mode.
- pair's `keybinds clear-defaults=true` leaves Scroll mode with no bindings, so
  every key is dropped, and nothing leaves the mode.

Fix: set `scroll_mode_sync false` in `zellij/config.kdl` (and its runtime-bundle
mirror). The client stays in Normal mode, and zellij's `Action::Write` sends
`ClearScroll` before `WriteCharacter` (`route.rs`), which snaps the pane to the
bottom. 0.44.x ignores unknown option names, as it does `pane_frame_style`.

## Done when

- `zellij/config.kdl` and the bundle mirror set `scroll_mode_sync false`, pinned
  by a test that ignores commented mentions.
- Operator smoke: scroll the right terminal back, type a key, and the view snaps
  to the bottom with the key delivered.

## Plan

- [x] Add `scroll_mode_sync false` with a why-comment; regenerate the bundle.
- [x] Regression test beside `TestConfigStatesFullPaneFrames`.
- [ ] Operator smoke test on a rebuilt binary.

## Log

### 2026-10-08
- 2026-10-08: closed — TestConfigDisablesScrollModeSync fails on stale bundle mirror, passes after regen; zellij 0.45.1 setup --check: well defined. Root cause read from zellij v0.45.1 source (scroll_mode_sync -> Scroll mode drops unbound keys; Write path ClearScrolls). Full make -k test (scrubbed env): only test-review fails, identically on origin/main; test-changelog green w/ scratchpad TMPDIR; unsandboxed go test ./...: 4 failures also on main, couchcmd ColdResume fails in this slot with config reverted (environmental). --no-atlas: config-only bugfix, no new surface. --no-plan-check: remaining item is the operator live smoke test, which needs pair:0 rebuilt after merge.; review verdict: SHIP

- Diagnosed from zellij v0.45.1 source (route.rs `Action::Write`, screen.rs
  `sync_scroll_mode_on_focus`, input_handler.rs Normal/Locked passthrough).
- Fix: `scroll_mode_sync false` (1a63116c). `TestConfigDisablesScrollModeSync`
  failed on the stale bundle mirror, then passed after regeneration;
  `zellij setup --check` (0.45.1) reports the config well defined.
- Full suite: `make -k test` with a scrubbed env fails only `test-review`, which
  fails identically on origin/main. `test-changelog` passes with a scratchpad
  TMPDIR. Unsandboxed `go test ./...` fails 5 tests: artifactpath,
  couchsingleton x2 and gcruntime fail on main too, and couchcmd
  `TestColdResumeOfAParkedPrimaryRegistersFromBothOrigins` fails in this slot
  even with config.kdl reverted to main (it passes in a fresh main worktree),
  so it is environmental, not this change.
- Operator smoke test pending: rebuild pair:0 and start a fresh session.
