---
id: 000661
status: codecomplete
created: 2026-10-18
updated: 2026-10-18
estimate_hours: 7.14
github_issue:
started: 2026-10-18T02:23:24-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
actual_hours: 1.91
tracker:
    version: 1
    completion:
        token: close-c91302b3cb58
        repository: github.com/xianxu/pair
        reviewed_head: 3862431a728b78c31f179ac7078bb6b7cc069b16
        evidence_commit: d1c1ecd299d1ee8f4a6a91fc2e422be26781782e
---

# S4 one edit path: couch --session-sync apply, derived rounds, and the review pane on apply

## Problem

Slice 4 of the universal-agent-input-interface project (design: `ops:teams/devinfra/projects/universal-agent-input-interface/design.side/10-s4-edit-path.md`, sections 3, 5 and 8). Today an agent changes a document in two ways that don't meet: `couch --session-sync write` rewrites a body or section with no record of what changed, and the review pane's nvim applies `{old, occurrence, new, explain}` records itself, dropping the ones that fail. S4 puts every agent edit on one Couch path and keeps each round so parley can show it.
