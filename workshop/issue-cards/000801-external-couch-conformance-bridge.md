---
id: 000801
status: working
created: 2026-10-22
updated: 2026-10-22
estimate_hours: 3.75
github_issue:
started: 2026-10-22T07:20:32-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
---

# External hold-open Couch conformance bridge

## Problem

parley.nvim#405 needs a real hold-open #758 editor endpoint producer for external Lua conformance. parley.nvim#408 exercised only public adopt-store/direct-store registry commands and did not start a daemon. Pair's cmd/internal/testcouch owns genuine disposable fleets but is a Go testing.TB API, with no external driver entrypoint. Reimplementing its lifetime and teardown in Parley would duplicate authority and risk live-state leakage.
