---
id: 000666
status: codecomplete
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
github_issue:
started: 2026-10-18T10:34:48-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
actual_hours: 0.19
tracker:
    version: 1
    completion:
        token: close-061d6ab8a28b
        repository: github.com/xianxu/pair
        reviewed_head: ed4453bdd51193696f695905c38a378ae956ca7a
        evidence_commit: 54b03ac828945439f5b4787c90c06196b2183093
---

# Guard native fixture root handoffs

## Problem

#636 fixed live recovery/park fixtures that accidentally lost disposable native socket and runtime roots: testpair.ApplyEnvironment clears inherited values, and nested helper environments need explicit propagation. The local source guard prevents recurrence only at its named entrypoints. Capture this Pair-specific mistake class in the repository lint path without making the shared catalog depend on Pair helper names.
