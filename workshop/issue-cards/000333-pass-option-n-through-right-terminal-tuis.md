---
id: '000333'
status: done
started: 2026-09-26T16:51:02-07:00
created: 2026-09-26
updated: 2026-09-26
actual_hours: 2.54
---

# Pass Option+n through to right-terminal TUIs

## Problem

When the right terminal pane is focused on a TUI program, Pair currently
intercepts `Option+n` as its own reload action. This prevents TUIs that bind
that key from receiving it.
