---
id: 000764
status: working
deps: []
github_issue:
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
collaboration-mode: collaboration
card_mirror: '7eaf58f747d972eb006d96da963ab7e99eb98b7f' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-21T03:15:27-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: operator, spec: "813f0cf7", done: "34a6f408"}
---

# Controlled Zellij fixture closes its PTY twice and hides cleanup errors

## Problem

ControlledZellij.KillClient closes its PTY but retains the non-nil pointer. ControlledZellij.Close then joins a second Close error, even after exact server cleanup succeeds. TestStartupManagedAttachDetachTimingLive discards that cleanup error. A #761 scratch acceptance overlay surfaced close /dev/ptmx: file already closed after successful attach/detach and reattach/detach; exact server generation was absent and private roots removed.

## Spec

Make the helper lifecycle own and retire its client/PTY handles exactly once. Preserve real shutdown and observation errors; callers must not discard meaningful cleanup failures. This is fixture cleanup, independent of #761 native lookup recovery.

## Done when

A focused regression executes KillClient then Close and observes successful exact cleanup without a spurious closed-PTY error. A genuine shutdown/observation failure still surfaces. Audit callers that discard Close errors and keep native cleanup proof explicit.

## Plan

- [ ] Reproduce KillClient then Close with a real owned child/PTY and exact fake server evidence, then retire each client/PTY handle exactly once in the shared cleanup path.
- [ ] Preserve shutdown/observation/handle errors; audit this fixture’s callers and startup refusal paths so meaningful cleanup failures are joined or reported. Add a local lint regression for discarded fixture cleanup.
- [ ] Verify focused lifetime/error regressions and affected packages, run recorded verification, close and land.

Actions: detach client, close fixture, retry diagnosis after cleanup failure.

| Internal state / failure mode | Visible state | Recovery action | Possible loss |
|---|---|---|---|
| Client and PTY owned | KillClient consumes them once; server survives | close fixture | attached client ends intentionally |
| Client retired, server owned | Close proves exact server exit and removes owned roots | close fixture | server/session intentionally removed |
| Shutdown or observation fails | error retained and test fails; uncertain roots retained | retry diagnosis | no cleanup-success claim |
| PTY/child handle cleanup fails | meaningful error returned, ownership retired once | retry diagnosis | no repeated close/reap attempt |

ARCH-DRY/ORDER/FUNERAL: one helper consumes handles before closing/signalling/reaping, shared by detach and final close. Existing exact-generation server observation and cache-removal policy remain authoritative. Adds O(1) handle work, no new process scan, loop, worker or durable artifact.

## Log

Observed during #761 on 830f29d9ef0892b6fb8d06869a4840eed28f7ad9. Relevant source: cmd/internal/pairlifecycletest/live_zellij.go, KillClient and close; caller startup_detach_timing_live_test.go. Retained log /Volumes/TestScratch/tmp/pair-slot5/pair-761-native-reattach.Id4Vb5/native.log SHA25649fdd7e0a1e461926b7d13edfa342319d278f28531fea2c8d86e36f29bffac92. No fix is stacked into #761.

- Traced the fixture and callers. KillClient closes the PTY without clearing it, and keeps the waited process command; final close repeats both operations. Native timing drops the resulting error. Startup refusal paths and several ControlledZellij callers also discard Close results; auditing the same fixture class before code. #757 remains untouched on its preserved branch.
