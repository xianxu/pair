---
id: 000861
status: codecomplete
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
actual_hours: 1.78
tracker:
    version: 1
    completion:
        token: close-395e36c4dbf0
        repository: github.com/xianxu/pair
        reviewed_head: bddf4ed8812aeaa9f35ba6863f8d0035411f0400
        evidence_commit: 75d47de57803d2f5e13a77dbb3d60f8cede39457
---

# Adopt Ariadne shared sidedoc pin and qualify newly active lint rules

## Problem

Pair#838 requires the landed shared sidedoc API from ariadne#575, #576 and #581. Main still pins ariadne 81c832d5; the required pin 5a17f789619d also activates newer golint rules and changes Rule.Match to accept *CheckContext. This blocks #838 pre-close and its joint operator stage.
