---
id: 000611
status: working
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
github_issue:
started: 2026-10-16T06:20:09-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
---

# Bring couchcore under a measured test execution budget

## Problem

Couchcore consumes nearly Go's default ten-minute per-package timeout: #591's integrated run abe0a5ed883f06ffe58305cb45a29c09 timed out at 600.245 s, while its automatic main comparison 2064bde22de3d40f5d15091fe41ef443 passed with couchcore at 580.585 s. #589 is affected too. #610 temporarily gives the broad suite an explicit 20m timeout to unblock landings; it does not solve the accumulated execution cost. This is part of ops/test-economy, whose project is maintained by the hub.
