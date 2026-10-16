---
id: 000612
status: done
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
github_issue:
severity: medium
started: 2026-10-16T05:47:20-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
actual_hours: 0.44
tracker:
    version: 1
    completion:
        token: close-342f1fdf5cfc
        repository: github.com/xianxu/pair
        reviewed_head: 7d7e566b2ec852d97ee23759459f544715337be1
        evidence_commit: f90583d54b3cc79bfe32a92d3b29cfe270517e5a
        landed_commit: c756f3cc4429df9dd0ad3004b8fca52162243c0b
---

# Warm reattach acceptance intermittently resolves native binding

## Problem

TestSwitcherWarmReattachWithoutNativeBindingReachesTerminal intermittently fails at cmd/internal/couchcmd/warm_reattach_test.go:142: warm acceptance resolved native binding 1 times (expected zero). The first observed run was #589's integrated-head full suite at 1a2e5f787b4021ff410f83896e2745f14c6b8f0c. It was the only failing test; all other packages passed, including couchcore in 548.732s. Full failure log: /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/a3db77884e3a91eb92e51b263f4d6a8a.log (sha256 prefix 118c2364bb97).

One automatic full-suite main baseline at d59420ed5eee5e3b94113bb8cac0c82cff5d4a59 passed. That single passing comparison was insufficient: the requested repeated whole-package matrix reproduces the same failure on both main and #589's head. The failure therefore does not require #589's changed dormant-readmission fixture.
