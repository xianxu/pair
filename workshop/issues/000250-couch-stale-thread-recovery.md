---
id: 000250
status: working
deps: []
github_issue:
created: 2026-09-14
updated: 2026-09-14
estimate_hours:
started: 2026-09-14T10:31:03-07:00
---

# Recover stale Couch threads without losing live sessions or checkpoints

## Problem

A failed hosted restart or lost Couch helper can leave a thread recorded as
live while Couch refuses to host it. The operator cannot resume work, and
archive can also refuse because the persisted incarnation is still occupied.
Recovery must be available without hand-editing the thread store or discarding
a surviving agent conversation.

Reported 2026-09-14 for Pair address
`e108517d46ab4575/couch-36b623f6869ebaa2`, following the continuation failure
tracked in #249. Read-only `couch --show pair` reported recorded live PID 5330
and `unusable: stale — couch exited unexpectedly`. At inspection time,
`zellij list-sessions --no-formatting` showed no live Pair session. The operator
had reported an archive refusal attributed to a live Zellij session; the exact
UI error was not captured, so distinguish that report from current observation.

Source inspection found `archivableRecord` rejects occupied persisted
incarnations before `Couch.ArchiveThread` reaches session quiescence. Thus a
stale live record can block the documented manual escape even when its process
has died. Recheck runtime evidence and the exact refusal before repair.

The archived proposal
`workshop/history/issues/000171-reconcile-stale-incarnations-after-crash.md`
was punted because manual archive was considered sufficient. This incident
invalidates that premise. Refer to that file by path: another archived issue
also uses ID 000171. #214 covers a related launch-race recovery case; #249
prevents the continuation failure. This issue owns recovery from an already
stranded thread, independently of those triggering bugs.

## Spec

Provide an operator-accessible recovery path from a stale/unusable Couch row,
with a diagnosis and available next action. Recover ownership through Couch's
existing store transactions and session observation mechanisms; do not ask the
operator to edit JSON or bypass ownership checks (ARCH-DRY, ARCH-SECURE).

Distinguish the dead Couch helper from its potentially surviving Zellij and
agent processes. Retire only incarnations whose exact PID and process-start
identity prove they are dead. Unknown observations, active clients, ambiguous
ownership and open lifecycle transactions require an actionable explanation,
not an assumption of death. Recheck evidence at execution and serialize recovery
with start, park, resume, relaunch, archive and other recovery attempts
(ARCH-ORDER).

When an exactly owned live detached session survives, restore Couch attachment
to that session, preserving the running agent. Warm reattachment must not require
a native transcript binding or silently fall back to a new agent (#248).
When the source session is gone, preserve and offer recovery from the exact
saved continuation, including one authored in another worktree. Explicitly
distinguish a fresh conversation seeded by that checkpoint from native
conversation resume; only offer native resume when its binding is verified.
Missing or unreadable recovery evidence must be reported without silently
launching an empty conversation.

Make archive a reachable explicit escape from a reconciled stale row. Preserve
checkpoint, native transcript references and Pair history. A genuinely live
session must not be terminated as a side effect of inspection or reconciliation;
any stop/archive must follow the operator's explicit action and verified scope.
Do not let one stale row permanently prevent work in its repository.

Include recovery of the currently blocked Pair thread as an acceptance step,
after rechecking its identity and evidence. Select the operator's intended
recovery source and verify usable access through Couch. Record the outcome;
a unit-test-only fix does not resolve the reported operational blockage.

## Done when

- A stale row exposes a working recovery action and truthful diagnosis; a dead
  helper with a surviving detached session reattaches to the same agent.
- A dead session with a valid checkpoint can recover into usable Couch-hosted
  work seeded from that exact checkpoint, including across worktrees.
- A stale persisted live incarnation cannot permanently block both recovery
  and explicit archive. Missing recovery evidence still leaves a clear escape.
- Stateful tests cross inventory, UI eligibility, process observation, store
  transition and attachment/archive. Cover live/dead/unknown processes, PID
  reuse, absent native binding, occupied sessions, missing checkpoint, open
  transactions, concurrent actions, interruption and retry without duplicate
  ownership or silent loss of conversation context.
- The actual blocked Pair thread is recovered with operator-confirmed usable
  access; docs describe recovery choices and failure handling.

## Plan

- [ ] Reproduce the stale-record/archive dead end and surviving/dead-session variants with stateful fixtures.
- [ ] Design recovery choices, ownership proofs and interruption handling in a durable plan.
- [ ] Implement the shared recovery path, verify UI-to-store behavior and update the Couch atlas.
- [ ] Recheck and recover the reported Pair thread, preserving its checkpoint and history; record outcome and close through SDLC review.

## Log

### 2026-09-14

Created at operator request specifically for recovery, after locating the
punted reconciliation proposal and checking the current stale Pair record.
The checkpoint was found at
`/Users/xianxu/workspace/worktree/pair/000239-pair-s-own-data-store-has-no-garbage-collection-13-gb-under-local-share-pair-and-nothing-ever-prunes-it/workshop/continuation/20260913T210234-pair-storage-retention.md`.
Runtime addresses and paths are incident evidence, not authority to act later
without fresh verification. No sessions or runtime metadata were modified;
implementation has not started.


### 2026-09-14 — Operator-authorized manual unblock

Detach-all failed partway through on exact address
`e108517d46ab4575/couch-36b623f6869ebaa2`, reporting no live Pair session.
Operator explicitly requested manual state repair before implementing #250.
Rechecked: recorded PID 5330 absent, exact Zellij session `📁pair-couch-27`
absent, revision 22, identity `1789341245.53810`, one live incarnation,
no open start/park transaction. Backed up the entire threadstore and exact
#239 continuation to `/tmp/pair-250-manual-recovery-57qbkgg9`.

A temporary Go helper called the existing `ThreadStore.RetireIncarnation`
transaction with revision 22 and that exact identity, immediately rechecking
`OSProcOps.Exists == Dead`. Passed the existing LastActiveAt to preserve
historical activity rather than claiming the absent session ran today.
Result revision 23, no incarnation, no fabricated VerifiedPark. Raw comparison
proved only this record changed, and only its revision/incarnations fields;
all other thread records unchanged. Helper source retained in backup and
removed from checkout. `couch --show pair` now truthfully reports `session gone`.
`Couch.Leave` skips records without an active incarnation, so the identified
stale-record detach blocker is removed. No process/session was signalled or
archived. Operator must retry detach-all and confirm usable access; the absent
Pair conversation is not restored by this repair. General #250 remains open.

## Revisions

### 2026-09-14T10:40:00-07:00 — Reconcile then reuse recovery operations

Operator approved design and implementation of #248 → #249 → #250, pausing for
smoke after #248. Coordinated contract is recorded in
`workshop/plans/000248-couch-warm-reattach-gate-plan.md`; #250 receives its own
executable plan before implementation. Its responsibility is exact stale-owner
reconciliation and recovery choices, reusing #248 reattachment and #249 durable
checkpoint execution rather than building another launcher. Diagnose observed
helper/session state without asserting the whole Couch supervisor crashed.
The operational acceptance baseline includes the manual incarnation retirement
already logged above; usable conversation recovery remains to be confirmed.
