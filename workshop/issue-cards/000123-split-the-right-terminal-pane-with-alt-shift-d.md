---
id: '000123'
status: done
started: 2026-07-27T17:29:35-07:00
created: 2026-07-27
updated: 2026-07-28
estimate_hours: 1.6
actual_hours: 5.56
---

# Split the right terminal pane with Alt+Shift+d

## Problem

Layout 3 has a Pair-owned right-side terminal area, but it only supports
terminal tabs inside one floating pane. When two live terminal views are needed
side-by-side in the right area, the user has to leave Pair's keybinding model and
manually invoke Zellij splitting. The desired workflow is one terminal shortcut:
`Alt+Shift+d` creates a top/bottom split in the right-side terminal area.
