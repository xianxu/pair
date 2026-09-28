---
id: '000078'
status: done
started: 2026-06-30T15:58:17-07:00
created: 2026-06-26
updated: 2026-06-30
estimate_hours: 3.12
actual_hours: 0.60
---

# pair Go stateful shell glue

## Problem

After the public entrypoint is Go-owned, remaining stateful shell scripts can keep packaging brittle and hide reliability bugs. The biggest candidates are long-running or session-observing scripts, not short native glue.
