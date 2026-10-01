---
id: 000361
status: open
deps: []
github_issue:
created: 2026-09-30
updated: 2026-09-30
estimate_hours:
card_mirror: 'dd862bfa9ac149196c5c23d53a038b0212ffb2de' # card fields mirrored from issue-cards; edit via sdlc
---

# Restore native Zellij conformance fixture launch

## Problem

The couch-zellij-conformance workflow has repeatedly failed before pair#353.
Both TestNativeConsoleWrapperZellij/direct-zellij-baseline and wrapped-zellij
wait 15 seconds for the native fixture and CPR receipt, then time out. Captures
show a default interactive shell rather than the requested fixture; wrapper.log
and receipts are absent. Other native nvim/scrolling/notification tests pass.

## Spec

Diagnose the isolated Zellij fixture launch in
cmd/internal/couchtty/terminal_native_test.go and restore trustworthy native
conformance. Verify command/layout selection and environment handling against
the pinned Zellij 0.45.1 baseline. A plain shell in the capture is evidence of
a launch mismatch, not proof yet of which flag/environment interaction caused
it. Do not weaken the receipt or terminal assertions or simply raise timeouts.

## Done when

- Direct and wrapped native subtests launch their intended fixture and produce
  CPR/input receipts on the CI host.
- The root cause has a focused regression where practical and the full native
  conformance workflow passes, including its later Zellij teardown/continuation
  and shortcut stage.
- Failure diagnostics make fixture-launch failure distinguishable from terminal
  rendering or input-delivery failure.

## Plan

- [ ] Reproduce and identify why Zellij opens a shell instead of the fixture.
- [ ] Correct the launch and verify native conformance locally and in CI.

## Log

### 2026-09-30

Operator requested a separate issue while closing #353. Evidence:
- PR #189 run https://github.com/xianxu/pair/actions/runs/36818804096,
  head db6a3ed04dfd: both subtests time out at terminal_native_test.go:166.
- Prior main run https://github.com/xianxu/pair/actions/runs/36785677892,
  head 50893b121617: identical direct/wrapped failures and missing receipts.
- Earlier main runs 36776313177, 36771991982 and 36767979827 also have failure
  conclusions; their exact failure logs have not yet been inspected.

No CI fix is included in #353; its agent messaging smoke and targeted tests
are separate evidence from this failing native transport workflow.
