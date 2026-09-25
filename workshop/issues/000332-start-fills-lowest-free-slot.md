---
id: 000332
status: open
deps: []
github_issue:
created: 2026-09-25
updated: 2026-09-25
estimate_hours:
---

# Starting a thread fills the lowest free slot number, :0 included

## Problem

Starting a thread on a repo leaves "holes" in the slot numbers. Archiving a
thread keeps its slot's worktree on disk, and `SelectNewSlot` treats every
existing slot directory as taken. So an archived `:1` is never reused and the
next start allocates `:N+1` (ariadne went to `ariadne-slot3` while `:0`/`:1`
had no thread). #331 fixed only the `:0` case with a `:0`-specific check.

Separately, #306's admission rule refuses to create any new slot while a
parked thread exists anywhere in the repo.

## Spec

Operator decisions (2026-09-25):

- **One rule for every number, `:0` included — no `:0` special case.** A slot
  number is *free* when it has no live or parked thread: never created, or
  its thread archived. Replaces #331's `:0`-only occupancy check.
- **Start fills the lowest free number.** An existing directory with no thread
  (e.g. archived `:1`) is reused as-is: the new thread inherits whatever
  branch and uncommitted files the old one left. Only when every number
  `0..N` is taken is a new slot `:N+1` created.
- **Parked threads occupy their number but no longer block creation.** When
  there is no hole and some threads are parked, create the next slot, and the
  start preview names the parked threads as a reminder. This reverses #306's
  "parked blocks new slot" rule to a notice.
- Unchanged: an unreadable record anywhere in the repo still refuses (can't
  tell whether its number is taken); the preview/fingerprint contract still
  refuses a changed selection instead of renumbering.
- Consequence: the thread menu's **Add slot** uses the same path, so it fills
  a hole before adding a number.

## Done when

- Tests: holes at `:0`, at `:1` with `:0`/`:2` live, and at `:1` with `:2`
  parked each resolve to the hole; the reused `:1` keeps its leftover files;
  no hole + a parked thread creates `:N+1` and the preview lists the parked
  thread; unreadable record still refuses.
- #306's "parked blocks create" tests updated to the notice behavior.
- Atlas (`workspace-provisioning.md`, `couch.md` slot section) states the rule.
- Operator smoke test in couch: archive a slot thread, start on the repo, land
  on the freed number.

## Plan

- [ ] `sdlc start-plan`; locate the occupancy rule (`slotstart.go`
  `resolveManagedStart`, `SelectNewSlot`, `checkSlotCreation`) and fold it into
  one per-number occupancy function
- [ ] Tests first for the cases above
- [ ] Preview notice listing parked threads
- [ ] Atlas; smoke test

## Log

### 2026-09-25
