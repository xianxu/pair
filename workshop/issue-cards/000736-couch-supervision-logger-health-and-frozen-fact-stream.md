---
id: 000736
status: codecomplete
created: 2026-10-20
updated: 2026-10-21
estimate_hours: 1.7225
github_issue:
started: 2026-10-20T20:02:41-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
actual_hours: 2.53
tracker:
    version: 1
    completion:
        token: close-340630ea010f
        repository: github.com/xianxu/pair
        reviewed_head: 51e1cb0d2d2ebba905f6e85fba34c0fbc8eb4812
        evidence_commit: 8dcdfd8ba5d41283f3dff8278884250daf09b8d4
---

# Couch supervision logger health and frozen fact stream

## Problem

During pair#735 investigation, Couch's supervision facts at ~/.local/share/pair/couch/supervision.jsonl were confirmed frozen since 2026-10-19 15:56:04 (26,061,670 bytes). Archived diagnostic segments do exist; there is no replacement per-message readiness-decision log. Existing facts record slot snapshots and completed hub deliveries.

The logger discards its Open error and becomes permanently unavailable for that supervisor lifetime. Write errors increment a dropped counter without exposing the error. Current metadata matches the frozen file. Scratch-copy Open/Write probes pass, both with writer registrations removed and retained, so the historical failure cannot be attributed from disk. Live writer PID/start matches metadata. No live files were mutated during investigation.
