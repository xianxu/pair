---
id: 000780
status: open
deps: []
github_issue:
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'e9a6ddde06e386131564b7096d6e99ed98f18d4a' # card fields mirrored from issue-cards; edit via sdlc
---

# flake: TestRecoveryRunningChildAndProducersFinishBeforeHandoff/terminate=false

## Problem

Known flaky tests (#394), registered with same-head fail→pass evidence:

- github.com/xianxu/pair/cmd/internal/termcmd.TestRecoveryRunningChildAndProducersFinishBeforeHandoff/terminate=false: Same exact cf7e78d4af58aa04c5a57929519aa4035650ffa9: retained recorded suite attempt96cb2e3f495bf699eb2371b62bfd3603 failed termcmd TestRecoveryRunningChildAndProducersFinishBeforeHandoff/terminate=false at recovery_test.go387: input producer still running at raw restore. On its unchanged detached checkout /Volumes/TestScratch/local-checks/sdlc-local-2549292918, anchored whole-parent go test -count=2 passed in0.180s (exit0). Suite is continuing; no cancellation or blind whole-suite retry. Separate from prior pre-test ENOSPC.

## Spec

## Done when

- The root cause is fixed (never a retry loop or a longer timeout); closing this issue unregisters the tests.

## Plan

- [ ]

## Log

### 2026-10-21
