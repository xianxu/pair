---
id: 000229
status: open
deps: []
github_issue:
target: workbench-latency
created: 2026-09-11
updated: 2026-09-11
estimate_hours:
---

# Detaching one thread is slow

(The slug names the lead suspect as it stood at filing: the post-detach
inventory refresh. It is unmeasured, and the Plan measures before choosing.)

## Problem

**Detaching one thread still feels slow after #228 made reattach fast.** The
operator said so on 2026-09-11, right after confirming #228's reattach speedup
("way much faster"). Nothing tracks single-detach latency. `#205` covers
running *batches* of park and detach in parallel, which is a different
question.

What one detach does today, from the code (`couchcore/detach.go`):

1. `PairSession` before signalling. It is liveness only since #228: two
   `list-sessions` calls and no `list-clients`.
2. `SignalGroup(pid, SIGTERM)` to the pair client's process group. Pair
   installs no SIGTERM handler, so the default disposition should end it at
   once.
3. `awaitExactProcessExit`: a 10 ms poll of `Exists` + `Identity` on the exact
   pid, bounded at 15 s.
4. `PairSession` after, again liveness only.
5. `RetireIncarnation`, a store write with a bounded revision-conflict retry.

Then the console runs `finishOperation` and `requestMenuRefresh`.

**Lead suspect, unmeasured: the operator waits on the inventory refresh, not on
`Detach`.** The detach hotkey lands the operator in the switcher with a
"detaching …" spinner. Once the operation completes:

- the menu marks its projection pending (`ProjectionAfterGeneration`, rendered
  as "refresh pending");
- the row changes only when the next full inventory refresh lands;
- that refresh runs `DetachedSessions` over every detach candidate, which is
  one `list-clients` per candidate session (#228 site 7);
- `list-clients` costs about 250 ms against a real detached pair session (#228
  and #206 Logs).

With 10 detached threads that is about 2.5 s after `Detach` itself has already
finished. The thread that just changed is the only row whose state is new.

Other candidates to rule in or out by measurement, not by argument:

- the pair process taking longer than expected to exit after SIGTERM;
- `Identity` shelling out per poll;
- the single-worker operation queue holding the detach behind other work;
- the console's `onExit` handling of the pty child.

## Spec

**Measure before deciding** (#206's discipline: a probe replaces the analogy).
Then fix what dominates.

- The metric is what the operator experiences: from the detach keypress to the
  switcher showing that thread as detached. Break it down by phase: dispatch
  and queue wait, `Detach` (and within it, the SIGTERM-to-exit wait), the
  store write, operation completion, and the refresh landing.
- If the refresh dominates, the candidate fix is to update the changed row
  without waiting for a full re-scan. That could be a targeted refresh of just
  the detached thread's session (the `SnapshotSessionsContext` form #228
  added), or an optimistic row update the next refresh confirms. Either keeps
  the full refresh's authority. Planning decides which, and must state how a
  refresh that disagrees with the optimistic row wins.
- **Detach's safety properties are not traded for speed.**
  - SIGTERM stays the only signal.
  - The before and after session proofs stay.
  - A client that ignores SIGTERM still fails the detach, not escalates it.

## Done when

- A single detach's keypress-to-row-updated latency is measured on the real
  stack with a per-phase breakdown and the co-tenancy recorded (agent count,
  live and detached session counts). The procedure is reproducible (a probe,
  or a timing trace with a named command).
- The dominant phase is fixed, or its cost is justified in the Log.
- Where the cost is zellij work, it is asserted as a count, independent of the
  session count where it can be (the `workbench-latency` Tier 1 promise).
- Detach's safety tests pass unmodified. That includes the SIGTERM-only rule
  and the before and after proofs.
- The operator confirms a detach feels fast.

## Plan

- [ ] Measure one detach end to end, with a per-phase breakdown. Record it in
      the Log with co-tenancy.
- [ ] Decide the fix from the measurement, as a durable plan if it crosses
      packages.
- [ ] Implement with count-based tests where the cost is zellij work.
- [ ] Re-measure, and operator smoke.

## Log

### 2026-09-11

Operator request after #228 landed: *"can you make a task to improve detach
performance?"* The same session had just established that the switcher's
post-mutation refresh is O(detach candidates) `list-clients` calls. That makes
it the lead suspect for the wait the operator sees, but it is unmeasured.
