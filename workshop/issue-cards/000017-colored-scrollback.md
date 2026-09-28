---
id: '000017'
status: done
created: 2026-05-09
updated: 2026-05-27
actual_hours: N/A
---

# Colored, line-numbered scrollback dump

## Problem

zellij now renders a frame on the agent pane (#08bf61b) so the
top-right scroll-position indicator (e.g. `500/540`) is visible. The
user can see "I'm at line 880 of scrollback" — but has no way to *jump
back* to that line.

zellij has `EditScrollback`, which dumps the pane's scrollback to a
tempfile and opens it in `$SCROLLBACK_EDITOR`. Two problems:

1. **It strips styles.** zellij stores scrollback as a styled cell
   grid internally, but the dump writes plain text — claude's TUI
   palette (cyan commands, dim block-quotes, color-coded diffs) all
   flatten to default fg. Reading colorless TUI output is painful.
2. **It opens in a new tiled pane.** Breaks pair's two-pane invariant
   and disables swap layouts (`exact_panes=2`) for the duration.

We want: from the agent pane (or anywhere), trigger a keybind that
opens the agent's full scrollback — **with colors preserved, line
numbers matching the indicator, in a floating pane** — and lets the
user `:880` to jump.
