---
id: '000070'
status: done
started: 2026-06-25T09:12:47-07:00
created: 2026-06-25
updated: 2026-06-25
actual_hours: N/A
---

# Fix Codex session id capture with stale pidfile

## Problem
In a live Codex pair session, Alt+x shows `session id: <not captured>` even
though Codex exposes a native thread/session id and has a rollout transcript on
disk.
