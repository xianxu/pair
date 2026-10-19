---
id: 000702
status: done
created: 2026-10-19
updated: 2026-10-19
estimate_hours:
github_issue:
started: 2026-10-19T04:24:04-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
actual_hours: 0.86
tracker:
    version: 1
    completion:
        token: close-3691a81012fd
        repository: github.com/xianxu/pair
        reviewed_head: ebb6962cd0cbbd7e50572a79409281119b3fcd6c
        evidence_commit: 2338a4b35d68675fa59d7c31b4b103dc7dcdcee1
        landed_commit: b84fb5d9d7a7a9301077fee7af36e1ca470855c1
---

# Compress large database snapshots faster with measured levels and threads

## Problem

#688's coherent 1,999,003,648-byte history snapshot acquired in 7.31s, but the following existing zstd -19 --single-thread plus gpg pipeline exhausted the acceptance pass's five-minute outer budget. Large daily database compression, rather than snapshot acquisition, now dominates latency. The snapshot reader had already exited and its WAL did not grow.
