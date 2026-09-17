---
id: 000275
status: open
deps: [pair#256]
github_issue:
created: 2026-09-16
updated: 2026-09-16
estimate_hours:
---

# Replace the park transaction with an ordered idempotent write

## Problem

Park carries a durable multi-phase transaction — `ParkTransaction{Phase,
Attempts[], Nonce, Closed, Tombstoned, SuccessfulAttempt}` plus `ParkHistory[]`
and `VerifiedPark` (`couchcore/parktransaction.go`, `park.go` ≈812 lines). That
machinery is a durable shadow of one directly observable fact: **is the zellij
session torn down?**

Split out of `pair#256` on 2026-09-16. Operator framing, which is the reason this
issue exists:

> if couch crash, since we know zellij is not affected, thus all threads'
> essentially live, we should pick the default, that that state is recoverable,
> not relying on clean "shutdown" signal.

Measured basis (`pair#256`'s Log): the zellij server is **PPID 1 at birth**, so a
couch death kills only the launcher — couch's own child. `Detach`
(`detach.go:91-99`) SIGTERMs that launcher and clears the incarnation, so a clean
detach and a couch crash leave **identical external state**. The bookkeeping is
the only difference, and treating its absence as brokenness is what wedges
threads.

The cost of the shadow is measured: `pair#271` — a park whose attempt timed out
leaves `Phase: awaiting_completion` forever, with no timeout, no expiry and no
owner-liveness check. The operator's `brain` thread sat wedged for ~18 hours and
was unreachable by every gesture couch offers.

`pair#256` stops the **classifier** reading `record.Park`, which unwedges the
operator. It deliberately does not touch how parks are **performed**. This issue
does that.

## Spec

Park has exactly one property that is not cosmetic: it is **irreversible and
ordered**. The native session id and the preserved scrollback must be captured
*before* the session is torn down, because afterwards the source is gone. That
constraint needs ordering and idempotence — it does not need phases, numbered
attempts, nonces, tombstones or a history array.

- Replace the transaction with: **write the resume payload durably first
  (idempotent, safe to repeat), then tear the session down.** A crash anywhere in
  between is safe — the payload is saved and the session's state is directly
  observable on the next look.
- Remove `ParkTransaction`, `ParkHistory` and the phase/attempt vocabulary from
  the durable record, keeping `VerifiedPark`'s **payload** (native session id
  correlation, scrollback) which is genuinely not derivable from the world.
- Retire the permanent-tombstone rule. `DecideResume` currently refuses forever
  on any historical tombstone absent a `VerifiedPark`, so one abandoned park
  removes cold-resume authority for good.
- Decide what `RecoverActiveParks` (`couch.go:74`, wired at `couchcmd/run.go:348`)
  becomes when there is no transaction to reconcile.
- Schema: `threadrecord` decodes with `strictjson` (`DisallowUnknownFields`), so
  removed fields must become decode tombstones or every existing record is
  bricked. Follow the `claim_generation`/`policy` precedent in
  `threadrecord/compat_test.go`.

## Done when

- A park is a durable payload write followed by a teardown, with no durable
  phase/attempt state, and it is safe to repeat at any point.
- Killing couch at each step of a park leaves a thread that is resumable or
  parked — never a state needing a reconciler to interpret.
- No historical park event can make a thread permanently unresumable.
- Existing records still decode; the removed fields are tombstoned, not deleted.
- Sequence tests cover: crash before the payload write, crash between write and
  teardown, crash mid-teardown, repeated park of the same thread, and teardown
  that reports success while the session survives (`pair#274`'s shape).

## Plan

- [ ] Land `pair#256` first — it establishes the session-first classification this
      rests on.
- [ ] Claim/start-plan; durable design before code.
- [ ] Implement the ordered idempotent write with schema tombstones.
- [ ] Sequence/fault tests at the production boundary.
- [ ] Verify, atlas, close through SDLC.

## Log

### 2026-09-16

- Split out of `pair#256` at the operator's direction while re-cutting that
  plan: *"yes, agree for Park, and yes, simplify it as you mentioned. sure, make
  a new issue for it."*
