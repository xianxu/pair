---
id: 000589
status: done
created: 2026-10-15
updated: 2026-10-16
estimate_hours:
github_issue:
severity: medium
started: 2026-10-15T20:45:41-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
actual_hours: 4.94
tracker:
    version: 1
    completion:
        token: close-bc45927f2fef
        repository: github.com/xianxu/pair
        reviewed_head: 8a00404d7ae9a70daba1485be3b595fb6661f7f8
        evidence_commit: d0d0f817a8c176a2179d87427905dc02aa17dabd
        landed_commit: 5d40e6bdd0d60e0070c1afe3c1cf44c92908cb5f
---

# Dormant message readmission test races its last asynchronous attempt

## Problem

The recorded pair#584 full suite at code head 9979945fb failed TestMessageSendToDormantSlotReadmits in cmd/internal/couchcmd/message_service_test.go:449. The test expected Code=unavailable but received Code=accepted after restoring launch evidence. Both the test and messaging service are unchanged from origin/main in #584. The previous full suite passed couchcmd; an isolated 30-repeat rerun passed in 5.061 seconds.

The fixture fires the counted retry callbacks, sleeps 100 ms, then makes launch evidence healthy and immediately sends. That elapsed sleep does not prove the last asynchronous admission result has reached Dormant. A late attempt may validly admit after launch becomes healthy. This is the suspected race, not yet a deterministic baseline reproduction.
