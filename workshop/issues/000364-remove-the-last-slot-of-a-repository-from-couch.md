---
id: 000364
status: open
deps: []
github_issue:
created: 2026-09-30
updated: 2026-09-30
estimate_hours:
card_mirror: '1d806d3fbd54a6ef569f6c465ab11babed0f09b5' # card fields mirrored from issue-cards; edit via sdlc
---

# Remove the last slot of a repository from couch
## Problem

Couch can add a slot to a repository (`add slot` on the live `:0` row) but has
no way to remove one. Slots accumulate as checkouts on disk and rows in the
switcher.

## Spec

- Offered as "remove last slot" on the live `:0` row only (see #363's action
  table). It removes the highest-numbered slot, so numbers stay dense and
  `repo:N` addresses never shift.
- Destructive, so it is guarded and confirmed:
  - the slot is not live (parked or detached slots are first parked/stopped
    with confirmation, or the removal refuses; decide at design);
  - its working tree is clean, with no untracked files;
  - its branch is the resting branch or fully merged, so no commits are lost.
- The slot's thread record goes to the archive (evidence preserved, as with
  #363's reboot); the Git worktree is removed through the existing slot
  provisioning seam, not ad hoc `git worktree remove`.
- A refusal names the failed guard and the next action.

## Done when

- Removing the last slot of a repository with `:0..:2` leaves `:0..:1`, with
  the worktree gone and the thread record archived.
- Each guard refuses with its own message and leaves the slot intact, with a
  test per guard.
- `:0` is never removable through this action.

## Plan

- [ ] Design the guards (not live, clean tree, no untracked files, branch resting or merged) and how a parked or detached slot is handled; record the decision in the Spec.
- [ ] Implement removal of the highest-numbered slot through the slot provisioning seam, archiving its thread record.
- [ ] Offer "remove last slot" on the live :0 row only, confirmed.
- [ ] Tests: removal of :2 from :0..:2, one refusal test per guard leaving the slot intact, :0 never removable.

## Log

### 2026-09-30

- Filed from #360's design conversation with the operator; split out because
  it is the one destructive new feature in the slot-world cleanup (#363).
