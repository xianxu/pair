---
id: 000387
status: open
deps: []
github_issue:
created: 2026-10-02
updated: 2026-10-02
estimate_hours:
card_mirror: '7d7197ce467f5d38cbad25b365efe77a14f64807' # card fields mirrored from issue-cards; edit via sdlc
---

# Add slot repairs a slot whose directory was deleted

## Problem

A slot directory deleted without `git worktree remove` (for example `rm -rf
worktree/pair-slotN`) leaves git's worktree registration behind until `git
worktree prune`. Reading the code (not yet reproduced): `SelectStartSlot`
(`couchcore/slotallocation.go`) refuses to allocate any number while a slot
candidate is unverified or carries an error ("slot N needs attention"), so one
half-deleted slot may block add slot for the whole repository. Even when the
number is free, `git worktree add` at a path git still has registered may
fail. And a stale thread record for the slot keeps its number occupied, so add
slot skips the hole instead of filling it.

## Spec

Add slot repairs this case instead of refusing. The repair runs inside
provisioning, the guarded creation boundary add slot already uses: before
creating slot N at its path, if git holds a registration for exactly that path
and no directory exists behind it, clear that one registration, then create
(prefer a targeted removal over a repository-wide `git worktree prune`; verify
which git form works for a missing directory). A locked registration is never
cleared: the repair stops and names it. Allocation treats a registered but
missing slot as a repairable hole rather than an error that blocks every
number, so the lowest free number can be that hole. A stale thread record for
the missing slot must be archived (never deleted) before its number is free.

The branch and its committed work survive as refs; uncommitted files were lost
with the directory and the repair does not pretend otherwise. The slot's
substrate clones (e.g. its `ariadne`) are re-provisioned as usual. A claim
recorded on that worktree path (fleet inventory's `dangling_claims`) is
reported when the slot is recreated, not silently inherited by the new slot.

## Done when

- A test reproduces the `rm -rf` case first: the current refusal (whole
  repository blocked, or `git worktree add` failing) is observed before the fix.
- After the fix, add slot fills the hole: the stale registration for that path
  is cleared, the worktree and its substrates are re-provisioned, and other
  slots' registrations are untouched.
- A locked registration stops the repair with its reason; nothing is cleared.
- A stale thread record keeps the number occupied until archived; archiving
  then lets add slot reuse the number.
- A dangling claim on the recreated path is reported to the operator.

## Plan

Implementation plan to be designed after issue claim and start-plan.

## Log

### 2026-10-02

Filed at the operator's request from the #363/#367 design talk: a deleted slot
directory should be repaired by prune-then-add-slot, not left blocking.
