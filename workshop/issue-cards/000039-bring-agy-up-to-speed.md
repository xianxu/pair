---
id: '000039'
status: done
created: 2026-06-01
updated: 2026-06-01
estimate_hours: 1
actual_hours: 1
---

# Bring agy agent to full capability parity

## Problem

To fully support the `agy` (Antigravity) TUI agent CLI, we need to ensure all seven integration aspects are validated and active. In a previous iteration, aspects like return remapping, session watchers, recovery flags, and `pair-slug` integration were initially implemented. However, the search of human prompt start (`Alt+b`) glyph registration for `agy` is missing from `nvim/scrollback.lua`, and we need to verify overall capability parity.
