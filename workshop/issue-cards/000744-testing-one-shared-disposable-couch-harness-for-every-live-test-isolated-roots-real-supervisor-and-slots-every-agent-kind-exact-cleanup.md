---
id: 000744
status: done
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
actual_hours: 8.89
tracker:
    version: 1
    completion:
        token: close-e5c6ca5f1866
        repository: github.com/xianxu/pair
        reviewed_head: 86e8804dce281ad1d22f89ffc88b7342cbe15dc1
        evidence_commit: 4704a853ad0ce0f0799d32a3d3b91d754d171fe5
        landed_commit: 247339977628c8f8a47678dd666ed86ba0f5780f
---

# testing: one shared disposable-Couch harness for every live test (isolated roots, real supervisor and slots, every agent kind, exact cleanup)

## Problem

Live tests each build their own disposable Couch: pair#722's three-slot drain matrix, pair#726's held-cold resumes,
pair#740's global park, pair#741's hub-resume conformance, and pair#743's conformance in a VM. Each reinvents
isolation (state folder, socket namespace, HOME, queue root) and cleanup, and a mistake in any of them can touch the
live fleet. Operator, 10-20 (ops side doc `2026-10-20-19-conformance-structure.md`, decision 1): one shared harness.
