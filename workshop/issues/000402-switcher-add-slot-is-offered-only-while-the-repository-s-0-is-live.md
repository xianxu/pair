---
id: 000402
status: working
deps: []
github_issue:
created: 2026-10-06
updated: 2026-10-06
estimate_hours:
card_mirror: '27615b541a4c90755e90b248fb7ed062bb149f41' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-06T22:11:19-07:00
claimant:
    operator: T
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "cd2a532f", done: "7c7d4e5d"}
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

- **Spec bullet 2, checked:** the add-slot path (`couchtty/menu.go`, the
  `add-slot` case) reads only `menuAddSlotPath(thread)` and opens the start form.
  It never reads the `:0` actor, so no decoupling was needed.
- **Fix:** `menuActionItems` appends `add-slot` for resumable and unusable `:0` rows
  unless `DirectoryMissing` holds, because a slot needs the primary checkout.
- **Close review (FIX-THEN-SHIP), fixed as rules:**
  - BR-1: swept every "live `:0`" claim about add slot (README how-to, the
    `menuRowActions` doc, the sweep comment). The alias claims stay, because alias
    is still live-only.
  - BR-2: the advice sweep now requires an `OnPrimary` action to be offered by
    `:0` when live, parked and detached. The mutation (dropping the fix) fails it
    on the `:1+` directory-missing notice.
- **Live smoke test (Done-when clause 2): waived by the operator on 2026-10-06**
  ("this is so small, you got to get it right"). Testing needs a Couch restart,
  because the switcher runs inside the live supervisor.
