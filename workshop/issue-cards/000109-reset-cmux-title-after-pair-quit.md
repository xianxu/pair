---
id: '000109'
status: done
started: 2026-07-07T17:48:57-07:00
created: 2026-07-07
updated: 2026-07-07
estimate_hours: 0.25
actual_hours: 0.05
---

# Reset cmux title after pair quit

## Problem

After exiting pair from inside cmux, the cmux workspace title sometimes remains
on the pair session title instead of resetting to the shell cwd. The native
launcher cleanup is supposed to reset the title when this pair owns the cmux
workspace.
