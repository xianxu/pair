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
actual_hours: 0.84
tracker:
    version: 1
    completion:
        token: close-e9ed36ce2a11
        repository: github.com/xianxu/pair
        reviewed_head: 5620eb1d7a9592f06992466f17083a784d97ecb7
        evidence_commit: 13e4d0ef7957bd6cf955eff369597945e443d899
---

# session-sync: serialize operator acquisition with the writer commit boundary

## Problem

The operator requested closing the residual InsertEnter race after parley.nvim#404. Auditing cmd/internal/sessionsync/sync.go shows f.guard at write entry, preparation and replaceShared later, and only a post-write sidecar inspection that emits a warning. The source explicitly notes that guard and write cannot be one step against Parley, which does not take the Couch flock. An agent writer that passed the guard can commit after the operator creates the lock.
