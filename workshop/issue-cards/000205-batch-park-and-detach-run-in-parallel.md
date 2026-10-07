---
id: 000205
status: working
created: 2026-09-06
updated: 2026-10-04
estimate_hours: 3.34
github_issue:
started: 2026-10-04T17:59:49-07:00
claimant:
    operator: T
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
---

# batch park and detach run in parallel

## Problem

Parking or detaching several threads from the switcher runs them one at a time.
Each operation is dominated by waiting — `zellij action` round-trips measured at
**17.6 ms on a quiet host and 145 ms (max 467 ms) under load** — so a batch of
eight spends most of its wall-clock blocked on IPC that could overlap.

This is shutdown latency the operator sits through, and nothing about it is
CPU-bound.

### The architecture already supports it

Three facts found while scoping, all of which say the serialization is
incidental rather than load-bearing:

1. **`ThreadStore` is CAS-based, not lock-based.**
   `UpdateExistingThread(address, expectedRevision, …)` takes an expected
   revision per record, and `CommitStartClaim`'s comment states the model
   outright: *"the revision CAS is what makes the decision still true at the
   moment of the write."* Distinct threads are distinct records, so parallel
   per-thread operations do not contend by construction.
2. ~~**The dedup invariant that actually matters already exists.**~~
   **WRONG — corrected 2026-09-08, see `#214`.** The key is
   `fmt.Sprintf("menu\x00%d\x00%s", effect.Attempt, effect.Operation)`
   (`console.go:1509`): **attempt + operation name, with no thread address**. So
   the same operation cannot be double-submitted, but *different* operations on
   one thread are different keys and both admit. There is no "one operation in
   flight per thread" invariant — **the single worker is the only thing
   serialising them**, which is precisely what this issue proposes to remove.
   `#214` records a real incident where `resume` racing `relaunch` produced three
   launches in 32 seconds and left the thread unresumable. **Fix `#214` first**;
   this issue now depends on it.
3. **Park already has a future seam.** `parkworker.go` carries `parkFuture`,
   `Await`, and an admission limit (`ErrParkWorkerOverloaded`) — the shape this
   issue needs, already built.

The serialization is **one line**: `console.go:538` starts exactly one
`c.operationQueue.Run(c.stop)` goroutine, and `Run` processes `request.run()`
to completion before taking the next.

### Expectation to set correctly

The queue already runs off the console goroutine, so a batch does **not** freeze
the UI today — it trickles. The win here is batch wall-clock, not
responsiveness. That is a smaller prize than it feels like, and it is the honest
reason this is a nice-to-have rather than a fix.
