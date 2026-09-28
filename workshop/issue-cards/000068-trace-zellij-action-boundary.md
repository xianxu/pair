---
id: '000068'
status: done
started: 2026-06-23T09:31:06-07:00
created: 2026-06-23
updated: 2026-06-26
estimate_hours: 1.5
actual_hours: 0.23
---

# Trace zellij action boundary

## Problem

Codex pair sessions can reproduce a zellij logout where the final visible line
is `Bye from Zellij!` and the zellij server log reports more than 1000
consecutive unknown/empty client messages. The existing logs identify zellij's
disconnect guard but do not show which pair-originated zellij action happened
immediately before the failure.
