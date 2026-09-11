---
id: 000230
status: open
deps: []
github_issue:
created: 2026-09-11
updated: 2026-09-11
estimate_hours:
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
- **The thread returns to detached.** Once the helper is proven dead and the
  session proven still present, the start claim rolls back, so the thread has
  no incarnation and reads as Detached again, resumable by hand.
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

## Plan

- [ ] Red: the warm post-ack failure table at the fake seam.
- [ ] Carry `Warm` into the post-ack failure path. For warm: end the helper,
      skip the quiesce, then roll back or mark unknown on observed presence.
- [ ] Pin the cold path's quiesce in the same table.
- [ ] Mutation check; full suite.

## Log

### 2026-09-11

Found while designing #206's background reattach pass. The pass runs warm
reattaches behind the operator's back, and the question was what quitting
mid-reattach does to one. The answer was that it deletes the session.
