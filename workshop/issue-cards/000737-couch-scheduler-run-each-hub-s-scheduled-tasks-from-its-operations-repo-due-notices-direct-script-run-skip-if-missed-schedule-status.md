---
id: 000737
status: open
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
github_issue:
---

# couch: scheduler: run each hub's scheduled tasks from its operations repo (due notices, direct script-run, skip-if-missed, --schedule --status)

## Problem

Recurring operational work has no scheduler: it runs when someone remembers, or on private timers inside pair
(backup freshness, the Drive report). Project couch-scheduling (ops, `teams/devinfra/projects/couch-scheduling/`)
settled the contour with the operator on 10-20; its `design.md` is the reference.
