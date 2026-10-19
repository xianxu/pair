---
id: 000677
status: done
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
actual_hours: 2.56
tracker:
    version: 1
    completion:
        token: close-4fa30131bf2b
        repository: github.com/xianxu/pair
        reviewed_head: 163c91304af64e5d7d4f35677a870aef8aa95669
        evidence_commit: a04d3bfa741e2c01c2e2a2dd4a18cae60af3ff57
        landed_commit: 477e30cfb056630da756163bf10173c107eff507
---

# diagnostic logs: untie from backup, enforce the per-file cap, keep ~7 days (48 GB, +1.5 GB/day)

## Problem

Pair's diagnostic logs (wrap-events, .pair-diagnostics segments, adapt, fullscreen) are ~48 GB and grow ~1.5 GB a day (survey 2026-10-18, ops qa 2026-10-18-16-history-backup-review.md). Their deletion is blocked by diagnosticlog.KeepDiagnosticsUntilBackup = true ("diagnostic eviction awaits history backup"), and that backup (pair#394) does not exist. Single files reach 4.9 GB despite the 64 MB per-generation cap. The operator agreed debugging logs can be trimmed without a backup.
