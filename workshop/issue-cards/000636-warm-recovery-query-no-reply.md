---
id: 000636
status: codecomplete
created: 2026-10-16
updated: 2026-10-18
estimate_hours:
github_issue:
started: 2026-10-17T10:14:59-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
actual_hours: N/A
tracker:
    version: 1
    completion:
        token: close-215034b98246
        repository: github.com/xianxu/pair
        reviewed_head: 823bafe3b328613b8016edd9baebe32be5b23a38
        evidence_commit: 71ee0920cf7f30b74005ec6851ea3e682d3fa22a
---

# Warm recovery live conformance intermittently receives no Zellij query reply

## Problem

Native Zellij 0.45.1 occasionally returns no reply to a recovery fixture query while the exact owned session remains alive. The pre-existing warm live recovery test fails before fixture cleanup. This is separate from #627's daemon/cache cleanup join.
