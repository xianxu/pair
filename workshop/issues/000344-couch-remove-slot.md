---
id: 000344
status: open
deps: []
github_issue:
created: 2026-09-29
updated: 2026-09-29
estimate_hours:
card_mirror: '57cd1a8b1a63571d6f85ec928c7c9cf01a078b60' # card fields mirrored from issue-cards; edit via sdlc
---

# Couch: remove slot (highest-numbered, operator-confirmed)

## Problem

Couch can **Add slot** (#313) but has no way to take one away. Removing one by hand
(`rm -rf` the worktree folder) leaves Couch wedged. A 2026-09-29 probe (real Git in a
temp fleet with the stateful process fake) showed that:

- `rm -rf worktree/<repo>-slotN/` while Git still registers it: the snapshot drops the
  slot row (and its conversation record, which lived in `<env>/.couch/`), but
  `Discover` keeps an unverified `incomplete slot host` candidate. A still-running
  process in that slot vanishes from the inventory and is orphaned.
- Removing only the inner checkout (`.couch/` kept) leaves an unreadable slot row.
  Resume refuses with `metadata requires recovery`, and `OpenSlot` refuses with
  `workspace environment already exists without ownership evidence`.
- While any slot is broken, **Add slot is blocked** for the whole repository:
  `repository slot N needs attention before creating another slot`.
- Even after the folders are gone and `git worktree prune` has run, the leftover
  `main-slotN` resting branch blocks re-creating that number: `resting branch
  main-slotN already exists without ownership evidence`. Because Add slot always
  picks the lowest free number, Add slot stays wedged until an operator runs
  `git branch -D main-slotN` by hand.

## Spec

A **Remove slot** entry in the thread action menu (next to Add slot) that removes
the **highest-numbered** slot of the selected repository. It removes only the
highest-numbered slot, so the numbering stays dense and Add slot's lowest-free-number
choice stays predictable.

- **Confirm first.** Before doing anything, show the operator exactly what will go:
  the slot label (`repo:N`), the worktree path, its branch, and any thread in it. The
  operator must confirm explicitly; the default is cancel. Nothing is touched before
  the confirmation.
- **Refuse, don't force**, when:
  - the slot's thread is live or attached. The operator parks it first; this action
    never kills a process.
  - the checkout has uncommitted or untracked changes, or is off its resting branch
    `main-slotN` (for example an issue branch is checked out there).
  - the resting branch has commits that aren't on `main`.
  - the slot is `:0` (the primary checkout).
  
  The refusal message names the next action to take.
- **Removal order** (each step must be safe to repeat after a crash, so an
  interrupted run can be resumed): archive the slot's current conversation and its
  retained `.couch/` history into the global archive, so native session transcripts
  stay GC-protected and restorable. Then `git worktree remove`, delete the
  `main-slotN` branch (only when it is merged or equal to `main`), and remove the
  `worktree/<repo>-slotN/` folder.
- **Recovery for already-broken slots:** the same action, pointed at an incomplete or
  pruned highest-numbered slot (the probe cases above), finishes the cleanup: stale
  worktree registration, leftover resting branch, orphan `.couch/`. That unblocks Add
  slot. It still refuses on a live process.
- A CLI or internal op mirror (`couch --internal remove-slot <primary>
  --slot=N`?) is open. Decide during planning whether it's needed. ARCH: keep the
  single op schema in `couchcore.Operations()`.

## Done when

- The thread action menu offers **Remove slot** for a repository with at least one
  numbered slot. It targets the highest-numbered slot, and nothing changes until
  the operator confirms (cancel is the default and has been tested).
- A clean, parked highest slot is removed completely: worktree, `main-slotN` branch,
  folder, and `.couch/` (its records archived and restorable). A following Add slot
  reuses that number.
- It refuses, and nothing changes, when: the thread is live, the checkout is dirty,
  it's off the resting branch, it has unmerged commits, or the target is `:0`.
  Each case is tested.
- The broken states from the probe (rm'd env with Git still registered, checkout
  removed with `.couch/` kept, pruned with the leftover branch) are cleaned up by the
  same action, and Add slot works again afterwards. Each case has a real-Git test.
- The couch atlas section on durable numbered slots documents the action.

## Plan

- [ ]

## Log

### 2026-09-29

- Filed from a live probe of manual slot-folder deletion (a temporary test in
  `couchcore`, not committed); the findings are in Problem. The key blocker is the leftover
  `main-slotN` branch wedging Add slot.
