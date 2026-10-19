---
id: 000677
status: codecomplete
created: 2026-10-18
updated: 2026-10-18
estimate_hours: 2.52
github_issue:
started: 2026-10-18T16:54:38-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
actual_hours: 2.28
tracker:
    version: 1
    completion:
        token: close-98ad33c0041b
        repository: github.com/xianxu/pair
        reviewed_head: 4729c0d969d5995197d53c9a8063986de5bcf04e
        evidence_commit: c3b41e9fdfe8e1ccef7915ce29586884ba0fb51b
---

# diagnostic logs: untie from backup, enforce the per-file cap, keep ~7 days (48 GB, +1.5 GB/day)

## Problem

Pair's diagnostic logs (wrap-events, .pair-diagnostics segments, adapt, fullscreen) are ~48 GB and grow ~1.5 GB a day (survey 2026-10-18, ops qa 2026-10-18-16-history-backup-review.md). Their deletion is blocked by diagnosticlog.KeepDiagnosticsUntilBackup = true ("diagnostic eviction awaits history backup"), and that backup (pair#394) does not exist. Single files reach 4.9 GB despite the 64 MB per-generation cap. The operator agreed debugging logs can be trimmed without a backup.
