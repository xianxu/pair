---
id: 000171
status: open
deps: []
github_issue:
created: 2026-09-02
updated: 2026-09-02
estimate_hours:
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

## Spec

On couch startup, retire incarnations that can be **proved** dead, so the thread
falls back to the same detached/parked classification a clean exit produces.

The transition already exists: `ThreadStore.RetireIncarnation` (added by
`pair#170` M2) removes exactly one live incarnation whose `{PID, Identity}`
matches, leaves no verified park, and keeps `LatestLaunchProfile`. What is
missing is the reconciler that decides *when* to call it.

Proof, not assumption — this is a fail-closed projection and must stay one:

- `observeExactProcess` must report `Dead` for the recorded `{PID, Identity}`.
  `Unknown` is **not** dead: a process couch cannot observe has not been proved
  gone, and retiring it would drop a live thread's incarnation.
- A record with an open park or start transaction is left alone; those have
  their own recovery paths and two reconcilers over one record is how they
  fight.

Out of scope: any change to the detach or park paths, and any reconciliation of
whole-incarnation *quiescence* (that is `pair#152`'s territory).

## Done when

- A couch whose predecessor died without leaving finds its threads in the
  switcher again — detached where the zellij session survived, and diagnosable
  rather than invisible where it did not.
- A live thread whose process couch cannot observe is **never** retired; a test
  crosses Dead / Unknown / Live against the recorded identity, including a
  recycled PID whose start token differs.
- Records with an open park or start transaction are untouched by this pass.
- `atlas/couch.md` records the reconciler beside the existing
  "Liveness is recomputed, never stored" rule.

## Plan

- [ ] Pure decision: given a record and an exact process observation, retire /
      leave / defer-to-existing-recovery. Table-driven, no IO.
- [ ] Wire it into couch startup beside `reconcileInterruptedStarts`, using
      `RetireIncarnation` for the retire arm.
- [ ] Restart-level test: build a record in the shape a crashed couch leaves,
      construct a fresh `Couch`, and assert the row returns.

## Log

### 2026-09-02

Filed from `pair#170` M2, which named this gap explicitly rather than absorbing
it: detach's whole point is that the *clean* path is safe, and conflating a
crash with a clean exit is exactly the fail-closed weakening M2 refused to make
in the projector.
