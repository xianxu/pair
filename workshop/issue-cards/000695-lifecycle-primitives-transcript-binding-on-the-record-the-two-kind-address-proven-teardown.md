---
id: 000695
status: codecomplete
created: 2026-10-18
updated: 2026-10-19
estimate_hours: 4.91
github_issue:
started: 2026-10-18T22:57:15-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
actual_hours: 7.36
tracker:
    version: 1
    completion:
        token: close-a5edb9f5beb2
        repository: github.com/xianxu/pair
        reviewed_head: cd3cf1ac2a3d89da9471e2e426f8a57ecf86dc3b
        evidence_commit: 1e62bd8f63fd96ca11911775f7aab7e0df77f7d0
---

# Lifecycle primitives: transcript binding on the record, the two-kind address, proven teardown

## Problem

pair#684 (one lifecycle mechanism for every slot path) was planned with five milestones. Its first two are the shared primitives: the transcript binding stored on the record, the two-kind lifecycle address, and proven teardown. pair#694 (Couch teams) needs them on main before its team-lifecycle milestone. The hub decided (2026-10-18) to split them into their own issue. That keeps one issue per branch, with no landing in the middle of an issue, and lets both #684's remaining milestones and #694's M3 build from main.
