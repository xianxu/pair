---
id: 000657
status: open
deps: [pair#574]
github_issue:
created: 2026-10-17
updated: 2026-10-17
estimate_hours:
collaboration-mode: collaboration
card_mirror: '9329ea390f72797e50249bf78d365a03b5286802' # card fields mirrored from issue-cards; edit via sdlc
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

- [ ]

## Log

- 2026-10-17: Filed unclaimed at the hub's request while simplifying #574. Relevant seams: cmd/internal/couchtty/console.go:onExit (attachment client); cmd/internal/wrapcmd/wrap.go native child group.Wait (actual child); couchmessage registry session currently has no terminal child-exit observation. No implementation or protocol change is included in #574.
