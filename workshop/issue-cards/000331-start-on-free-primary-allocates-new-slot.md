---
id: '000331'
status: done
started: 2026-09-25T10:24:07-07:00
created: 2026-09-25
updated: 2026-09-25
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
