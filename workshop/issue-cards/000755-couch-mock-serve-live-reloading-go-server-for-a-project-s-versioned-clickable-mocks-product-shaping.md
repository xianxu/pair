---
id: 000755
status: codecomplete
created: 2026-10-20
updated: 2026-10-21
estimate_hours: 1.69
github_issue:
started: 2026-10-20T22:40:54-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
actual_hours: 2.36
tracker:
    version: 1
    completion:
        token: close-f2e3dc3705b4
        repository: github.com/xianxu/pair
        reviewed_head: 438872758e60a6f4212cad05b293f677dabae63e
        evidence_commit: 38f02dc4c4c7f5cdd1806a150944261f2d93fabc
---

# couch mock serve: live-reloading Go server for a project's versioned, clickable mocks (product-shaping)

## Problem

Project product-shaping (ops `teams/devinfra/projects/product-shaping/`, operator-approved 10-20) shapes UI projects
through clickable mocks iterated by sketching. It needs one default way to serve a project's mocks.
