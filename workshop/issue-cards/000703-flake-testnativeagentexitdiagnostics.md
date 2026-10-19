---
id: 000703
status: open
created: 2026-10-19
updated: 2026-10-19
estimate_hours:
github_issue:
flake_tests: [github.com/xianxu/pair/cmd/internal/couchtty.TestNativeAgentExitDiagnostics]
started: 2026-10-19T06:30:51-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
---

# flake: TestNativeAgentExitDiagnostics

## Problem

Known flaky tests (#394), registered with same-head fail→pass evidence:

- github.com/xianxu/pair/cmd/internal/couchtty.TestNativeAgentExitDiagnostics: Same exact integrated head 48cf4264e56a67f6737db77997e2e3cec9f42468: recorded suite attempt a57d6744a65388be7e757875ca3aa520 failed TestNativeAgentExitDiagnostics/int at native_agent_exit_diagnostics_test.go:117, actor exceeded cleanup bound after 10s in the uninstrumented Python baseline run before Couch instrumentation. Direct go test ./cmd/internal/couchtty -run ^TestNativeAgentExitDiagnostics$ -count=1 -v on that unchanged HEAD then passed all seven subtests in 0.53s, package 0.827s (owned exec session 80249). Test introduced by existing main commit 9cb5fc7ed; no fixture or exit instrumentation changes in #683. Failure log: /Users/xianxu/workspace/pair/.git/sdlc/test-stats/inflight/a57d6744a65388be7e757875ca3aa520/run.log. Root cause of the signal timeout remains unproved; retain the native baseline boundary rather than widening timeout or weakening assertion.
