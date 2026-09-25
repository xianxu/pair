---
id: 000332
status: working
deps: []
github_issue:
created: 2026-09-25
updated: 2026-09-25
estimate_hours:
started: 2026-09-25T10:42:40-07:00
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

Design (from code reading + a probe test, 2026-09-25):

- Occupancy source = the switcher's own classifier:
  `ActionableThreadInventoryContext` rows belonging to the repo. Row number =
  `Target.Slot.Number` for slot rows, `0` for an ordinary row in the primary's
  scope. A number is *occupied* iff some row with that number carries a thread
  (non-empty `Address`). The probe showed an archived `:1` stays as a slot row
  with empty address (`unusable`/`never-started`) — that is a hole.
  Unreadable/unknown rows still refuse (can't know their number's state).
- One pure function `lowestFreeSlot(numbers occupied, existing dirs) → (n,
  exists)` (ARCH-PURE), unit-tested; `resolveManagedStart`'s StartCreate
  branch maps its answer to an action, replacing #331's `:0`-only check and
  the "any slot exists ⇒ occupied" rule:
  - `0` → ordinary primary start (as today when free);
  - `n` with an existing directory → `StartFresh` on that slot (StartFreshSlot
    already starts a new conversation in an existing, ownerless slot);
  - `n` without a directory → `StartCreate` of slot `n` (provision as today).
- `checkSlotCreation` drops the parked blocker (keeps the unreadable refusal);
  `StartResolution` gains `ParkedInRepo []string` (labels, omitted from the
  fingerprint so a park elsewhere doesn't invalidate a preview), rendered in
  the start form as `  parked: repo:1, repo:2`.
- ARCH-FUNERAL: creates nothing durable beyond what slot create already made;
  reuse *reduces* slot-directory growth. No new files.

Steps:
- [ ] Unit tests for the pure selector; integration tests for holes at :0,
  :1 (dir exists, thread archived — reused with leftover file), :1 with :2
  parked, no-hole+parked → new slot with notice, unreadable refuses
- [ ] Implement selector + resolveManagedStart mapping; drop parked blocker
- [ ] Preview notice in start form (+ render test)
- [ ] Update #306 parked-blocks tests to the notice behavior; atlas
- [ ] Full make test; operator smoke test in couch

## Log

### 2026-09-25
