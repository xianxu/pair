---
id: 000263
status: open
deps: []
github_issue:
created: 2026-09-15
updated: 2026-09-15
estimate_hours:
---

# Fix scrollback viewer exit and empty-screen behavior

## Problem

The scrollback viewer opened with `Alt+/` is not reliably usable. In the
observed session it showed an empty screen, and pressing `Esc` printed a raw
escape sequence into the viewer instead of exiting. `Ctrl+C` did exit, but the
reason it worked is currently unknown. This leaves the operator without a
predictable way to inspect or leave scrollback.

## Spec

Trace the `Alt+/` opener, rendered input file and Neovim scrollback viewer
startup, then trace `Esc` and `Ctrl+C` through the viewer's mappings and mode
state. Ensure the viewer displays the captured scrollback when data exists,
handles an empty capture explicitly, and exits through a documented keyboard
path without leaking control bytes into the screen. Preserve pending-marker
behavior and the existing detached/in-process opener lifecycle.

## Done when

- `Alt+/` opens a non-empty capture with visible scrollback content.
- Empty or unavailable captures show an actionable state rather than a blank,
  ambiguous viewer.
- `Esc` exits the viewer cleanly (or presents the documented pending-marker
  confirmation) without printing an escape sequence.
- `Ctrl+C` behavior is intentional and covered, whether retained as a fallback
  or replaced by the primary exit path.
- Regression coverage exercises the opener, viewer initialization, and exit
  input at the production boundary.

## Plan

- [ ] Reproduce with a captured non-empty scrollback and inspect the opener's
  rendered/viewport artifacts and Neovim mode/mappings.
- [ ] Trace why `Esc` is rendered literally and why `Ctrl+C` exits; fix the
  viewer initialization or input route at the responsible boundary.
- [ ] Add non-empty, empty-capture, pending-marker and clean-exit regression
  coverage.
- [ ] Run focused scrollback/open tests and an operator smoke check.

## Log

### 2026-09-15

Filed from operator smoke testing: `Alt+/` showed an empty screen; `Esc`
printed its escape sequence instead of exiting; `Ctrl+C` exited successfully,
but its route is unexplained.
