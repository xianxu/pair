---
id: 000758
status: working
created: 2026-10-20
updated: 2026-10-22
estimate_hours: 4.136
github_issue:
started: 2026-10-22T04:59:40-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
---

# Couch-owned editor endpoint registry and lifecycle

## Problem

remote-session (ops project, operator-approved 10-20): the phone talks only to Couch; Couch serves the session views by
proxying to the nvim that has each file open (registered by parley, parley.nvim issue filed alongside).
