---
id: 000542
status: open
deps: []
github_issue:
created: 2026-10-13
updated: 2026-10-13
estimate_hours:
card_mirror: '6e8d14b70cc1aaca216c0496c3a8f05b17ec2db8' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-13T18:00:12-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
---

# PTY canceled-before-ack reports a live process group after the child is reaped

## Problem

TestBlockedRunnerCancellationConformance/pty/canceled-before-ack failed in /tmp/pair536-make-k.log at runner_contract_test.go:59: canceled blocked child remained alive. It passed three consecutive isolated reruns (/tmp/pair536-runner-rerun.log).

## Spec

Measure the cancellation/observation race without relaxing timing. Cancel closes the acknowledgement writer and terminal.Close kills/reaps the direct child; Wait observes that child, while Alive probes the whole process group. Before acknowledgement the launch helper cannot exec the target or create target descendants. Preserve process-group ownership guarantees.

This is independent of #519: it uses couchcore runner/helper code, not notifytransport or the wrapper output pump. A Darwin group-teardown/probe race is a hypothesis, not a proven cause. Capture PID, exit status, direct-child and group liveness, and group members on failure.

## Done when

- Root cause reproduced with retained process evidence and fixed without a sleep or weakened descendant check.
- Cancellation conformance passes repeatedly alone and in the full suite.

## Plan

- [ ] Retain failure diagnostics and establish the failing interleaving.
- [ ] Fix ownership or observation at its source; add regression coverage and verify.

## Log

### 2026-10-13

- Filed from #519 investigation at TL request. Read-only source analysis confirms a separate subsystem; no live-slot changes or additional cancellation loops run.
