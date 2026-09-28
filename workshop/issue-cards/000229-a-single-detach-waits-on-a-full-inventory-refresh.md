---
id: '000229'
status: wontfix
created: 2026-09-11
updated: 2026-09-12
---

# Detaching one thread is slow

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
