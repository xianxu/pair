---
id: 000597
status: codecomplete
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
github_issue:
severity: critical
started: 2026-10-15T20:32:41-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
actual_hours: 1.75
tracker:
    version: 1
    completion:
        token: close-6e6e7c51eba5
        repository: github.com/xianxu/pair
        reviewed_head: 7daa8f2bf37dfd1eaf52831b5b4718d9c4b588b2
        evidence_commit: d51982bd1e753d38f832d12d5bd1512a98ae3723
---

# Reject stale legacy native owners

## Problem

Tonight's main Couch restart parked ariadne:4, then refused its resume because legacy readiness named reused PID 1111, now an unrelated system service born before the wrapper. ariadne:5 also refuses on lsof timeouts for unrelated legacy owners under load (TL receipt 368e95f0).
