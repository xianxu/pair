---
id: 000616
status: codecomplete
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
github_issue:
severity: medium
started: 2026-10-16T11:06:23-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
actual_hours: 0.58
tracker:
    version: 1
    completion:
        token: close-9b85c3bc59eb
        repository: github.com/xianxu/pair
        reviewed_head: bedbf0ba13f30d25b364f1feafcc5173b1970af5
        evidence_commit: 3564f8a46ece0405cf8dfe155360c02d83ffbfba
---

# history-backup: bound database snapshots by compressed bundle size, not source size

## Problem

history-backup's per-file bound (`MaxBundleBytes`, 1 GiB, checked on the source file in `SelectedFiles.Describe`) skips Codex's `thread_history_1.sqlite` (1.5 GB) in every continuous pass (pair#592), so the largest Codex database has no backup. Database snapshots compress well.
