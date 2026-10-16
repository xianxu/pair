---
id: 000590
status: done
created: 2026-10-15
updated: 2026-10-16
estimate_hours:
github_issue:
started: 2026-10-16T01:10:48-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
actual_hours: 0.44
tracker:
    version: 1
    completion:
        token: close-45c951815172
        repository: github.com/xianxu/pair
        reviewed_head: a2a2337bced071aa144bc18e25b45dae944d87bb
        evidence_commit: 0ff2f81bcafb6838d8c060195bf9d2400174a4b7
        landed_commit: 2ede2ad1d3cfc1275ad1b8b532a8fae99d9cee1e
---

# Diagnose concurrent launch diagnostics lock timeout under load

## Problem

`TestLaunchFailureStoreReopenConcurrent` intermittently fails under full-suite load when one of 17 concurrent diagnostics Save calls times out acquiring the store lock. The later retention assertion also fails because that failure record was not saved. This is independent of #571's Codex adapter; the diagnostics store and its test are unchanged by that branch.
