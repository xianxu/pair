---
id: '000076'
status: done
started: 2026-06-30T11:58:44-07:00
created: 2026-06-26
updated: 2026-06-30
estimate_hours: 2.86
actual_hours: 0.54
---

# pair Go helper dispatch

## Problem

Pair already has several Go helpers, but packaging still exposes them as separate binaries in `bin/`. A single-primary-binary architecture should route those helpers through `pair` without copying code or breaking existing callers.
