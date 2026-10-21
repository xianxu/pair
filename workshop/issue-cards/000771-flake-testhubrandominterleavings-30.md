---
id: 000771
status: codecomplete
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
github_issue:
flake_tests: [github.com/xianxu/pair/cmd/internal/broadcast.TestHubRandomInterleavings]
started: 2026-10-21T09:04:08-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:9
    worktree: /Users/xianxu/workspace/worktree/pair-slot9/pair
    repository: github.com/xianxu/pair
actual_hours: 0.14
tracker:
    version: 1
    completion:
        token: close-ef84f0b19094
        repository: github.com/xianxu/pair
        reviewed_head: dbceceae886f526872b22548e381e65c942a9bf2
        evidence_commit: d9e85e9e5313533fa8445d02c251b1b29dcffebc
---

# flake: TestHubRandomInterleavings/30

## Problem

Known flaky tests (#394), registered with same-head fail→pass evidence:

- github.com/xianxu/pair/cmd/internal/broadcast.TestHubRandomInterleavings/30: Same exact integrated train234 head d6efc430bb4fbd8b1e943981688eec3914f8c1ca: recorded suite attempt88fb48b8719d5a57732691bd0fe8cb2f raw inflight run.log failed only broadcast.TestHubRandomInterleavings/30 at hub_test.go405 step4, pointer callback0 with armed=true shown=false. On that unchanged detached checkout, exact anchored seed30 count100 passed in1.126s; broader seeds30/130/230/330/430 count100 also passed5.203s. Underlying manual-clock/callback ordering is unproven; register only the evidenced subtest, not sibling seeds. Full suite continues without cancellation; no production source or timeout change.
