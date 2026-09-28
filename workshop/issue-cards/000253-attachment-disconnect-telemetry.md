---
id: 000253
status: working
started: 2026-09-14T13:56:39-07:00
created: 2026-09-14
updated: 2026-09-14
estimate_hours:
github_issue:
---

# Record attachment disconnect causes

## Problem

Astro lost its Couch attachment helper and Zellij client while its Zellij server
and Claude survived. Existing notices are transient; procutil.WaitCode and the
launcher handoff collapse signal termination into an integer. PTY read errors
are discarded. The available logs cannot identify this incident's trigger.
The operator requests durable evidence for the next disconnect alongside #250.
