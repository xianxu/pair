---
id: 000230
status: working
deps: []
github_issue:
created: 2026-09-11
updated: 2026-09-11
estimate_hours: 1.25
started: 2026-09-11T09:55:31-07:00
---

# A failed warm reattach deletes the session it was reattaching

## Problem

**A warm reattach that fails after its child is acknowledged deletes the zellij
session it was reattaching, and with it the running agent.** A warm reattach
exists to preserve that agent.

The path, from the code:

1. `ResumeContext` proves the thread detached and calls `launchTrackedThread`
   with `Resume: true, Warm: true`. The child is a bare `pair resume <tag>`,
   which attaches a client to the surviving session.
2. After `h.Acknowledge()`, every failure goes to
   `failTrackedPostAckStart(in.Resume, …)` (`launch_existing.go`). That covers
   the context being cancelled, the 15 s Pair registration timeout, and a
   failed `AdvanceStart`. The function takes `Resume` but not `Warm`.
3. For any resume, it calls `quiescePostAckStart`, which ends the helper and
   then calls `c.Artifacts.Quiesce(address)`.
4. In production that is `launcher.QuiesceThreadSession`, which deletes the
   thread's bound session: `zellij delete-session --force`, plus a kill of
   that session's server (`session_quiescence.go`).

Quiescing is right for a **cold** resume, which created that session and owns
cleaning it up. For a warm one, the session predates the attempt.

**Evidence.** A temporary test at the fake seam (2026-09-11) cancelled a warm
`ResumeContext` from the runner's `AfterAcknowledge` hook. The fake checker
recorded a `Quiesce` of the thread's address. The resume returned `context
canceled`, and the binding was then absent. No existing test covers a warm
reattach failing after acknowledgement.

**Reachable today:**
- Closing the terminal (SIGHUP) or sending SIGTERM while an operator's resume
  waits for Pair to register: `Run` returns, `teardown` cancels the lifetime
  context, and the in-flight resume takes the post-ack path.
- A warm reattach whose `pair resume` takes longer than the registration
  timeout to register, for example on a loaded host. Startup's own resume of
  the cwd thread is exposed to this too.

**#206 makes it much more reachable.** Its background pass reattaches every
detached thread one after another for several seconds after each startup, and
the operator's quit gestures and the last-pane exit are not held off while it
runs.

## Spec

- **A warm reattach's failure path ends only the client it started, never the
  session.** The helper (the `pair resume` client) is ended as today. The
  durable session is not quiesced. Killing a client leaves its session and
  agent running: that is the behaviour `TestSessionDetachLive` pins against
  real zellij.
- **The thread returns to detached.** Once the helper is proven dead, the
  start's own durable write is undone: a start claim rolls back, a live
  incarnation is retired. The thread then has no incarnation and reads as
  Detached again, resumable by hand. Session presence gates only the retire
  (the proof `Detach` already requires); a rollback removes nothing but this
  start's claim, so it does not need one.
- **Fail closed, but never destructively.** If the session's presence cannot
  be observed, mark the start unknown, as the cold path does today. Deleting
  the session is never the answer to not knowing.
- **Cold resume is unchanged.** It still quiesces the session it created.
- Enumerate every post-ack exit of `launchTrackedThread` and cover each for the
  warm shape. Fix the class, not the cancellation site.

## Done when

- A table test over the post-ack failure causes (cancel after acknowledge,
  registration timeout, failed acknowledge, failed registration promotion)
  shows, for a warm reattach:
  - no `Quiesce` of the thread's address;
  - the helper ended;
  - the thread rolled back with no incarnation, classifying Detached again
    while its session is observed detached.
- The same table for a cold resume still quiesces: the existing behaviour is
  pinned, not changed.
- Reintroducing the quiesce on the warm path fails the test (a mutation check).
- Unsandboxed `make test` passes.

## Estimate

Derived after the plan cleared plan-quality (2 rounds). Sized against `#228`,
the nearest comparable in this repo: same package, same
counted-invariant-plus-mutation-sweep shape, est 1.98 / actual 1.26. This is
less exploratory — the design is settled and the route table is enumerated —
but more test-heavy, so it lands near #228's actual rather than its estimate.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
design-buffer: 0.15
item: greenfield-go-module   design=0.10 impl=0.16
item: smaller-go-module      design=0.03 impl=0.08
item: smaller-go-module      design=0.05 impl=0.20
item: smaller-go-module      design=0.08 impl=0.24
item: atlas-docs             design=0.03 impl=0.04
item: milestone-review       design=0.00 impl=0.20
total: 1.25
```

- **greenfield-go-module** — `startcleanup.go`: the pure decider plus its
  24-row exhaustive table. New file, one concern, no IO.
- **smaller-go-module** — the fake's `Quiesce` made stateful, with its own
  test. Small, but it is what makes every later assertion falsifiable.
- **smaller-go-module** — the six-route × {warm, owning} table at the fake
  seam. The largest test item: each row needs its own injection.
- **smaller-go-module** — carrying `ActorRecord.Warm`, `applyStartCleanup`,
  and extracting `retireDetachedIncarnation` from `Detach` without disturbing
  its tests.
- **atlas-docs** — the lifecycle paragraph and the `quiescePostAckStart`
  comment.
- **milestone-review** — one boundary review at close.

Design buffer is 0.15 rather than 0.30: the work has a thorough plan doc that
has already cleared the gate.

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.*

## Plan

- [x] Design: `workshop/plans/000230-a-failed-warm-reattach-deletes-the-session-it-was-reattaching-plan.md`.
- [x] Red: the warm post-ack failure table at the fake seam, over all six
      routes.
- [x] Carry the start's SHAPE (not a warm boolean) into the post-ack failure
      path, decided by the pure `DecideStartCleanup`. For warm: end the helper,
      skip the quiesce, then roll back or retire on the observed presence.
- [x] Pin the owning paths' quiesce in the same table.
- [x] Mutation check (8/8); unsandboxed full suite (197 ok).

## Log

### 2026-09-11

Found while designing #206's background reattach pass. The pass runs warm
reattaches behind the operator's back, and the question was what quitting
mid-reattach does to one. The answer was that it deletes the session.

## Revisions

### 2026-09-11 — the class has six routes, not four

**Reason.** Designing the fix enumerated the callers of
`quiescePostAckStart`, which is the only caller of `Artifacts.Quiesce`. Four
exits of `launchTrackedThread` reach it through `failTrackedPostAckStart`, as
filed. Two more reach it through `failPostAckStart`, and both destroy a warm
session the same way:
- **route 5**, the registry persistence failure at the end of
  `launchTrackedThread`;
- **route 6**, `AbortStarted`, which `couchcmd` calls when the console cannot
  attach a started child. It covers startup's initial attach and every
  switcher resume.

`reconcileInterruptedStarts` never quiesces, so it is not a member.

**Delta.**
- The fix is a property every route reads: whether the start **owns** its
  session. A spawn and a cold resume created theirs; a warm reattach did not.
- `StartResult` carries `Warm`, so route 6 can read it.
- Routes 5–6 retire the live incarnation under Detach's two proofs, instead
  of rolling back a start.
- Done-when's failure table covers all six routes. The plan file has the
  table.

### 2026-09-11 — the rollback rule, stated once

**Reason.** The plan gate (PQ-2) found the Spec and the plan disagreeing on
whether session presence gates the routes-1–4 rollback. The Spec said it did;
the plan said it did not. One of them had to be wrong in writing.

**Delta.** The rule is now stated once, above, and is the plan's:
- **Rollback** (a start claim, routes 1–4) needs only a dead helper. It
  removes this start's own claim and nothing else. If the session survived,
  the thread reads Detached; if it died independently, `session-gone` — which
  is the honest answer either way.
- **Retire** (a live incarnation, routes 5–6) needs the dead helper *and* an
  observed present session, which is exactly what `Detach` requires before it
  retires an incarnation.
- Neither is reached while the helper is unaccounted for: that stays
  `mark-unknown`.

The pure decider `DecideStartCleanup` is where the rule lives, so the two
documents cannot drift again.

### 2026-09-11 — implemented

**The class was six routes; the input was three-valued, not two.** The plan
gate (PQ-12) caught the design error: `failTrackedPostAckStart` opens with `if
!resume { return c.failPostAckStart(...) }`, so a spawn's claim-phase failure
already took a different tail from a cold resume's. Modelling ownership as a
warm boolean would have merged them and broken spawn. `StartShape` is
therefore spawn / cold-resume / warm-reattach, and `DecideStartCleanup`'s
36-row table is its enumeration.

**The honest fake found a test that had been green for the wrong reason.**
Making `FakeThreadArtifactCollisionChecker.Quiesce` model the deletion it
performs immediately failed `TestResumeAmbiguousAckKeepsUnknownOccupied`. That
test set a session present, let the acknowledgement fail, and asserted the
thread stayed Unknown. Production quiesces the session *before* consulting its
presence, so the branch the test pinned is unreachable there: with a faithful
fake the path rolls back to the verified park instead. The test is now
`TestResumeAmbiguousAckRollsBackOnceItsSessionIsGone`, and
`TestResumeUnobservableSessionKeepsUnknownOccupied` pins the arm that really
does keep the record Unknown -- when the session cannot be observed at all.
This is the ARCH-MOCK failure mode in miniature: a fake laxer than production
hides the bug it stands in for.

**Mutation sweep, 8 of 8 killed as named** (apply-asserted, named failures
only, tree verified identical to the pre-sweep snapshot):
the rule always quiescing; spawn folded into cold resume; routes 5-6 ignoring
the shape; `AbortStarted` trusting the relayed `StartResult`; routes 1-4
inverting the spawn branch; presence assumed absent; the fake's quiesce
logging only; and the fake's quiesce modelling one of its two effects.

Two of those needed a second attempt, and the reason is worth recording: the
first `AbortStarted` mutation SURVIVED because every route-6 test relayed the
record couch itself built, so `Warm` was correct by accident and the guard was
invisible. `TestAbortStartedReadsOwnershipFromTheRegistryNotTheCaller` clears
the field to make the registry the only source that can answer. A first sweep
attempt also produced a false kill, where the only failing test was one the
sandbox fails anyway (`pty`); the row was re-run against a non-pty filter.

**Advisory findings carried to the close review:** PQ-6 (three unstated
non-goals), PQ-7 (taken: the shared retire helper KEEPS `Detach`'s `ctx.Err()`
interrupt, and cleanup passes `context.WithoutCancel` instead), PQ-10 (stale
task prose), PQ-11 (taken: the SIGKILL reaches only the helper's process
group, which the zellij server and its agent predate).

### 2026-09-11 — three claims in the Revisions above are wrong (PQ-5, BR-8)

The plan gate raised these as PQ-5 and I left them undisposed for three rounds
while fixing the code they described. The close review raised them again. They
are corrected here rather than edited in place, per the append-don't-overwrite
rule:

- **"`StartResult` carries `Warm`"** — it does not, and deliberately. Ownership
  is read from `ActorRecord.Shape`, couch's own registry record. A struct the
  caller relays back has a zero value, and that value must not be able to mean
  "you may delete this session".
- **"`quiescePostAckStart` is the only caller of `Artifacts.Quiesce`"** — it is
  one of two. `ArchiveThread` is the other, and it is deliberate and unchanged:
  archiving a thread means removing it, so ending its session is the point.
- **"a 24-row exhaustive table"** — 36 rows shipped. The input is three-valued
  (spawn / cold resume / warm reattach), not a warm boolean; that was PQ-12.

The close review's own findings are dispositioned in the implementation Log
below.

### 2026-09-11 — close review fixes (verdict FIX-THEN-SHIP)

**BR-5, the one real defect (Important).** A retire that failed returned
immediately, leaving an `IncarnationLive` behind a helper that was already
dead -- the stale state `pair#171` names, reached from an ordinary failure
path. The reviewer probe-confirmed it on warm route 5. Cleanup now falls
through to the recoverable disposition on any retire failure.

Its first test passed without exercising anything: the presence hook counted
`awaitResumeRegistration`'s own poll, so the session read absent at decision
time and cleanup chose mark-unknown, never attempting a retire. The test now
counts all three reads in order and asserts both that it reached the third and
that the returned error is the retire's own -- so it cannot pass without
entering the branch it is named for.

**BR-6 (Important).** `StartCleanup.Quiesce` was never read: production asked
`shape.OwnsSession()` at three call sites, so forcing the field true left the
whole seam suite green. The two halves are answered at different moments -- the
session is ended first, and its absence afterwards is an input to the record's
disposition -- so they are now two functions. `OwnsSession` is the named
authority, consumed once by `quiescePostAckStart`, and `DecideStartCleanup`
returns a `DurableAction`.

**BR-7, BR-8 (Important, both documentation).** The plan's Core-concepts tables
named `applyStartCleanup`, which the first implementation had not built, and
filed an IO helper under Pure entities. Both corrected, and
`applyStartCleanup` now exists as the single shell both entry points use. The
issue's stale Revisions claims are corrected above.

**Minors.** `AbortStarted` no longer labels every non-warm start a cold resume
-- `ActorRecord.Shape` records which it was, and the identity loop supplies it
without a second registry scan. Route 3's warm assertion now requires
`unusable/session-gone` rather than merely "not detached". The unreachable
retire arm at claim phase and the no-op `context.WithoutCancel(Background())`
are gone.

**Mutation sweep after the fixes, 6 of 6 killed as named** (apply-asserted,
tree verified identical to the pre-sweep snapshot): `OwnsSession` always true;
an unrecognised shape treated as owning; the retire fallback removed; the
quiesce ignoring ownership; `AbortStarted` trusting the relayed shape; a warm
start never labelled warm. Two of these survived their first run -- the
fallback row for the reason above, and the `AbortStarted` row because the test
relayed an empty shape, which `OwnsSession` already answers no to. It now
relays `StartColdResume`, the value a caller would plausibly fill in.

### 2026-09-11 — close review round 2

**BR-12, a regression my own refactor introduced (Important).** The
cold-resume tail that `applyStartCleanup` replaced joined the
session-observation error into its return, so an operator could see WHY a
thread was left occupied. `observeSessionPresence` folded that into a bare
`PresenceUnobserved` and dropped it. It now returns `(SessionPresence, error)`
and the shell joins it. The DECISION is unchanged -- unobserved is still never
treated as absent -- only the diagnostic is restored, pinned by
`TestCleanupSurfacesWhyItCouldNotObserveTheSession` and its mutation.

Worth recording as a pattern: consolidating several tails into one shell is a
deletion of behaviour unless each tail's outputs are enumerated first. I
checked what the tails DECIDED and missed what they REPORTED.

**BR-13, a rule rather than a site (Important).** Eight passages across five
files still described design decisions this issue had reversed:
`StartResult.Warm`, a destructive zero value, `context.WithoutCancel`, and the
decider answering whether to quiesce. Two of them were the same paragraph
duplicated. All corrected in one sweep, including `atlas/couch.md` and this
plan's prose.

The `WithoutCancel` one had substance behind it: the code said
`context.WithoutCancel(context.Background())`, a no-op wrapper, because neither
cleanup function receives the caller's context in the first place. It is plain
`context.Background()` now, and `Detach` keeps its own `ctx.Err()` interrupt.

**Suite:** unsandboxed `make test` exit 0, 197 packages ok.
