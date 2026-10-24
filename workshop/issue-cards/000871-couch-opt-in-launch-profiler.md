---
id: 000871
status: done
created: 2026-10-24
updated: 2026-10-24
estimate_hours: 0.804
github_issue:
started: 2026-10-24T08:26:21-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
actual_hours: N/A
tracker:
    version: 1
    completion:
        token: close-d064cdbc76a1
        repository: github.com/xianxu/pair
        reviewed_head: fd1276fe74291faedad6bbc13f9effd7721d414d
        evidence_commit: 2840cf46132c2f6d1142f634f5ddacbb56b39b7f
        landed_commit: 2413c4987413ac3678c517be93f2d58273c4016b
---

# Land the opt-in launch-only Couch profiler independently

## Problem

The live Couch CPU investigation (#865) needs an opt-in profile on the operator's next separately approved restart. Land its already implemented diagnostic independently so scaled synthetic controls and optimization do not hold it up. TL ops:0 explicitly requested this extraction; pair:7 owns implementation/landing from main.
