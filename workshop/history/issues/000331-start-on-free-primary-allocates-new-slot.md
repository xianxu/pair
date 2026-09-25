---
id: 000331
status: done
deps: []
github_issue:
created: 2026-09-25
updated: 2026-09-25
estimate_hours:
started: 2026-09-25T10:24:07-07:00
actual_hours: 0.18
---

# Starting a thread on a free primary allocates a new slot when numbered slots exist

## Problem

Operator report (ariadne): after the primary (:0) thread was parked and
archived, starting a thread on the primary path created a new numbered slot
(`ariadne-slot3`, 2026-09-25 10:20) instead of starting on :0.

Root cause: `resolveManagedStart` (`cmd/internal/couchcore/slotstart.go`)
treated :0 as occupied when `len(repository.Slots) > 0` or any record lived in
*any* of the repo's scopes (primary or numbered). With ariadne-slot1/2
present, :0 was "occupied" forever, so create always allocated :N.

## Spec

A create on the primary path starts on :0 iff :0's own scope holds no thread
record. Numbered slots and their threads don't occupy :0. Unchanged: a live or
parked :0 still leads to allocation (and parked work still blocks it via
`checkSlotCreation`); an unreadable record anywhere in the repo still refuses.
The thread menu's Add slot entry uses the same path, so it also fills a free :0
first.

## Done when

- `TestManagedCreateReturnsToPrimaryOnceItsThreadIsArchived`: primary thread,
  then slot1 via create, archive primary → create previews :0, not :2.
  Fails before the fix (got :2), passes after.
- Existing slot start/admission tests and full `make test` green.
- Atlas (`workspace-provisioning.md`) states the :0 occupancy rule.

## Plan

- [x] Regression test reproducing :2 allocation
- [x] Occupancy = records in the primary's own scope only
- [x] Atlas

## Log

### 2026-09-25
- 2026-09-25: closed — TestManagedCreateReturnsToPrimaryOnceItsThreadIsArchived fails before fix (preview chose slot 2) and passes after; couchcore slot start/admission tests + full make test green (scrubbed retention env, scratch TMPDIR); atlas workspace-provisioning updated; review verdict: SHIP

- Reproduced in test (preview chose slot 2), fixed in `slotstart.go`, full
  `make test` green. ariadne-slot3 left as is — it is a real slot now; archive
  it from couch if unwanted.
- Review fixes: regression test now also starts the thread and checks it lands on the primary; atlas rewrapped. Operator widened the policy to hole-filling for every slot number (follow-up issue).
