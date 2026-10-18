---
id: 000662
status: open
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
github_issue:
started: 2026-10-18T03:46:30-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
---

# Review observation diagnostic positive control times out in real BufWritePre resolver

## Problem

The normal make test shell/review phase can fail in ObservationFixtureDiagnosticsTest.test_positive_control_retains_wanted_fresh_and_request_order. The fixture calls a real readiness resolver during BufWritePre; a late synchronous resolver was killed at its existing 2500 ms deadline, producing an empty result and a write refusal. It blocked pair#658’s whole-device after measurement before the final Go recipe.
