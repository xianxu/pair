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

### 2026-09-17 — a second live wedge, and the operator's requirement for it

A fresh instance of the wedge this issue exists to remove, on `astro`. Its
zellij server panicked at startup (`pair#273`), leaving the launcher alive and
the session gone, so couch classifies the thread `live` from the launcher pid
(`pair#272`). Captured before it is reaped:

```
$ cd ~/workspace/astro && couch --list
astro                  /Users/xianxu/workspace/astro
  address: fcff31946c0ac9d2/couch-48340fd828040287
  recorded: live     pid 63065     # pair resume …, alive
  live                             # classifier, same rule as the switcher
```

This survives `pair#256` M1–M3 — measured with a binary built after M3 landed at
16:54, not the 15:49 process that was running.

Operator requirement, stated on the incident:

> I expect I can park it, and when park timeout, treat that as an error case,
> but user can continue to archive that parked — but failed — thread.

Three claims, and the middle one is already this issue's core (`pair#271`: a
timed-out park sits in `awaiting_completion` forever, no timeout, no expiry, no
owner-liveness check — the brain thread wedged ~18 hours on it):

1. Park is **attemptable** on a thread whose session is already gone. Park's job
   is to reach the parked state, and a torn-down session is that state reached
   early, not a precondition failure.
2. A park attempt that times out is a **confirmed failure** — a recorded
   terminal transition — not an open phase awaiting a completion that will never
   arrive.
3. A parked-but-failed thread remains **archivable**. This is the property that
   keeps a failure recoverable instead of terminal, and it is the one not stated
   anywhere today.

**Note on reproducing this:** the fixture is fragile. The launcher is couch's own
child, so restarting couch reaps pid 63065 and the thread reclassifies for a
reason unrelated to any fix — an easy way to credit a change that did nothing.
Capture the state first, or build the fixture deliberately in a test.

## Revisions

### 2026-09-17 — added the operator's failure-outcome requirement

**Reason:** a second live wedge (above) and an explicit operator requirement for
what a failed park must leave behind.

**Delta to `## Done when`:** add, as its own bullet —

- A park attempt that times out records a **confirmed failure** transition, and a
  thread left by one can still be archived. A failed park never produces a thread
  that refuses every gesture couch offers.

**Delta to `## Spec`:** park must accept a thread whose session is already
absent, and reaching the parked state by an unexpected route is a success, not
an error. The ordered idempotent write makes this natural — "is the session torn
down?" is the whole question, and a panicked server has already answered yes.

**Dependency note:** claim 3 is only reachable once the liveness witness stops
asserting `live` from the launcher pid (`pair#272`); until then the archive
guard sees a live thread and refuses regardless of what park recorded.

### 2026-09-17 (later) — after a couch restart: two guards, opposite answers, no exit

The operator restarted couch onto the post-M3 binary. The classification
improved; the thread is still unreachable, and now demonstrably so.

**The switcher row is right.** `astro` renders `session gone` while its
neighbours render `live` — so `ThreadUnusable` + `ReasonSessionGone`
(`couchcore/threadreason.go:103`). The phantom `live` is gone, which is what the
restart plus M3 bought.

**Enter on that row** notices:

```
error: astro: the session is gone; recover from a saved checkpoint or archive
```

`couchtty/menu.go:1183`, the `ReasonSessionGone` arm of `unusableThreadNotice`.
Consistent with the row, and it names archive as the way out.

**Archive, taken from that same row, refuses:**

```
threads > astro > archive
error: archive couch-48340fd828040287: it is live -- couch is hosting its
agent; detach or park it first
```

That string is `couchcore/detach.go:418` — the **`ThreadLive` arm** of
`archiveRefusal`. Not the unusable arm, which would have said "its state is
unresolved (session gone)". So archive classified the thread **live** at the
same moment the switcher classified it **session gone**.

Two findings, and the second outlives the first:

1. **The two paths disagree.** `pair#256` M3's close claims "three action guards
   read ONE authority… `ArchivableState` and `ResumableState` join
   `SwitchableState` as pure predicates over the classification", with
   `TestActionOfferedImpliesPermitted` comparing them against the switcher offer
   across `AllThreadStates × AllThreadReasons`. The offer here was made *and*
   refused, so either that equivalence has a hole or the two evaluations read
   different observations — `classifyForAction` re-observes, the row came from
   the menu snapshot. Which one is the first thing to measure.
2. **Even agreement would not fix it.** `archiveRefusal`'s arms are `live`,
   `busy` and `unusable/unknown` — so `ThreadUnusable/ReasonSessionGone` is
   *also* a refusal, with "nothing is known well enough to stop it, so retry".
   Agreeing on `session gone` would only change the wording of the no. **A
   thread whose session is proved gone is the most archivable state there is**;
   "session gone" should be archive's permission, not its refusal. M3 fixed the
   neighbouring case — "an `unknown` incarnation could never be archived at
   all" — and this one sits next to it.

**The operator is in a closed loop with no exit:** archive says detach or park
first; Enter says recover or archive; and alt+d is dead in the switcher
(`pair#279`), so the remedy archive names cannot be reached from where the
operator is standing. This is the concrete failing case for the requirement
recorded above — a failed/absent session must not produce a thread that refuses
every gesture couch offers.

The `pair` row in the same screenshot reads `continuation failed — retry
available`, which is `pair#249`'s retained-failure surface working exactly as
intended. That is the shape this wants: a failure that stays actionable.
