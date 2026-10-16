---
id: 000584
status: codecomplete
created: 2026-10-15
updated: 2026-10-15
estimate_hours: 1.3935
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
actual_hours: 2.91
tracker:
    version: 1
    completion:
        token: close-3c4135164a47
        repository: github.com/xianxu/pair
        reviewed_head: a7ce2731d8694771996b0f110f69586e04831ccd
        evidence_commit: 850c5c5d19aede124e1101e4308ef1afc63c71ba
---

# Native owner scan blocks fleet resume on unrelated processes

## Problem

#556 native-owner admission blocks Resume from disk fleet-wide on macOS. It requires command evidence for every process before filtering Pair wrappers. Real live root services return EINVAL for kern.procargs2 and are omitted from the command snapshot; zombies and process-table churn also leave metadata without command evidence. Two read-only probes reproduced the exact refusal with 205–207 missing entries. ariadne:5 is parked with a proven native binding but cannot resume.
