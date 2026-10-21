---
id: 000744
status: codecomplete
created: 2026-10-20
updated: 2026-10-21
estimate_hours: 8.337
github_issue:
started: 2026-10-20T14:46:17-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
actual_hours: 8.51
tracker:
    version: 1
    completion:
        token: close-0ee462ffa194
        repository: github.com/xianxu/pair
        reviewed_head: f37acbe8b92fe6c1b186b88874e60392722133df
        evidence_commit: d11f13c2142b9391aefaab83d281cd19c2ffd318
---

# testing: one shared disposable-Couch harness for every live test (isolated roots, real supervisor and slots, every agent kind, exact cleanup)

## Problem

Live tests each build their own disposable Couch: pair#722's three-slot drain matrix, pair#726's held-cold resumes,
pair#740's global park, pair#741's hub-resume conformance, and pair#743's conformance in a VM. Each reinvents
isolation (state folder, socket namespace, HOME, queue root) and cleanup, and a mistake in any of them can touch the
live fleet. Operator, 10-20 (ops side doc `2026-10-20-19-conformance-structure.md`, decision 1): one shared harness.
