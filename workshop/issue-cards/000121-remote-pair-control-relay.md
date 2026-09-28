---
id: 000121
status: working
started: 2026-07-26T11:20:10-07:00
created: 2026-07-26
updated: 2026-07-26
estimate_hours: 24.52
github_issue:
---

# Remote Pair control relay

## Problem

Pair sessions are useful from a tablet when the local workstation is already
running the coding environment, but Pair currently requires local terminal access
for session discovery and life cycle gestures. The immediate pain is context
refresh: when a Claude session is near the context limit, the user wants a simple
remote web control to see the active Pair session and trigger the existing
continuation/restart flow without being at the keyboard.

The first slice should prove remote control without widening the trust boundary:
the local machine remains authoritative for repos, files, Zellij, Pair sidecars,
and life cycle commands. A small Oracle-hosted service is only a dumb relay that
lets a browser reach the local Pair daemon through an outbound connection.
