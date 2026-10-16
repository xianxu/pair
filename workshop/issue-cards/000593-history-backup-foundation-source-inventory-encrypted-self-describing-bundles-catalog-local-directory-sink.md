---
id: 000593
status: codecomplete
created: 2026-10-15
updated: 2026-10-15
estimate_hours: 7.2025
github_issue:
started: 2026-10-15T20:15:22-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
actual_hours: 2.20
tracker:
    version: 1
    completion:
        token: close-9ae383c468f8
        repository: github.com/xianxu/pair
        reviewed_head: ec92c1fe36f427d3b6accd665a4b7147f79bf3ba
        evidence_commit: ab5aed090e8fcef108a311ee45588e0751161f21
---

# history-backup foundation: source inventory, encrypted self-describing bundles, catalog, local-directory sink

## Problem

Ops project history-backup (approved design: `ops/teams/devinfra/projects/history-backup/design.md`) needs its
foundation: a declared source inventory, a self-describing encrypted bundle format, a catalog, and a first sink.
