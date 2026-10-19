---
id: 000703
status: open
deps: []
github_issue:
created: 2026-10-19
updated: 2026-10-19
estimate_hours:
collaboration-mode: collaboration
card_mirror: '73cd3f4313d810d21331ff404e62dc158736c3be' # card fields mirrored from issue-cards; edit via sdlc
---

# flake: TestNativeAgentExitDiagnostics

## Problem

Known flaky tests (#394), registered with same-head fail→pass evidence:

- github.com/xianxu/pair/cmd/internal/couchtty.TestNativeAgentExitDiagnostics: Same exact integrated head 48cf4264e56a67f6737db77997e2e3cec9f42468: recorded suite attempt a57d6744a65388be7e757875ca3aa520 failed TestNativeAgentExitDiagnostics/int at native_agent_exit_diagnostics_test.go:117, actor exceeded cleanup bound after 10s in the uninstrumented Python baseline run before Couch instrumentation. Direct go test ./cmd/internal/couchtty -run ^TestNativeAgentExitDiagnostics$ -count=1 -v on that unchanged HEAD then passed all seven subtests in 0.53s, package 0.827s (owned exec session 80249). Test introduced by existing main commit 9cb5fc7ed; no fixture or exit instrumentation changes in #683. Failure log: /Users/xianxu/workspace/pair/.git/sdlc/test-stats/inflight/a57d6744a65388be7e757875ca3aa520/run.log. Root cause of the signal timeout remains unproved; retain the native baseline boundary rather than widening timeout or weakening assertion.

## Spec

## Done when

- The root cause is fixed (never a retry loop or a longer timeout); closing this issue unregisters the tests.

## Plan

- [ ]

## Log

### 2026-10-19
