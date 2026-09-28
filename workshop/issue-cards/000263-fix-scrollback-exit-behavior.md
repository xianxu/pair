---
id: 000263
status: open
created: 2026-09-15
updated: 2026-09-15
estimate_hours:
github_issue:
---

# Fix scrollback viewer exit and empty-screen behavior

## Problem

The scrollback viewer opened with `Alt+/` is not reliably usable. In the
observed session it showed an empty screen, and pressing `Esc` printed a raw
escape sequence into the viewer instead of exiting. `Ctrl+C` did exit, but the
reason it worked is currently unknown. This leaves the operator without a
predictable way to inspect or leave scrollback.
