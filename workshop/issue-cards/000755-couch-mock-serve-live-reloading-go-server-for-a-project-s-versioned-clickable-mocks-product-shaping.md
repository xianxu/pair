---
id: 000755
status: codecomplete
created: 2026-10-20
updated: 2026-10-21
estimate_hours: 1.69
github_issue:
started: 2026-10-20T22:40:54-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
actual_hours: 2.20
tracker:
    version: 1
    completion:
        token: close-93e3cf018838
        repository: github.com/xianxu/pair
        reviewed_head: 31384115c5407a4a5f2f3eabe253ff96bff9f651
        evidence_commit: 440071e9f1a8acd3fc98af1b84058d9f13a576ac
---

# couch mock serve: live-reloading Go server for a project's versioned, clickable mocks (product-shaping)

## Problem

Project product-shaping (ops `teams/devinfra/projects/product-shaping/`, operator-approved 10-20) shapes UI projects
through clickable mocks iterated by sketching. It needs one default way to serve a project's mocks.
