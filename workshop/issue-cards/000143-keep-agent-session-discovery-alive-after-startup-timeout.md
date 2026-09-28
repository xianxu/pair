---
id: '000143'
status: done
started: 2026-08-18T22:02:47-07:00
created: 2026-08-18
updated: 2026-08-19
estimate_hours: 1.00
actual_hours: 0.31
---

# Keep agent session discovery alive after startup timeout

## Problem

The asynchronous session watcher gives up after 60 seconds. Agents such as
Codex can create their transcript only after their first interaction, so an
agent left idle for longer than the startup window never gets a persisted
session ID. The context meter then has no transcript to read and the frame omits
context-window usage for the rest of the session.
