---
id: 000609
status: codecomplete
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
github_issue:
severity: medium
started: 2026-10-16T09:57:08-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
actual_hours: 0.58
tracker:
    version: 1
    completion:
        token: close-0e7f64ba545b
        repository: github.com/xianxu/pair
        reviewed_head: 0d3c84021677116dd163b013db555f7dcf0b40ab
        evidence_commit: 5cd88ba456b0bbb5dad262c14c602ff1abb15f9b
---

# Launcher clean-exit cancellation test races TERM trap readiness

## Problem

The recorded full suite for #589 at head ff788e65bb02408429d7e974389a2511d8492db2 failed TestCancelledHandoffReportsAChildsCleanExit in cmd/internal/launcher/osruntime_launch_test.go:85: handoff = -1, <nil>; want the child's own 0, nil. All other packages passed, including couchcmd, couchmessage and couchcore. Launcher source and tests have no diff from origin/main in #589.

The fixture sends cancellation after a fixed 100 ms without confirming that its shell installed the TERM trap. A delayed shell startup could therefore receive TERM before the trap is ready. This is a hypothesis, not a controlled reproduction. The neighboring TERM-ignoring test already waits for a trap-ready file before cancellation.
