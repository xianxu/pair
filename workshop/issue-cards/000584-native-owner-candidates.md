---
id: 000584
status: open
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
github_issue:
severity: critical
started: 2026-10-15T15:43:33-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
---

# Native owner scan blocks fleet resume on unrelated processes

## Problem

#556 native-owner admission blocks Resume from disk fleet-wide on macOS. It requires command evidence for every process before filtering Pair wrappers. Real live root services return EINVAL for kern.procargs2 and are omitted from the command snapshot; zombies and process-table churn also leave metadata without command evidence. Two read-only probes reproduced the exact refusal with 205–207 missing entries. ariadne:5 is parked with a proven native binding but cannot resume.
