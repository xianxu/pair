---
id: 000402
status: open
deps: []
github_issue:
created: 2026-10-06
updated: 2026-10-06
estimate_hours:
card_mirror: 'd7efa7cb6acbc549fc9d5823f119ca1fa2d11404' # card fields mirrored from issue-cards; edit via sdlc
---

# Switcher: add slot is offered only while the repository's :0 is live

## Problem

When a repository's `:0` thread is parked, the switcher offers no "add slot", so the
operator cannot create a `:1+` slot without first resuming `:0`. Reported by the
operator on 2026-10-06 while setting up scratch slots for pair#362's exercises.

Cause: `menuActionItems` (`cmd/internal/couchtty/menu_actions.go`) appends
`add-slot` only in the live-phase branch, for a primary row with `AddSlotOffered`.
A parked (resumable) or unusable primary row gets only `couchcore.ActorActions`
(resume, reboot). Creating a slot provisions a new worktree and starts its own
agent, and needs nothing from `:0`'s agent, so tying it to `:0` being live is
accidental.

## Spec

Captured for operator review; no implementation is authorized by this issue
creation.

- A primary row offers `add-slot` whenever `AddSlotOffered` holds, whatever the
  primary thread's phase (live, resumable, unusable). The live-phase ordering is
  kept.
- First verify that the add-slot execution path (`couchtty/menu.go`, the `add-slot`
  case) does not read the live `:0` actor. If it does, decouple it rather than
  resume `:0` implicitly.
- The row-advice sweep (`TestRowAdviceNamesOnlyReachableActions`) and the
  menu-action tables must still pass. An `OnPrimary` add-slot advice on a `:1+`
  row should be reachable when `:0` is parked too.

## Done when

- A parked or unusable `:0` row offers add slot, with a menu-action test for each
  phase.
- Add slot from a parked `:0` creates and starts a `:1+` slot without resuming `:0`.
  Covered by a test, and smoke-tested live by the operator.

## Plan

- [ ]

## Log

### 2026-10-06
