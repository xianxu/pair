---
id: '000136'
status: done
started: 2026-08-16T21:01:01-07:00
created: 2026-08-16
updated: 2026-08-16
estimate_hours: 0.4
actual_hours: 0.11
---

# Codex menu selection should accept Return

## Problem

Codex selection/permission menus can require Alt+Return to confirm because Pair
does not recognize the current menu footer as a blocking overlay. When
`pickerActive` is not armed, plain Return follows the Codex textarea remap and
sends LF instead of the bare CR Codex expects for menu confirmation.
