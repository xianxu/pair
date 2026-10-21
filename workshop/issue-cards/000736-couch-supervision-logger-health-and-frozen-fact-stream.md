---
id: 000736
status: working
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
github_issue:
started: 2026-10-20T20:02:41-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
---

# Couch supervision logger health and frozen fact stream

## Problem

During pair#735 investigation, Couch's supervision facts at ~/.local/share/pair/couch/supervision.jsonl were confirmed frozen since 2026-10-19 15:56:04 (26,061,670 bytes). Archived diagnostic segments do exist; there is no replacement per-message readiness-decision log. Existing facts record slot snapshots and completed hub deliveries.

The logger discards its Open error and becomes permanently unavailable for that supervisor lifetime. Write errors increment a dropped counter without exposing the error. Current metadata matches the frozen file. Scratch-copy Open/Write probes pass, both with writer registrations removed and retained, so the historical failure cannot be attributed from disk. Live writer PID/start matches metadata. No live files were mutated during investigation.
