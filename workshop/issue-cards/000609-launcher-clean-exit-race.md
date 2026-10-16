---
id: 000609
status: open
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
---

# Launcher clean-exit cancellation test races TERM trap readiness

## Problem

The recorded full suite for #589 at head ff788e65bb02408429d7e974389a2511d8492db2 failed TestCancelledHandoffReportsAChildsCleanExit in cmd/internal/launcher/osruntime_launch_test.go:85: handoff = -1, <nil>; want the child's own 0, nil. All other packages passed, including couchcmd, couchmessage and couchcore. Launcher source and tests have no diff from origin/main in #589.

The fixture sends cancellation after a fixed 100 ms without confirming that its shell installed the TERM trap. A delayed shell startup could therefore receive TERM before the trap is ready. This is a hypothesis, not a controlled reproduction. The neighboring TERM-ignoring test already waits for a trap-ready file before cancellation.
