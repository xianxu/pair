---
id: 000861
status: done
created: 2026-10-24
updated: 2026-10-24
estimate_hours: 4.11
github_issue:
started: 2026-10-24T00:02:47-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
actual_hours: N/A
tracker:
    version: 1
    completion:
        token: close-299d80da4037
        repository: github.com/xianxu/pair
        reviewed_head: 2bc8698da1c88a746406f1e0bc010c1efc7ac456
        evidence_commit: 8d3295e5e80cd9eeb1add83cb1b12713dbc0eb21
        landed_commit: 965ba1ee6097dba463e75a3409547bf589904430
---

# Adopt Ariadne shared sidedoc pin and qualify newly active lint rules

## Problem

Pair#838 requires the landed shared sidedoc API from ariadne#575, #576 and #581. Main still pins ariadne 81c832d5; the required pin 5a17f789619d also activates newer golint rules and changes Rule.Match to accept *CheckContext. This blocks #838 pre-close and its joint operator stage.
