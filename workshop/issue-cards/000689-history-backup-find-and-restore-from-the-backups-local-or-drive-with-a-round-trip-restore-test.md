---
id: 000689
status: done
created: 2026-10-18
updated: 2026-10-19
estimate_hours: 4.93
github_issue:
started: 2026-10-18T23:32:01-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
actual_hours: 9.80
tracker:
    version: 1
    completion:
        token: close-7b3e37931718
        repository: github.com/xianxu/pair
        reviewed_head: ca6ecbc6eec6bf94b8bfc7d7426bd7461bde27c0
        evidence_commit: bce9bcd77d4cc2a7983f1f6ef7e472a60945194a
        landed_commit: df16e5f535d3fabc46044958030c8f1e1280f007
---

# history backup: find and restore from the backups (local or Drive), with a round-trip restore test

## Problem

pair archive can create and upload backups (continuous pass, Drive sink #678) but has no way to read them back: no restore command and no index. The operator asked (2026-10-18) whether we have tested reading the backups back, for example fetching the logs from a few days selectively. We have not: verification today proves the bytes in Drive match the local ciphertext, not that a bundle decrypts, unpacks, and yields the files someone wants. Untested restore means unproven backup.
