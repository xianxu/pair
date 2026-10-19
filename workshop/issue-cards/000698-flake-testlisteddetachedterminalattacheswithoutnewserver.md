---
id: 000698
status: codecomplete
created: 2026-10-19
updated: 2026-10-19
estimate_hours:
github_issue:
flake_tests: [github.com/xianxu/pair/cmd/internal/couchcore.TestListedDetachedTerminalAttachesWithoutNewServer]
started: 2026-10-19T07:48:10-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
actual_hours: 0.11
tracker:
    version: 1
    completion:
        token: close-62f2e72652e5
        repository: github.com/xianxu/pair
        reviewed_head: 50b2d2b03dc9bcb432725bbccd51cd35dd04de7c
        evidence_commit: 84ac0ec1f4c33e09dfc17da12ffcf6cfe5d339f1
---

# flake: TestListedDetachedTerminalAttachesWithoutNewServer

## Problem

Known flaky tests (#394), registered with same-head fail→pass evidence:

- github.com/xianxu/pair/cmd/internal/couchcore.TestListedDetachedTerminalAttachesWithoutNewServer: failed in attempt adbcdbbf22e41d689a246f916f9dd2d2, passed on the same head in attempt b8cfedebf5403d1c880f0aa2a5e89ccd
