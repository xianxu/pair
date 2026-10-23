---
id: 000833
status: codecomplete
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
github_issue:
started: 2026-10-23T09:26:13-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:9
    worktree: /Users/xianxu/workspace/worktree/pair-slot9/pair
    repository: github.com/xianxu/pair
actual_hours: 0.19
tracker:
    version: 1
    completion:
        token: close-0cc0786d3a40
        repository: github.com/xianxu/pair
        reviewed_head: d219e97a0b48a990f856fbc78698b4b1d630195d
        evidence_commit: c6b75f4c1167ef51a46f29b906b68dc55c2b9b41
---

# Preserve shutdown cancellation cause across presenter requests

## Problem

A console owner shutdown can be reported as a terminal failure instead of cancellation. #739 recorded suite on eafb27680f56c799d06d18e7639b15b6e5d154c4 failed only TestRunConsoleObservesStartupBeforeArmingProducers: hubs_startup_test.go:457 console startup=1, stderr "couch: terminal: terminal: presenter unavailable". Attempt598a61ed6ef5c7be79c26f6cf168a978, failed log /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/598a61ed6ef5c7be79c26f6cf168a978.log, reported SHA prefix4d2073b0e476, duration11m43s. A focused20-run repeat passed before qualification, so no incidence rate is claimed.
