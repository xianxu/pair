---
id: 000619
status: working
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
github_issue:
started: 2026-10-16T18:18:08-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
---

# Couch key help: add Alt+Space, parley's attention finder (parley.nvim#354)

## Problem

parley.nvim#354 (the attention finder, part of ops project universal-agent-input-interface) binds **Alt+u** (M-u) in
parley to open the list of documents waiting on or answered by an agent. Couch's key help
(`cmd/internal/keyhelp/catalog.go`) doesn't list it, so the operator can't discover it from Couch.
