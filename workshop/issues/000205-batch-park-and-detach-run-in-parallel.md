---
id: 000205
status: open
deps: ["#214"]
github_issue:
created: 2026-09-06
updated: 2026-09-06
estimate_hours:
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

## Spec

**A bounded worker pool drains `operationQueue`, not a single worker.**

- Bounded, not unbounded. The measured hazard on this host is process-spawn
  contention (`#203`): each operation spawns `zellij action` subprocesses, and
  an unbounded fan-out over a large batch recreates the load shape that took the
  same call from 17.6 ms to 145 ms. A small pool captures nearly all the
  overlap without the storm.
- The safety invariant must be **established by `#214`**, not inherited: key the
  queue by thread + operation class so one thread admits one launch-producing
  operation. Only then does removing the single worker leave the cross-thread
  ordering as the sole thing given up.
- Results still land on the console goroutine through `q.results`, so completion
  handling and `c.mu` discipline are unchanged.

### The interleaving policy has to be written down (`ARCH-ORDER`, ariadne#215)

This is durable state plus events arriving from outside, and the cells below
have no modal answer — the plan must state each rather than let the
implementation sample one:

- The operator presses a key, switches, or opens the menu **mid-batch** —
  queue, preempt, ignore?
- **Three of eight fail.** Is the batch partial, retried, or rolled back? What
  does the operator see?
- A thread **exits on its own** while its park is in flight.
- The operator **quits couch** mid-batch.

## Done when

- A batch park/detach of N threads completes in materially less wall-clock than
  N sequential operations; measured, with the working-agent count recorded
  (per `workshop/targets/workbench-latency.md`, a timing without its co-tenancy
  is not a measurement).
- Concurrency is bounded; the bound is stated with its reason, not tuned by feel.
- No two operations ever run on one thread — asserted by a test, not inherited
  from the queue's current shape.
- Every interleaving cell above has a stated answer in the issue and a test.
- A failing operation inside a batch leaves the other threads' outcomes intact
  and the failure visible.
- `#204`'s suite gains a counted invariant for the batch path — batch park of N
  threads issues O(N) store writes, not O(N²).

## Plan

- [ ] Decide the interleaving policy cells above; record them in `## Spec`.
- [ ] Replace the single `q.Run` with a bounded pool; keep dedup and result
      delivery unchanged.
- [ ] Test: N threads, one in-flight op each, no cross-thread ordering assumed.
- [ ] Test the failure and mid-batch-input cells.
- [ ] Measure N-batch wall-clock before/after, recording agent count.
- [ ] Add the counted invariant to `#204`.

## Log

### 2026-09-06

Operator request, split from `#206` (auto-reattach at startup) at their
instruction — the two share a motivation but not a risk profile. This half is
well-founded: the store's concurrency model, the dedup invariant and the park
future all already exist, so the change is small and lands on seams built for it.

`#206` is the one that needs a measurement before its approach is chosen.
