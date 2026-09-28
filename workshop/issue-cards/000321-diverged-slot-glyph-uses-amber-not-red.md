---
id: '000321'
status: done
started: 2026-09-24T19:55:35-07:00
created: 2026-09-24
updated: 2026-09-24
actual_hours: 0.17
---

# Diverged slot glyph uses amber, not red

## Problem

#319 draws the diverged slot glyph `±` in red and the dirty mark `*` in amber.
In live use, the operator wants one attention colour: `±` should use the same
amber as `*`.
