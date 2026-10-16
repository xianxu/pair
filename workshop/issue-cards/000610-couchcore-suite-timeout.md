---
id: 000610
status: codecomplete
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
github_issue:
severity: high
started: 2026-10-16T03:54:06-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
actual_hours: 0.28
tracker:
    version: 1
    completion:
        token: close-fed3ad17a02a
        repository: github.com/xianxu/pair
        reviewed_head: 39c11589ec33136d49072399466d26efb94e68cb
        evidence_commit: bfe7a8935b87f88d8a0bdfdfb5e59989ac1f4282
---

# couchcore package takes ~10 min, at the go test timeout

## Problem

Couchcore takes nearly Go's default ten-minute per-package test timeout. This blocks otherwise verified pair landings under fleet load, including #591 and #589. In #591's recorded integration suite abe0a5ed883f06ffe58305cb45a29c09, couchcore timed out at 600.245 s; every other recipe/package passed. The automatic main comparison 2064bde22de3d40f5d15091fe41ef443 passed, but couchcore alone took 580.585 s. This is a pair item in ops/test-economy; the hub owns that project.
