---
id: 000676
status: done
created: 2026-10-18
updated: 2026-10-18
estimate_hours: 1.652
github_issue:
severity: critical
started: 2026-10-18T15:12:17-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
actual_hours: 0.47
tracker:
    version: 1
    completion:
        token: close-4ffa3456b98b
        repository: github.com/xianxu/pair
        reviewed_head: 518a5afda71522b491f82a3308e97db9b07d3195
        evidence_commit: 11d3ef587c0fce6006c7551a148eb963c8b5750e
        landed_commit: e6016fac1fb90e30c2756b24c332ceadc66faa76
---

# Exact Claude resume fails above native tree metadata cap

## Problem

Claude relaunch on the current main build exits before registration with `native resume unavailable (incomplete): native root inspection incomplete`, while Codex relaunch succeeds. The operator cannot restart Claude hubs safely. Both reported requested transcripts still exist; starting fresh abandoned their prior conversations rather than recovering them.
