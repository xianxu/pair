---
id: 000708
status: done
created: 2026-10-19
updated: 2026-10-19
estimate_hours: 1.78
github_issue:
started: 2026-10-19T10:47:24-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
actual_hours: 2.94
tracker:
    version: 1
    completion:
        token: close-000331653660
        repository: github.com/xianxu/pair
        reviewed_head: d7593dc70df0e65f516ef7baa1b45e4b1c632514
        evidence_commit: 6612719da4a1eefbcbdf55990f7838a6563c9954
        landed_commit: 537d37028aaae4a6a839d08acb10234e52af63b0
---

# couch: put slots' temp folders and Zellij session metadata on the TestScratch RAM disk

## Problem

The operator's write-only fs_usage sample with the machine quiet (2026-10-19, ops side doc 2026-10-19-06) showed two steady
sources of main-disk file churn outside full suites:
- **Slot temp folders.** Codex and Qoder slots run with the system default TMPDIR (`/var/folders/.../T`, the main SSD), so
  every targeted `go test` they run creates and deletes its temp folders there (~3,700 operations in 30 s). Claude slots use
  `/Volumes/AgentScratch/tmp` (also the SSD).
- **Zellij session metadata.** Every live Zellij session (29 slot panes) rewrites
  `~/Library/Caches/org.Zellij-Contributors.Zellij/contract_version_1/session_info/<session>/session-metadata.kdl` every
  second (~1,840 operations in 30 s), regardless of `session_serialization false`; each write is also journaled by
  fseventsd. The folder also holds 73 session directories for 29 live sessions (stale ones never pruned).

The operator approved a 16 GB RAM disk at `/Volumes/TestScratch` (ops `teams/devinfra/tools/testscratch`, login item;
contents vanish at reboot). ariadne#468 moves sdlc suites, suite checkouts and review checkouts onto it. This issue moves
the slots' own churn. Operator: top priority, for speed first; SSD writes are one bottleneck.
