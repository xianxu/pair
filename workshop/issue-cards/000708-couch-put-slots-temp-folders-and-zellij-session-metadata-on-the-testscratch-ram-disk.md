---
id: 000708
status: working
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
