---
id: 000757
status: codecomplete
created: 2026-10-20
updated: 2026-10-21
estimate_hours: 6.657
github_issue:
started: 2026-10-21T02:15:21-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
actual_hours: 5.43
tracker:
    version: 1
    completion:
        token: close-5e1659b51579
        repository: github.com/xianxu/pair
        reviewed_head: 0e4d26c70c40ef990988a2748c9bd219a42549ba
        evidence_commit: e68ece043f221dd9a3c1262dea43a992503a0a83
---

# mock-to-real app template: per-screen real-or-mock dev router, refusing production build, fixtures contract tests (product-shaping)

## Problem

product-shaping (ops) grows the real app out of its mock screen by screen. In development, unbuilt screens should
still be clickable as mocks; nothing half-mocked may ship.
