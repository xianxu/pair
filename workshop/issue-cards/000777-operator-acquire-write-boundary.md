---
id: 000777
status: codecomplete
created: 2026-10-21
updated: 2026-10-21
estimate_hours: 2.17
github_issue:
started: 2026-10-21T14:41:07-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
actual_hours: 1.05
tracker:
    version: 1
    completion:
        token: close-356fc39cf021
        repository: github.com/xianxu/pair
        reviewed_head: 89cc3757eef3a7631de94f1d9ddb125d60a00c93
        evidence_commit: 3bf1f7d643aeda0af8184100e4f9d492e05b6bc9
---

# session-sync: serialize operator acquisition with the writer commit boundary

## Problem

The operator requested closing the residual InsertEnter race after parley.nvim#404. Auditing cmd/internal/sessionsync/sync.go shows f.guard at write entry, preparation and replaceShared later, and only a post-write sidecar inspection that emits a warning. The source explicitly notes that guard and write cannot be one step against Parley, which does not take the Couch flock. An agent writer that passed the guard can commit after the operator creates the lock.
