---
id: 000700
status: open
created: 2026-10-19
updated: 2026-10-19
estimate_hours:
github_issue:
started: 2026-10-19T06:12:56-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
---

# Warn when a database backup is older than its freshness threshold

## Problem

#688 found thread_history_1.sqlite had no verified bundle for roughly 55 hours while continuous passes could finish with database skips. A pass-completion marker did not expose individual database freshness.
