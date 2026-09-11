---
id: 000206
status: blocked
deps: [000228]
github_issue:
created: 2026-09-06
updated: 2026-09-10
estimate_hours:
started: 2026-09-10T20:16:19-07:00
---

# reattach detached threads automatically at startup

## Problem

Starting couch does not bring back the threads that are still running behind
client-less zellij sessions. A detached thread's agent is alive — `couchcmd/run.go:641`
describes the state as *"detached (no client attached; the agent is still
running)"* — but the operator has to go find each one in the switcher and
reattach it by hand.

With the session populations this fleet actually runs (10–11 live sessions
observed 2026-09-06), that is the first thing the operator does every time, and
it is pure ceremony: the state needed to do it automatically is already in the
ThreadStore before anything launches.

**The feature is the auto-reattach. How it loads is a separate decision**, and
this issue deliberately does not pre-commit to one — see Spec.

## Spec

### The sequence (operator, 2026-09-10)

1. **Start the cwd thread exactly as today.** `StartInteractive` resolves the thread
   for the directory `couch` was run in — resume or new (`couchcmd/run.go:319`) — and
   `dispatchInitialAttach` attaches it (`:406`). The operator lands in it immediately.
   **Unchanged.**
2. **Then reattach every other *running* thread** — those whose agent is still alive
   behind a client-less zellij session (`ThreadDetached`, *"no client attached; the
   agent is still running"*).

**Parked threads are not touched.** A parked thread's agent was torn down; bringing it
back is a *resume* that starts an agent, which is a different and costlier action than
reattaching one that is already running. Startup reattaches warm threads only.

**Skip the cwd thread in step 2.** If the cwd thread was itself detached, step 1
already attached it. Step 2 must exclude it explicitly rather than rely on `#214`'s
per-thread guard to refuse the second launch.

### What the ordering settles

Because the operator is already working in the cwd thread after step 1, the step-2
reattaches are **off the critical path** — nobody is waiting on them. That largely
decides the strategy question below in favour of **B (sequential, in the
background)**: parallel loading buys speed nobody is waiting for, at the cost of the
spawn contention `#203` measured. The measurement in Plan step 1 still runs, but its
job shrinks to confirming sequential is fast enough, not choosing between strategies.

The greyed-row and queue-jump parts of B still apply: switching to a thread that has
not attached yet should move it to the front of the queue.

### Strategy options (kept for the record; see above)


**At startup, couch reattaches every detached thread without being asked.**

The load strategy is the open question, and the plan chooses it **after** the
measurement in step 1, not before. Three candidates:

### A. Parallel

All reattaches at once. The obvious approach and possibly fine.

The caution on record, with its limits stated honestly: `#203` measured that
concurrent agent activity took a `zellij action` round-trip from **17.6 ms to
145 ms (max 467 ms)** — an 8× degradation — and warned that parallel launches
recreate that load shape. **That analogy is not proportionate and should not be
treated as settled evidence.** The load that produced the 8× was ~60 runnable
threads: five agents each running `go test ./...` at `GOMAXPROCS=12`, spawning
hundreds of short-lived compile processes. A parallel reattach of eight threads
spawns roughly two processes each, once. Different magnitude, different shape.

What is *not* known is what a `zellij attach` costs on the server side — each
one drives a pane restore and replay, which may be heavier than its process
count suggests. That is the thing to measure.

### B. Sequential with a progressive switcher (operator's design)

Pre-allocate the rows immediately, greyed, and load one thread at a time in the
background; a row becomes live when its session attaches.

This is strong for a reason worth stating: **it optimises time-to-first-useful-thread
rather than total wall-clock, which is the metric the operator actually
experiences.** It also needs no new state — the ThreadStore already knows every
thread's address, label and state before anything launches, which is exactly
what `ActionableThreadInventoryContext` renders for the switcher today, and the
status row already carries `StatusActor{Label, Thread, Active, Bell}`. "Pending"
is one more field, not a new mechanism. And it sidesteps the contention question
entirely rather than answering it.

**With one addition that makes it clearly worth building: a switch to a pending
row jumps the load queue.** Promote that thread to the front. The order becomes
operator-driven, so the thread they want is always loading first and the rest
can finish whenever — which makes total wall-clock nearly irrelevant.

### C. Both

Bounded-parallel loading behind the progressive UI. Only worth the complexity if
step 1 shows parallelism buys real time *and* the operator still waits.

### The cell that exists in every variant (`ARCH-ORDER`, ariadne#215)

**What does a switch to a not-yet-attached thread do?** Queue the switch and land
on arrival, jump the queue, or refuse. Queue-and-jump is almost certainly right;
it must be written down rather than sampled per call site. Likewise: a thread
that fails to reattach, and the operator quitting mid-load.

## Done when

- Starting `couch` in a directory attaches that directory's thread first, exactly as
  today, and the operator can type in it before any other thread has attached.
- Every other running (detached) thread is then reattached with no operator action.
- Parked threads are not resumed at startup.
- The cwd thread is attached once, never twice, when it was itself detached.
- Step 1's measurement is recorded in `## Log` — single reattach vs N concurrent,
  with wall-clock and the resulting `zellij action` latency — and the chosen
  strategy cites it.
- A thread that fails to reattach is visible as failed, and does not block the
  others.
- The pending-row interleaving cell has a stated answer and a test.
- Startup does not make the workbench unusable while it runs: `zellij action`
  latency during startup is measured, not assumed (`workshop/targets/workbench-latency.md`).
- If B or C: the switcher shows every known thread immediately, and a switch to a
  pending row behaves as specified.

## Plan

- [ ] **Measure first.** Time one warm reattach, then N concurrent, on a quiet
      host: total wall-clock and `zellij action` latency during the burst. This
      decides A vs B vs C and nothing should be built before it.
- [ ] Decide the interleaving cells; record them in `## Spec`.
- [ ] Implement the chosen strategy.
- [ ] Test the failure path and the pending-row switch.
- [ ] Re-measure startup end to end, recording the agent count.

## Log

### 2026-09-06

Operator request, split from `#205` at their instruction. The two share a
motivation — startup and shutdown feel slow — but not a risk profile: `#205`
lands on seams already built for concurrency, while this one adds a new
behaviour whose approach is genuinely undecided.

Two corrections recorded so neither is re-litigated from memory:

**The argument against parallel was weaker than it was presented.** It was
reasoning by analogy from `#203`'s build storm, which is a different magnitude
and a different shape (hundreds of short-lived compile processes vs ~2 per
thread, once). Transferring that conclusion was the same error as the retracted
burner probe in `#203` — testing or citing one load shape and generalising. The
measurement in Plan step 1 replaces the analogy.

**The reattach path is `#196`'s.** The mouse-mode belief bug — the fourth
recurrence of one defect — lived exactly here, and was fixed on 2026-09-06.
Whatever strategy is chosen, `TestAReattachedChildKeepsItsTrackingMode` must
still pass unmodified, and a parallel variant should be checked against it
specifically: that fix reasons about a fresh `Screen` for a still-running child,
and N of those arriving at once is a case it was not written against.

## Revisions

### 2026-09-10 — the startup sequence is specified

**Reason.** The operator clarified the intended behaviour: *"normal start up of a
thread in cwd, and then reattach to all running/live threads."* The original Spec said
only "reattach every detached thread at startup" and left both the ordering and the
cold/warm distinction implicit.

**Delta.** Added the two-step sequence (cwd thread first via the unchanged
`StartInteractive` path, then every other detached thread), made explicit that parked
threads are not resumed, and required the cwd thread to be excluded from step 2. The
cwd-first ordering takes the background reattaches off the critical path, which
resolves the A/B/C strategy question toward B; the options are kept below for the
record.

### 2026-09-10: Plan step 1, the measurement (it moves the problem)

Claimed. A read-only exploration mapped the startup and reattach paths. Three
structural facts shape any design:

- **One operation at a time.** The switcher holds a single in-flight slot and
  silently refuses operator dispatches while it is taken (`menu.go`
  `dispatchMenuOperation`).
- **One worker.** The operation queue is serial (`operation_queue.go`).
- **Resume takes focus.** Every successful resume force-switches
  (`finishOperation`, `completed.origin.Operation == "resume"`).

A background reattach can therefore neither hold the in-flight slot nor steal
focus. Also found: `#214`'s per-thread guard was never built.

**Procedure.** `PAIR_PROBE_N=8 go run ./cmd/probes/reattachcost` (new; sandbox
off for zellij). The probe:

- creates N+1 detached sessions under the repo's `zellij/config.kdl`, with a
  pair-shaped layout (framed agent pane that has printed 120 wide rows, over a
  fixed 12-row borderless draft) at 200x50;
- times real `zellij attach` clients until the session's own marker renders;
- samples `zellij action query-tab-names` against a never-attached control
  session throughout.

**Co-tenancy:** load 2.19; 7 agent processes; 26 zellij sessions listed, of
which 8 are live pair sessions (5 detached); 12 cores; zellij 0.45.1.

| phase (N=8) | per thread | total | action p50 / p95 / max |
|---|---|---|---|
| quiet baseline | — | — | 19 / 26 / 47 ms |
| one attach (x3) | 52–55 ms | — | 19 / 21 / 21 ms |
| 8 sequential attaches | 51–57 ms | 447 ms | 19 / 21 / 21 ms |
| 8 concurrent attaches | 69–120 ms | 121 ms | 25 ms (1 sample) |
| **8 couch-shaped** (5 snapshots + attach) | **5.3–5.7 s** | **44 s** | 19 / 21 / 50 ms (n=626) |

**What it says.**

1. **`zellij attach` is cheap:** about 55 ms. Concurrency barely costs anything
   and does not move `zellij action` latency. The analogy with #203 was wrong
   in the direction the Log already suspected.
2. **The cost is couch's own proof work.** One couch reattach takes 5 session
   snapshots, counted from the code:
   - couchcore `ResumeContext`'s `DetachedSessions`, and `confirmStillDetached`;
   - `pair resume`'s launcher, at `createflow.go`'s loop top and in `runOnce`;
   - the console's inventory refresh on completion.

   Each snapshot is 2 `list-sessions` calls plus one `list-clients` for *every*
   live pair session on the host.
3. **`list-clients` is the expensive call, and only against real sessions.**
   Timed singly:
   - about 190–350 ms against a real **detached** pair session (one took 971 ms);
   - about 38 ms against an attached one;
   - about 53 ms against a fresh, empty detached session;
   - `query-tab-names` against the same real detached session: about 55 ms.

   A startup pass is therefore **O(N × S)** calls at about 250 ms each for
   detached S. With 10 threads all detached, that estimates to about 80 s for
   the pass, and about 13 s for the first reattach (the one a queue-jump
   waits on).
4. **The sequential pass does not hurt interactive latency** (p95 21 ms, max
   50 ms), so strategy B is *safe*. It is *slow in wall-clock*, and the
   queue-jump would be slow too, because each reattach is dominated by proof
   snapshots that ask every session about its clients.

This is the target's defect shape exactly (`workbench-latency`: "a cost that
scales with the wrong thing"). Proving that ONE thread is detached costs
O(all sessions). It also taxes every manual reattach today: about 5 s here.
`zellij list-sessions` in 0.45.1 carries no client information, so the lever is
how many sessions each snapshot asks, and how many snapshots each reattach
takes.

### 2026-09-10 — blocked on #228, the reattach cost, at the operator's direction

**Reason.** Plan step 1's measurement (Log, 2026-09-10) showed that a reattach
costs about 5.5 s, dominated by proof snapshots that ask every live pair session
for its clients, not by `zellij attach` (about 55 ms). As specified, the
background pass would take about 80 s for 10 threads, and a queue-jumped thread
about 13 s. The operator chose to fix the cost first, as its own issue.

**Delta.** `deps: [pair#228]`, status `blocked`. The strategy is unchanged: B,
sequential in the background. The measurement confirms it is safe for
interactive latency (p95 21 ms during the pass). #228 makes it fast. The probe
(`cmd/probes/reattachcost`) ships with #228, and #206's end-to-end re-measure
uses it.
