---
id: '000075'
status: done
started: 2026-06-29T21:55:46-07:00
created: 2026-06-26
updated: 2026-06-29
estimate_hours: 3.75
actual_hours: 0.98
---

# pair Go launcher prototype

## Problem

The launcher is the largest remaining shell surface and the most important packaging target, but it owns many behavioral edges: session picker, tag normalization, resume/continue/rename, zellij lifecycle, quit/restart markers, data-dir migrations, orphan cleanup, cmux title ownership, and dev rebuild behavior. Porting it must not break normal Pair usage.
