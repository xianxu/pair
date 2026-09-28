---
id: 000275
status: open
created: 2026-09-16
updated: 2026-09-16
estimate_hours:
github_issue:
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
