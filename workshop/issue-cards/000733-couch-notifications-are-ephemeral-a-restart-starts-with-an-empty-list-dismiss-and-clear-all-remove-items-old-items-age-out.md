---
id: 000733
status: open
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
github_issue:
started: 2026-10-20T14:54:44-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
---

# couch: notifications are ephemeral: a restart starts with an empty list, dismiss and clear-all remove items, old items age out

## Problem

Couch's "Waiting on you" list keeps every `couch --notify` forever, across restarts: after the 10-20 reboot it showed
11 items going back to 21:03 the night before, all long handled (operator, 10-20, screenshot in ops's session file).
A list that only grows stops being read, which defeats notifications.
