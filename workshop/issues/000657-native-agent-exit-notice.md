---
id: 000657
status: working
deps: [pair#574]
github_issue:
created: 2026-10-17
updated: 2026-10-18
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'af121bfeee2d3d2d6d896c4ea8dcd1c765f93f7e' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-18T07:05:50-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
---

# Couch reports native agent exit without mistaking attach disconnect

## Problem

#574 cannot report a native agent exit reliably from the existing Couch observations. Console.onExit observes the Pair attachment client, and registry EOF may be a reconnect or wrapper replacement. Neither proves that the native agent stopped.

## Spec

After #574 lands, extend its lifecycle notices to confirmed native agent exit. Establish evidence from the actual owned native child lifecycle, retaining exact session/incarnation fencing and compatibility. Do not infer agent exit from attach-client exit, EOF, failed inspection or an orphaned session. Choose the smallest reliable mechanism during design; this issue does not preselect a new terminal protocol frame.

## Done when

- A confirmed current native child exit produces one hub system notice through #574's internal producer.
- Stateful tests distinguish native exit from detach, reconnect, wrapper replacement and stale completion; no false exit notice in the negative cases.
- Compatibility and an owned-child conformance check cover the chosen evidence seam.

## Plan

Durable plan: `workshop/plans/000657-native-agent-exit-notice-plan.md` (a single pass, one close).

- [ ] couchmessage `native-exit` op plus ID
- [ ] lifecycle `exited` kind
- [ ] Couch handler, with stateful negative cases
- [ ] wrapper report after Wait (self-exit only) plus compatibility
- [ ] owned-child conformance test
- [ ] atlas, guards, suite, close

## Log

- 2026-10-17: Filed unclaimed at the hub's request while simplifying #574. Relevant seams: cmd/internal/couchtty/console.go:onExit (attachment client); cmd/internal/wrapcmd/wrap.go native child group.Wait (actual child); couchmessage registry session currently has no terminal child-exit observation. No implementation or protocol change is included in #574.
