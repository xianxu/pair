---
id: 000737
status: open
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
github_issue:
started: 2026-10-20T17:17:42-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
---

# couch: scheduler: run each hub's scheduled tasks from its operations repo (due notices, direct script-run, skip-if-missed, --schedule --status)

## Problem

Recurring operational work has no scheduler: it runs when someone remembers, or on private timers inside pair
(backup freshness, the Drive report). Project couch-scheduling (ops, `teams/devinfra/projects/couch-scheduling/`)
settled the contour with the operator on 10-20; its `design.md` is the reference.
