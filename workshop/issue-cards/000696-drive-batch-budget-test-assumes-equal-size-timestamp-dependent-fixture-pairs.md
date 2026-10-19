---
id: 000696
status: done
created: 2026-10-18
updated: 2026-10-19
estimate_hours:
github_issue:
started: 2026-10-19T00:10:41-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
actual_hours: 0.05
tracker:
    version: 1
    completion:
        token: close-50a26bcd914e
        repository: github.com/xianxu/pair
        reviewed_head: 29094fc81c281481a9be87fdeb44acda4a0a75ad
        evidence_commit: acd9d4ba3f40d12221cecc0282a57453ca36186f
        landed_commit: 9ff834e4340e478b7d6d3c8ab2e4e789bd0e8c9f
---

# Drive batch budget test assumes equal-size timestamp-dependent fixture pairs

## Problem

TestDriveMirrorBatchBudgetDefersCompletePairs assumes the second fixture pair fits a byte budget measured only from the first pair. During #689 label compatibility qualification it failed at the second pass with Verified=0, AlreadyRecorded=1, Deferred=1, Pending=1. This is a fixture defect, independent of the new codec: a method-free legacy label serialization equals the new label bytes exactly for both pairs.
