---
id: 000206
status: working
deps: [000228]
github_issue:
created: 2026-09-06
updated: 2026-09-11
estimate_hours: 3.13
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

## Estimate

Derived after the plan cleared its fresh-context review (verdict rework, then
reworked: 2 Critical and 9 Important folded in). Two milestones, three
packages, and one new state machine.

Sized against two closes in this repo: `#228` (est 1.98 / actual 1.26, same
couchcore snapshot surface) and `#230` (est 1.25 / actual **2.42**, where four
boundary-review rounds were the whole overrun). The review items below are
written up rather than down because of #230's evidence, not despite it.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
design-buffer: 0.15
item: smaller-go-module      design=0.05 impl=0.12
item: smaller-go-module      design=0.08 impl=0.20
item: smaller-go-module      design=0.08 impl=0.20
item: atlas-docs             design=0.03 impl=0.04
item: milestone-review       design=0.00 impl=0.20
item: smaller-go-module      design=0.05 impl=0.16
item: greenfield-go-module   design=0.12 impl=0.24
item: smaller-go-module      design=0.08 impl=0.20
item: tui-screen             design=0.15 impl=0.28
item: smaller-go-module      design=0.03 impl=0.10
item: smaller-go-module      design=0.02 impl=0.06
item: smaller-go-module      design=0.03 impl=0.08
item: atlas-docs             design=0.05 impl=0.06
item: milestone-review       design=0.00 impl=0.30
total: 3.13
```

**M1** (first five items) — the `ProjectDetachedSessions` claim-count
hardening M1 depends on; the red count/equivalence tests; the `startupAsks`
narrowing itself; atlas; one milestone review.

**M2** (the rest) — `warm-only` resume; the pure `ReattachPass` reducer
(greenfield, its own file, all transitions table-tested); routing it through
`ReduceMenu`; the console wiring (`tui-screen`: goroutines, the operation
queue, the focus rule, and the background-attach guard that must not seed
focus); rendering; `couchcmd` arming; the `COUCH_TRACE` events; atlas plus
the operator-assisted measurement; one close review.

The M2 close review is 0.30 rather than 0.20: it spans a state machine across
three packages, and #230's four rounds are the measured reason to expect more
than one.

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.*

## Plan

- [x] **Measure first.** Time one warm reattach, then N concurrent, on a quiet
      host: total wall-clock and `zellij action` latency during the burst. This
      decides A vs B vs C and nothing should be built before it. (Log
      2026-09-10; it moved the problem to #228, now merged.)
- [x] Decide the interleaving cells; recorded in the durable plan's
      "Decisions" and "The pass's transitions" (`workshop/plans/000206-*-plan.md`).
- [x] M1 — startup proves only the threads its readers consume (`startupAsks`),
      and the duplicate-name rule counts each thread once over the union of
      index files it reads.
- [x] M2 — the background reattach pass: `warm-only` resume, the pure
      `ReattachPass` in `MenuState`, the pass view applied at the row lookups,
      console wiring that never takes focus, rendering, arming, the trace, the
      operator-assisted measurement, and the smoke.

## Log



- 2026-09-12: closed M2 — M2 background reattach pass: warm-only resume, the pure pass in MenuState, the pass view at every row lookup, console wiring that never takes focus, greyed unclickable placeholders, start-only arming, COUCH_TRACE. Tests: cells 1-13 plus generated-sequence routing invariants; M2 mutation sweep 38 of 38 killed; Task 11 sweep 23 of 23; unsandboxed make test exit 0 across 197 packages. Operator smoke live on the real stack, including a quit mid-pass; it found the last placeholder still spinning, fixed with a test red before and green after. Measured from the operator traced restart: first frame 0.72 s, a 5-thread pass in 2.34 s (median attempt 302 ms), zellij action at most 35 ms during startup against a quiet max of 36 ms. Round 4 fixes in 4d331487: BR-13 README paragraph on the pass; BR-14 plan points at menu_reattach.go instead of restating it, lesson extended, re-paste guarded. Actual 3.80 h = measured 8.86 h cumulative minus the measured 5.06 h M1 closed at.; review verdict: SHIP
- 2026-09-11: closed M1 — startupAsks resolves only the union its readers consume; gates ResolveEstablished and the zellij query. Counted at the seam: 3 detach candidates, 1 binding resolution at 2 and 12 other threads. Equivalence test now covers all 7 plan rows and all 4 readers (conflicts compared by address; layout3 for a valid conflict, layout1 for unreadable); a count test kills the dropped-scope-arm mutation the equivalence test structurally cannot. Duplicate-name rule counts each thread once at its effective binding over the union of index files read, pinned by a two-scope legacy seam test. Mutations 13/13 killed as named. Unsandboxed make test exit 0, 197 packages. OPERATOR SMOKE: startup ~1.5x faster on the real stack. Actual 5.06h = measured #206 share since claim, incl. measure-first probe and 4 plan-gate rounds.; review verdict: SHIP
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

### 2026-09-11 — unblocked: #228 is done (PR #123)

**Reason.** #228 made one reattach's zellij cost independent of the host's
session count. A warm reattach now asks exactly 2 sessions for their clients,
whatever S is. `pair resume`'s launcher takes liveness snapshots only, and
skips the name probe for a live name.

**What it changes here.**
- **Per reattach.** The probe measured 272–282 ms per thread, against
  7.7–8.7 s before, under one co-tenancy (N=8, #228 Log). On real detached
  sessions, whose `list-clients` takes about 250 ms, expect roughly 0.7 s.
- **The background pass (strategy B)** for 10 detached threads becomes roughly
  7 s, not about 80 s. A queue-jumped thread waits about one reattach, not
  about 13 s.
- **Startup's blocking inventory is now O(C), not O(S).**
  `StartInteractive` (`couchcore/startup.go`) runs
  `ActionableThreadInventoryContext` before the first frame. Since #228 that
  asks only the detach candidates' sessions, but it still asks every
  candidate. With 10 detached threads that is about 2.5 s before the cwd
  thread attaches. The operator felt this as "the initial startup attach is
  still slow" after #228.

**For the plan.** Startup's blocking step should prove only what the first
frame needs, the cwd thread, and hand every other candidate to the background
pass. One constraint: the mixed-layout guard in `StartInteractive` reads these
same rows, and its comment says they are startup's only session enumeration.
Narrowing the blocking step has to say where that guard's rows come from.

**Delta.** Status `working`. `deps: [000228]` stays, now satisfied.

### 2026-09-11 — M1 implemented (startup narrowing)

**Startup proves only what its readers consume.** `startupAsks` is the union of
the cwd filter (the two selectors and the one-thread-per-path guards) and the
layout filter (`ResolveLayoutConflicts`, which reads rows at ANY path whose
layout differs). A candidate outside both keeps `ProofUnresolved` and
classifies `unknown` -- a row no reader here can act on. The rows never leave
`StartInteractive`, so unasked state cannot reach the switcher.

**The filter is applied before `ResolveEstablished`, not just before the zellij
query.** Each resolution reads that thread's own ledger, so filtering only the
`list-clients` calls would have left time-to-first-frame growing with the
store, looking fixed. A mutation that moves the filter after the resolution is
in the sweep.

**Counted at the seam:** 3 detach candidates and 1 binding resolution,
identical with 2 other threads or 12. Candidates, not calls -- the fan-out is
batched into one call, so a call count cannot see it grow.

**The prerequisite (Task 1):** `ProjectDetachedSessions`' duplicate-name rule
counted claims across the bindings passed in, so a narrowed ask made a
contested session look unique and startup would resume a thread whose session
belongs to something else. It now counts over the scope's whole index, per
thread at its effective (newest) binding. The fake goes through the same
function rather than its own map (`ARCH-MOCK`).

**Two fixture bugs that made tests pass for the wrong reason**, both the same
shape as #230's:
- the fixture used a hand-written repo-scope key that never matched what
  `StartInteractive` resolves, so the narrowing predicate matched nothing at
  the cwd and the count test passed on a coincidence (the unnarrowed count
  happened to equal the expected number at that thread count);
- without a native binding every row classified `binding-lost`, so the
  detached proof was never exercised.

**Mutations, 7 of 7 killed as named** across Task 1 and the narrowing: claims
counted over the ask rather than the index; every index entry counted rather
than each thread once; contested names admitted; `ask` ignored; the layout arm
dropped; the cwd arm dropped; `ask` applied after `ResolveEstablished`. Two of
them survived a first attempt -- the index-entry one needed the "moved off a
name" case, and the harness itself reported a false SURVIVED because a
pipeline's exit status came from `head` rather than `grep`.

**Suite:** unsandboxed `make test` exit 0, 197 packages.

### 2026-09-11 — the Plan's boundaries, and the plan gate converged

The original Plan rows predated the design, so they are restated as the two
review boundaries the durable plan actually has: **M1** (startup narrowing, in
`couchcore`) and **M2** (the pass, across `couchtty` and `couchcmd`). The
measure-first and decide-the-cells rows are done and ticked, with pointers.

**Plan gate, four rounds.** Beyond the rounds already logged, round 3 found a
real bug in Task 1's shipped code -- per-scope claim counts were summed, so a
thread bound only by the legacy index file counted once per scope asked and
read session-gone (56 such bindings on the operator's host). Fixed by merging
over the union of reads; pinned at the seam by a two-scope test, because the
pure helper's own test survived reintroducing the bug.

**Estimate-quality: INFO, no refusal.** Its main note is that Task 12 is
under-slotted (the probe's sample mode, the 30 s sampler run, the smoke and the
Done-when conversation are more than an atlas slot) -- expect 0.1-0.2h over on
M2's close-out. It also notes M1's items were costed after M1 was built, so
they are retrospective; the ledger's actual will say how far off they were.

### 2026-09-11 — the Done-when deviation, decided by the operator

**Asked before M2, as the plan gate required.** Done-when says the switcher
shows every known thread "immediately". The plan shows them when the first
inventory lands -- before any background reattach starts, so no row ever waits
on one -- but not on the literal first frame, which still reads "thread
inventory unavailable" as it does today. The operator chose **first inventory
is fine**. No first-frame seeding and no "checking" row state; M2 builds as
planned.

### 2026-09-11 — M1 operator smoke

After `make install` (pair rebuilt 15:19, carrying #230 and M1), the operator
detached and restarted couch: **"start up seems faster, not 10x, maybe 1.5x."**

Read honestly, that says the proof work M1 removed -- a `list-clients` per
other detach candidate and a ledger read per resume-shaped record -- was about a
third of startup on this store. The counted invariant holds (3 candidates and 1
resolution whatever else the store holds), so what remains is NOT proportional
to the other threads: it is the cwd thread's own reattach (`ResumeContext`'s two
detached proofs, the spawn, the registration wait) plus fixed setup (git
resolution, `reconcileInterruptedStarts`, the supervisor lease). None of that is
measured yet. M2 Task 11's `COUCH_TRACE` is what would show where it goes, and
further startup work should start from that trace rather than from a guess.

**The operator also reported "detach became 10x faster though."** Nothing in M1
or #230 is on the detach success path: M1 changed startup's inventory and the
detached-proof's claim count, and #230 changed only post-acknowledgement
failure cleanup (its `retireDetachedIncarnation` extraction is behaviour-
identical). So this is recorded as UNEXPLAINED rather than credited. The likely
explanation was the gesture (`leave` never waits on the post-detach refresh),
but the operator confirmed **the same gesture both times**, so that is ruled
out.

**What was checked, and ruled out.** The `couch` shell function rebuilds from
the working tree on every launch (`go build -o bin/couch ./cmd/couch`), so each
smoke ran exactly its branch's code: #228's then, #228+#230+M1 now. Between
those two points only `couchcore` changed, and on every function the detach
touches or triggers:
- `Detach` itself only MOVED code into `retireDetachedIncarnation`
  (behaviour-identical; error text changed);
- the post-detach refresh passes `ask == nil`, so M1's filter never applies;
- `DetachedSessions` and `PairSession` make exactly the same zellij calls;
- no leftover reattachcost probe sessions (26 sessions, as #228 recorded).

**So there is no code cause I can find, and none is claimed.** The likeliest
explanation is zellij's own variance under load: #228 measured `list-clients`
against real detached sessions anywhere from 190 ms to 971 ms for the same call,
a 5x spread on one call, and a post-detach refresh chains several of them.
Earlier today the host ran the reattachcost probe, repeated test suites and
builds. A single number from each of two different-load moments cannot separate
code from co-tenancy.

**Settling it is #229's first plan step**, not a reason to detour #206: time a
single-thread detach end to end, several runs, with a per-phase breakdown and
co-tenancy recorded. This observation is carried there as input.

### 2026-09-11 — the operator's UX for pending threads

**Reason.** Mid-M2, the operator specified how threads look while the pass
brings them back:

> *"ideally, when we attempt to start a thread, we would add placeholder of it
> in the couch status bar, with a spinner: brain [spinner], while the thread is
> being started. during this state of being reattached, it's not clickable,
> and when user click on it, we won't switch to it as it's not yet ready. if
> user do search in switcher, same thing, that line is grayed, and not
> selectable."*

Asked how QUEUED threads (pending, not yet starting) should appear, they chose
**all pending, none selectable**: every thread the pass will reattach appears
at once, greyed, with a spinner on the one currently starting, and none can be
clicked or selected until it attaches.

**Delta.** This settles the Spec's open cell -- "what does a switch to a
not-yet-attached thread do?" -- as **refuse, visibly**: the row and the chip are
present but inert until ready. The Spec's strategy-B idea of jumping the queue
is dropped, and so is the plan's "adopt the loading row". Since #228 each
reattach takes about 0.3-0.7 s, so ten threads fill in within seconds, and
reordering them would buy little. Done-when's "a switch to a pending row
behaves as specified" now means *it is not selectable*. The plan's Decisions,
pass view, transitions and Tasks 6-12 are rewritten to match.

### 2026-09-11 — M2 Tasks 5-7: warm-only, the pure pass, and its routing

**Task 5 (`01de3aa7`).** A warm-only resume refuses a verified park before the
binding is resolved, and a thread with no detached session before
`CommitStartClaim` (`ResumeNotDetached`). `warm-only` is an Implicit arg the CLI
refuses as unknown. It is pinned through the operation table as well as
directly: a direct-call test survived the dispatcher dropping the argument.

**Task 6: the pure pass** (`couchtty/menu_reattach.go`).
- Four explicit phases: Idle, Armed, Running, and Done. Done is separate from
  Idle so a second arm cannot re-seed.
- It seeds from detached and resume-shaped unknown rows, most recent first.
- It advances one attempt at a time, and holds while the operator has an
  operation in flight (cell 10).
- `finishReattach` resolves an attempt as attached, skipped or failed.
- `passViewOf` is the only function that knows what the pass means for a row.
- `pendingPlaceholders` drives the status bar.

**Task 7: routed through `ReduceMenu`, with the view applied at the lookups.**
- **The lookups.** Every inventory read goes through `menuRows`, `menuThread`
  or `visibleMenuRows`. An attached thread is overlaid as live while the
  inventory lags.
- **Selection.** The cursor and auto-select skip pending rows, through one
  predicate (`menuRowSelectable`), and a click on a pending row lands nowhere.
- **Background results** are routed BEFORE the in-flight match, which would
  otherwise drop them, because the pass never holds the operator's slot.
- **Hold and retry.** A held pass resumes when the operator's slot clears.
- **Cell 9.** Resuming a failed row by hand clears its mark.
- **The guard.** A source-parsing test,
  `TestMenuCodeReadsTheInventoryOnlyThroughTheViewedLookups`, fails any menu
  code that reads the inventory around the view.

**Tests:**
- cell tests;
- routing tests;
- generated-sequence invariants: 60 seeds of 120 steps each, over inventory,
  results, operator operations, cursor, Enter, Tab and click. After every step
  they assert:
  - no pending address is ever the selection;
  - the operator never dispatches on a pending row;
  - the pass never emits under an operator operation;
  - the queue never grows;
  - the pass's sets stay disjoint.

**One test of mine was wrong, not the code.** The held-pass test started the
operator's operation AFTER the pass attempt finished, by which point the pass
had already advanced. Nothing was held for the cleared slot to release. It now
starts the operation while the attempt is in flight, which is the only way a
hold arises.

**Mutations: 12 of 12 killed, each by a test that names it.**
- One first attempt was a BUILD-ERR: `return true` left variables unused. The
  harness correctly refused to count it, and it was re-run in a compiling
  form.
- The harness's whole-tree hash reported "not restored". A per-line check
  proved every original line intact, so that was the hash, not a leak.

**Suite:** unsandboxed `make test` exit 0, 197 packages. The one sandboxed
failure, `TestNotificationPTYConformance`, is the sandbox blocking PTY tests; it
passes unsandboxed.

### 2026-09-11 — M2 Tasks 8-10: the console runs the pass, placeholders, and only a start arms

**Task 8: the console runs the pass.**
- A Background effect goes to `runBackgroundOperation`. Its origin is marked
  Background, and it never takes the operator's `InFlight` slot.
- A background attach (the Implicit `background` arg on `attach`) sets neither
  focus nor the tracker, and a background success never steals focus. That is
  the operator's requirement that the pass not disturb the startup thread.
- A finished attempt dispatches the pass's next effect after the console
  unlocks.
- **A successful leave ends the pass.** This is a pure rule. It was found
  because a leave was followed by one more reattach.
- The sweep found one equivalent mutant: the queue-key prefix. Keys are unique
  because of the shared operation counter, not the prefix. The comment that
  claimed otherwise was corrected.
- `TestSwitchAsksTheIncomingChildToRepaint` failed once in a full run. It then
  passed 20 of 20 times on both trees, including the tree before Task 8. It is
  a race inside the test that predates #206.

**Task 9: placeholders, per the operator's UX.**
- The status row shows one greyed `<repo>` placeholder per pending thread, with
  the spinner on the one that is loading. A placeholder records no ChipSpan, so
  it is unclickable by construction.
- The status-row tick runs at 120 ms and is armed only while a thread is
  loading.
- In the switcher, pending rows are greyed and read `queued` or
  `reattaching…`. A failed row reads `reattach failed: <diagnostic>`.
- The cursor, auto-select and clicks skip pending rows through
  `menuRowSelectable`.

**Task 10: only a start arms the pass.**
- `armsReattachPass`: a start arms it, and a resume of one named thread does
  not (decision 9).
- `beginConsole`, split out of `runConsole`, runs the attach, then the arm,
  then `Run`. A console that never came up reattaches nothing behind it.
- **My first tests were wrong, not the code.** They drove `runConsole` with a
  child that had already exited. The initial attach refuses such a child, so:
  - the "arms" test failed;
  - the "never arms on a failed attach" test passed for the wrong reason: the
    exited child, not the failure it meant to test.

  The tests now run at `beginConsole` with a fake dispatcher, in three cases.
- My doc comment quoted the old argv and tripped
  `TestNoCurrentSourcesAdvertiseObsoleteCouchArgv`. It is reworded.

**Task 10 mutations: 4 of 4 killed**, each by the case that names it:
- the arm moved before the attach check;
- always arm;
- never arm;
- a resume arms too.

The M2-wide sweep is Task 12.

**Suite:** unsandboxed `make test` exits 0 across 197 packages.

### 2026-09-11 — M2 Task 11: the `COUCH_TRACE` timing trace

**What it records.** One line per event,
`<unix-ms>\t<event>\t<scope>/<tag>\t<detail>`:
- `startup`, stamped with the process start that couchcmd reads at package
  init;
- `first-frame`, `Run`'s first paint;
- `inventory`, for every inventory that lands. This event was added because
  Task 12 counts the refreshes during the pass;
- `pass-seeded`, with `pending=N`;
- `reattach-start`, with `attempt=N`;
- `reattach-done`, with `ok`, a diagnostic code, or `error`.

The trace holds addresses, counts and timings, never content.

**Shape.** `traceFile`, in the new `couchtty/trace.go`, is the one
append-only file plumbing. The keystroke probe now writes through it too
(`ARCH-DRY`). Each event is recorded at its single source in the console, not
derived from every `ReduceMenu` call.

**Mutations: 23 of 23 killed, 3 of them only after tests were added.**
- **`first-frame` on every paint survived at first.** The end-to-end test
  paints only once, because the attempts finish inside the spinner's 120 ms
  tick. `TestTheFirstFrameIsTracedOnce` now paints twice.
- **`reattach-done` for every operation survived at first.** No test finished
  an operator operation with a trace open. `TestOnlyAPassAttemptsEndIsTraced`
  now does.
- **`pass-seeded` on every inventory didn't compile at first.** In a compiling
  form, `TestAnUnarmedConsoleTracesNoSeeding` kills it. It is deterministic
  because it stops the console and joins `Run` before reading the trace.
- **The refactor exposed an older gap.** No test checked that the keystroke
  probe writes anything at all. `TestAClosedTracerStopsRecording` now checks
  that the record made before `Close` lands.

**The suite, under heavy load.** Unsandboxed `make test` ran at a 5-minute
load average of 77, with 31 agent processes and 27 zellij sessions. It failed
four tests outside this change:
- **Three zellij-timing tests**, in couchcore and launcher. Each returned an
  empty result at its deadline. Each passes in isolation, and all three
  packages pass in full once the load fell to about 10-16.
- **`TestSwitchAsksTheIncomingChildToRepaint`**, the race noted at Task 8.
  An A/B ran 40 runs per tree, interleaved under the same load:
  - the merge-base, before #206 touched couchtty: 1 failure;
  - HEAD before Task 11: 0;
  - the current tree: 0.

  The race predates #206.

The full suite reruns before close.

### 2026-09-11 — M2 Task 12, before the operator's part: the probe, the atlas, the sweep, and the plan checked against the code

**The probe's sample mode** (`64d297a7`).
`PAIR_PROBE_SAMPLE_SECS=N make test-reattach-cost` creates only the control
session and samples `zellij action query-tab-names` for N seconds (1-600). It
prints the window in unix ms, so it lines up with a `COUCH_TRACE` file. The
sampler and the latency summary are now functions the phased run shares.

A live 3-second run cleaned up its own session. It measured a median of
1954 ms, at load 5.8 with 27 zellij sessions. That is three calls, too few to
trust; the operator's 30-second run will settle it.

**The atlas.** `atlas/couch.md` gains:
- the pass, as a bullet in the startup costs: its decisions, not its cells;
- the placeholders, in the reserved-row section;
- `COUCH_TRACE`, beside `COUCH_INPUT_TRACE`.

The `COUCH_INPUT_TRACE` paragraph called it "the one env var" couch reads for
itself, which is no longer true. The claim is corrected and now guarded.

**The M2 sweep: 38 mutants over the final code, all killed in the end.** The
first pass ran 33 and killed 32. It found two gaps at the reducer, each of
which would have shipped silently. Each now has its own test:
- **Nothing checked that an inventory calls `expireAttached`.** Without the
  call, a thread whose pane exited reads live forever. Now pinned by
  `TestANewerInventoryTakesBackARowThePassAttached`.
- **Nothing checked that a failed first inventory leaves the pass armed.**
  Seeding from it ends the pass before it ever sees a real inventory. Now
  pinned by `TestAFailedFirstInventoryLeavesThePassArmed`.

**The plan, read against the code.**
- Four places still cited cells by their numbers from before the
  renumbering: decisions 8, 10 and 12, and the `Attached` entry.
- **Cell 12 described a queue prune that was never built, and should not
  be.** Cell 4 rules pruning out. At its turn, a queued thread that was
  parked, or relaunched from the CLI, is skipped. An archived one fails, with
  no row left to show the mark on. The table now says so.
- **Decision 11 promised the error's first line on a failed row; the row
  showed `reattach-failed`.** Now built:
  - the pass stores the error's first line;
  - `passSuffix` sanitizes it and fits it to the row, leaving the label
    `menuLabelFloor` columns;
  - the same fitting stops a long code overflowing a 40-column row, which it
    did before.
- Task 6's named tests landed under other names, so the plan now maps each
  cell to its test.
- M1's Task 4 items were done at M1's close but never ticked. They are ticked
  now, pointing at that Log line.
- The corrected prose is registered as dead tokens in
  `tests/plan-superseded-facts-test.sh`, so it cannot come back. The script
  also checks each token was really in the file before it was corrected.

**Suite.** Unsandboxed `make test` exits 0 across 197 packages, at load 2-3.
The earlier loaded run's three zellij-timing failures did not recur, and
neither did the older repaint race.

**Left:**
- the operator-assisted measurement and smoke;
- `sdlc milestone-close --milestone M2`, then the issue close.

### 2026-09-12 — M2 operator smoke: it works, with one status-bar bug, now fixed

**The operator's smoke test**, live on the real stack rather than the
scripted measurement:
- they detached and restarted couch to check the speed;
- they quit while threads were being reattached;
- "everything seems to work well".

**One bug.** The last thread the pass reattached (brain) stayed on the status
bar as a loading placeholder, spinning, for a long time. Opening the switcher
showed it live, and repainted the status bar.

**Cause, read off every branch.**
- While a thread loads, the status tick is the only thing that repaints the
  status row.
- When the last attempt lands, nothing is loading, so `syncStatusTick`
  stopped the timer without painting.
- `finishOperation` repaints only when the switcher is focused, and so does
  the inventory refresh it requests.

So the last frame the tick painted, with the spinning placeholder, stayed
until something else repainted. My test of the tick checked that it stops,
not the frame it leaves behind.

**Fix.** A stopping tick now paints one final frame, the one without the
placeholder (`syncStatusTick`). `TestTheFrameThePassLeavesBehindIsPainted`
waits for a painted status row that carries the attached thread's chip.
- Before the fix it failed, timing out: the symptom, reproduced.
- After the fix it passed 5 of 5 runs, alongside the tick test.
- A mutant that drops the final paint is killed by it.

**Left:**
- the operator's re-check of the status bar;
- the Task 12 measurement.

### 2026-09-12 — M2 Task 12: the measurement, the re-check, and a fixture race found at the close

**The measurement**, from the operator's traced restart (`COUCH_TRACE`, at
12:35). Five threads were detached; load was about 1.0-1.4, with 27 zellij
sessions.
- **The first frame painted 0.72 s after the process started**, so the
  startup thread could be typed in from then.
- **The first inventory landed at 2.65 s**, and the pass seeded from it with
  `pending=5`.
- **All 5 attempts succeeded, one at a time:** 249, 1179, 271, 302 and
  343 ms, a median of 302 ms.
- **The pass took 2.34 s.** Everything, pass included, was done 5.0 s after
  launch.
- **2 inventory refreshes landed during the pass**, 4 in all.

**`zellij action` latency during startup was at most 35 ms.** The sampler ran
the probe's phased run rather than sample mode, so there are no raw samples.
The phases can still be placed on the clock, walking each phase's printed
duration back from the report file's modification time. couch's whole
startup, from launch to its last event, falls inside the probe's "8 old
pattern" phase, with 18.6 s and 20.9 s to spare against about 1 s of
uncertainty. That phase had 633 samples (p50 19 ms, p95 23 ms, max 35 ms, no
errors), so no sample during startup exceeded 35 ms. The 9-minute quiet
baseline measured 7,528 samples: p50 21 ms, p95 23 ms, max 36 ms.

That meets the done-when's "measured, not assumed". The bound is
conservative, because the probe was driving zellij hard at the same time.

**Why the sampler went phased: not reproduced.**
- `make` passes the variables: sample mode refuses `0` through
  `make test-reattach-cost`.
- The same zsh one-liner shape passes both variables to `make`.
- The branch never switched, and the source had sample mode before the run.

The cause is unknown. The bound above does not depend on it.

**The re-check of the status-bar fix.** The operator restarted with the fix
and reports it "seems working", though they were unsure what to look for. The
fix's own evidence is the test that reproduced the symptom before it.

**A fixture race found at the close** (side-quest `dad4de14`). The full
suite failed twice on switch-nudge tests:
`TestASwitchThroughTheOperationQueueNudgesLikeAnyOther`, and earlier
`TestSwitchAsksTheIncomingChildToRepaint`.
- **An A/B at low load (200 runs per tree) found no failures anywhere**, so
  the race predates #206 and appears only under load.
- **The mechanism.** A fake pty child starts at the host's full height, while
  production children are born at the console's child size. The tests
  attached the fake while `Run` was starting, so whether the one startup
  layout resized it was a race. When the race was lost, the switch's nudge
  restored the child to the fake's own height. Production cannot lose that
  race.
- **Pinned.** Attached after the console starts, as production does, the old
  fake fails 5 of 5. Born at the console's size, all three tests with that
  setup pass 200 of 200.
- **A lesson was added.** The A/B said "pre-existing", but only the mechanism
  said "the fixture, not couch".

**Suite.** Unsandboxed `make test` exits 0 across 197 packages, at load
1-2.7.

### 2026-09-12 — M2 boundary review, round 4: FIX-THEN-SHIP

The review checked four claimed fixes by reverting each in a scratch overlay,
and every named test went red. It passed every ARCH principle. It raised:
- **BR-13 (Important, blocking): the README did not describe the pass.**
  Fixed: a README paragraph covers the pass, its placeholders and its
  non-selectable rows, the failure mark, and quitting part-way.
- **BR-14 (Minor): the plan restated `ReattachPhase` and `ReattachPass`, and
  the copy drifted.** Fixed as the rule:
  - the block is replaced by a pointer to `menu_reattach.go`;
  - the "one home" lesson now covers declarations;
  - the superseded-facts test fails on a re-paste.
- **Advisory, done:** a comment on `advanceReattach`'s counter guard. The
  guard stops the attempt counter wrapping to 0, the "no attempt" identity.
- **Advisory, noted:** this branch also carries docs-only commits from a
  parallel session. They are the #232, #233 and #234 issue files, and the
  `couch-slots` project file (`1cc55d99`). They ride the PR unchanged.
