---
id: 000697
status: working
created: 2026-10-19
updated: 2026-10-19
estimate_hours:
github_issue:
flake_tests: [github.com/xianxu/pair/cmd/internal/couchcmd.TestHubStartupLaterFailureLeavesFirstUsable]
started: 2026-10-19T06:54:40-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
---

# flake: TestHubStartupLaterFailureLeavesFirstUsable

## Problem

Known flaky tests (#394), registered with same-head fail→pass evidence:

- github.com/xianxu/pair/cmd/internal/couchcmd.TestHubStartupLaterFailureLeavesFirstUsable: failed in attempt 4fed2c456da3773658b2bd4328454d6a, passed on the same head in attempt 5bafd7c76410f108c1e6ee72e3e78952
