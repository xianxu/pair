---
id: 000586
status: codecomplete
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
github_issue:
started: 2026-10-15T19:17:48-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
actual_hours: 2.19
tracker:
    version: 1
    completion:
        token: close-a7bfa72666ec
        repository: github.com/xianxu/pair
        reviewed_head: f4818f763f4247261f413b2eb9b0748b056bedc7
        evidence_commit: 472635b1a6730885534c556c4d07968e502dca59
---

# Diagnose intermittent review restoration observation mismatch

## Problem

`tests/review-observation-test.sh` intermittently refuses its repeated branch restoration during a full suite, with insufficient evidence to distinguish a stale observation, a timed-out observation, and an actual branch mismatch. This blocks unrelated #571 verification; the adapter changes none of the review observation/controller/test code.
