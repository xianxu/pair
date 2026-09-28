---
id: '000311'
status: done
started: 2026-09-23T13:07:35-07:00
created: 2026-09-23
updated: 2026-09-23
actual_hours: 0.32
---

# Click right-pane tab to switch tabs

## Problem

The right pane renders a tab strip, but clicking a tab does not switch the
right-pane terminal to that tab. Operators must use the keyboard tab controls
instead, even when the desired tab is visible.

This is the focused successor to the punted `#200`: implement the interaction
without reintroducing a second mouse-mode arbitration path.
