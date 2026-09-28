---
id: '000166'
status: punt
started: 2026-09-01T17:08:04-07:00
created: 2026-09-01
updated: 2026-09-01
---

# Mouse scroll stops after resuming parked Codex agent

## Problem

After parking and resuming a Codex thread through Couch, Zellij can continue to
report a growing scrollback count while mouse and programmatic `scroll-up`
remain at offset zero. A parked/resumed Claude thread did not reproduce the
failure. The exact Codex native session binding was preserved; restoring the
precise pre-park viewport is explicitly out of scope.
