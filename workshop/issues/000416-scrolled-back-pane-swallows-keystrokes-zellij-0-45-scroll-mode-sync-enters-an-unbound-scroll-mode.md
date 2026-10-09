---
id: 000416
status: open
deps: []
github_issue:
created: 2026-10-08
updated: 2026-10-08
estimate_hours:
card_mirror: '35a93ae2e28b5eaaed98c84701d8e6ad77d74012' # card fields mirrored from issue-cards; edit via sdlc
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

- [ ] Add `scroll_mode_sync false` with a why-comment; regenerate the bundle.
- [ ] Regression test beside `TestConfigStatesFullPaneFrames`.
- [ ] Operator smoke test on a rebuilt binary.

## Log

### 2026-10-08

- Diagnosed from zellij v0.45.1 source (route.rs `Action::Write`, screen.rs
  `sync_scroll_mode_on_focus`, input_handler.rs Normal/Locked passthrough).
