---
id: 000742
status: done
created: 2026-10-20
updated: 2026-10-21
estimate_hours:
github_issue:
started: 2026-10-20T12:53:03-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
actual_hours: 1.54
tracker:
    version: 1
    completion:
        token: close-374449780220
        repository: github.com/xianxu/pair
        reviewed_head: e767dc64d4e161b2539ae87251d276872b9a7bee
        evidence_commit: f91119135c857a50ac87ebada22ae59e61c8c783
        landed_commit: c72583a87f41de17736593c035a7d4f12a6519a4
---

# Couch interrupt refuses a working Codex turn

## Problem

Follow-up to #717 (landed in PR #483): the morning live check reported couch --interrupt refusing a Codex turn that was working. The existing admission proof combines reducer-owned working state, recognized composer, current viewport interrupt hint, and automatic-input ownership. Diagnose which proof is missing from the real Codex path; do not weaken idle/panel/picker or second-interrupt protections to make the working case pass.
