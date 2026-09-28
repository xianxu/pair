---
id: '000171'
status: punt
created: 2026-09-02
updated: 2026-09-03
---

# Reconcile stale incarnations left by a crashed couch

## Problem

A couch that exits *cleanly* now detaches every thread (`pair#170` M2), so its
agents keep running and the next couch reattaches them. A couch that dies
**without** leaving cleanly — crash, SIGKILL, power loss — does not, and the
records it leaves behind are invisible and unresumable:

- The child's pty closes with the supervisor, so the Pair client dies and the
  zellij session survives with zero clients. The durable record keeps
  `Incarnations: [{State: IncarnationLive, PID: <dead>}]`.
- `ProjectActionableThreads` requires **zero** incarnations for `ThreadDetached`
  and a matching TTY observation for `ThreadLive`, so the row is emitted as
  neither and does not appear in the switcher.
- `DecideResume` refuses any record with a Live/Creating/Unknown incarnation
  (`couchcore/resume.go:73-86`), so even addressed directly it cannot resume.

This is **pre-existing**, not a `pair#170` regression: `reconcileInterruptedStarts`
(`couchcore/couch.go`) only touches records with an *open start transaction*, so
a fully-started thread has never been reconciled. #170 makes it more visible by
making detached the normal resting state, and supplies the mechanism the fix
needs.
