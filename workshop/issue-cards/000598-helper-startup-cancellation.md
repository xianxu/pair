---
id: 000598
status: done
created: 2026-10-15
updated: 2026-10-17
estimate_hours:
github_issue:
started: 2026-10-17T09:39:41-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
actual_hours: 0.33
tracker:
    version: 1
    completion:
        token: close-660388e0b336
        repository: github.com/xianxu/pair
        reviewed_head: 88fbac6183549f2aad6ab5e5ad98c61965617f07
        evidence_commit: 09342bb66d1eae5af855ebf802507a61fd1054b6
        landed_commit: 555014c077da941ef567cb8c8274a241dcd6276b
---

# Diagnose helper startup cancellation timeout

## Problem

Recorded #586 integrated suite fails before the review fixtures in LifetimeTests.test_cancellation_during_startup for SIGINT with delay0.02. Helper process does not exit within the existing3s wait. No output retained because the fixture redirects both streams to DEVNULL.
