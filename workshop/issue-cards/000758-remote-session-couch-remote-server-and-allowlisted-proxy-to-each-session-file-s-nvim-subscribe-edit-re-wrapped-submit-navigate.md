---
id: 000758
status: done
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
actual_hours: 3.65
tracker:
    version: 1
    completion:
        token: close-a23dcba93853
        repository: github.com/xianxu/pair
        reviewed_head: be457f2be2f39f88610d131c03955aef84989843
        evidence_commit: 9d70ae43ed703146f8f993d8293fdc298aa99d27
        landed_commit: 03e736c71e60ba1428e601a0bcb9d94f4c66f2e7
---

# Couch-owned editor endpoint registry and lifecycle

## Problem

remote-session (ops project, operator-approved 10-20): the phone talks only to Couch; Couch serves the session views by
proxying to the nvim that has each file open (registered by parley, parley.nvim issue filed alongside).
