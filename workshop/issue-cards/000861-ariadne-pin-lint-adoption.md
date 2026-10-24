---
id: 000861
status: working
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
---

# Adopt Ariadne shared sidedoc pin and qualify newly active lint rules

## Problem

Pair#838 requires the landed shared sidedoc API from ariadne#575, #576 and #581. Main still pins ariadne 81c832d5; the required pin 5a17f789619d also activates newer golint rules and changes Rule.Match to accept *CheckContext. This blocks #838 pre-close and its joint operator stage.
