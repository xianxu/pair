---
id: 000689
status: codecomplete
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
actual_hours: 9.76
tracker:
    version: 1
    completion:
        token: close-228dac1168f8
        repository: github.com/xianxu/pair
        reviewed_head: 6bb343364a02a37e1a7a343ab0b1a4491b3c5dc9
        evidence_commit: 51997b1d9aaa4d3e939c656748c50f3e964199b3
---

# history backup: find and restore from the backups (local or Drive), with a round-trip restore test

## Problem

pair archive can create and upload backups (continuous pass, Drive sink #678) but has no way to read them back: no restore command and no index. The operator asked (2026-10-18) whether we have tested reading the backups back, for example fetching the logs from a few days selectively. We have not: verification today proves the bytes in Drive match the local ciphertext, not that a bundle decrypts, unpacks, and yields the files someone wants. Untested restore means unproven backup.
