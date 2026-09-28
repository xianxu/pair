---
id: '000142'
status: done
started: 2026-08-16T23:04:16-07:00
created: 2026-08-16
updated: 2026-08-17
estimate_hours: 0.57
actual_hours: 0.42
---

# Codex agent-pane Return should use submit rewrite

## Problem

In the Codex agent pane, plain Return currently submits the composer text. Pair's intended convention is that plain Return acts like `/r` by inserting a newline in the active agent composer, while Alt+Return submits.
