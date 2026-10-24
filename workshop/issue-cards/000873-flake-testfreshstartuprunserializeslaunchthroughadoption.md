---
id: 000873
status: codecomplete
created: 2026-10-24
updated: 2026-10-24
estimate_hours:
github_issue:
flake_tests: [github.com/xianxu/pair/cmd/internal/couchcmd.TestFreshStartupRunSerializesLaunchThroughAdoption]
started: 2026-10-24T09:26:10-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
actual_hours: N/A
tracker:
    version: 1
    completion:
        token: close-72da6d849d0b
        repository: github.com/xianxu/pair
        reviewed_head: 55c0fe32201270273c9aa5ff360c0b8ba62498e4
        evidence_commit: e242ef74d68d4bcbddd9b14ee80a9e70991f73ea
---

# flake: TestFreshStartupRunSerializesLaunchThroughAdoption

## Problem

Known flaky tests (#394), registered with same-head fail→pass evidence:

- github.com/xianxu/pair/cmd/internal/couchcmd.TestFreshStartupRunSerializesLaunchThroughAdoption: Failed in sdlc merge suites on 2026-10-24: PR 590 head 8747012d (log .git/sdlc/test-stats/failed/da7653e44c27a012b3893c4378e7053c.log) and the main baseline 2b9334fd (log .git/sdlc/test-stats/failed/668931af27a5603e851eae1f4d974e35.log). Both: fresh_startup_run_test.go:199: startup failed: lock thread store: resource temporarily unavailable; no other test failed. sdlc labelled it a persistent main failure. Registered on TL ops:0 instruction so pair merges can proceed; TL gives the root cause to pair:7 (author of the #863 fresh-startup test).
