---
id: 000730
status: done
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
github_issue:
started: 2026-10-20T10:08:52-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
actual_hours: 0.63
tracker:
    version: 1
    completion:
        token: close-078e2c745edb
        repository: github.com/xianxu/pair
        reviewed_head: e4fd5c1cd2372111d926c0cfddfb4e7faf0b8c83
        evidence_commit: edbf9f62a935a7ec5127663633ac5ea6cd9f88d2
        landed_commit: 4b1f46b03ca7fc8473805055a61d5f0315dd6370
---

# couch: switcher draws hub rows underlined, not reversed, so the selection is the only reverse-video row

## Problem

In Couch's thread switcher, a hub row (ops, ops-echo) is drawn in reverse video, the same treatment as the
selected row, so with two hubs and a selection the operator sees three highlighted rows and can't tell at a glance
which one is selected (operator, 10-20, screenshot in ops's session file).
