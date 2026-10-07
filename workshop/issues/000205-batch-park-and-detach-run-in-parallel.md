---
id: 000205
status: working
deps: []
github_issue:
created: 2026-09-06
updated: 2026-10-04
estimate_hours: 3.34
card_mirror: 'ec7ab0fa323dde39fab1a0b59f2004ef990a85b6' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-04T17:59:49-07:00
claimant:
    operator: T
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
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
   *(2026-10-06: the guard moved into this issue as M1; the dependency is
   gone. See Revisions.)*
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
- The safety invariant must be **established here, before the pool** (M1; moved
  from `#214` on 2026-10-06): key admission by thread + operation class so one
  thread admits one launch-producing operation. Only then does removing the
  single worker leave the cross-thread ordering as the sole thing given up. See
  "Per-thread admission guard" below.
- Results still land on the console goroutine through `q.results`, so completion
  handling and `c.mu` discipline are unchanged.

### Per-thread admission guard (M1, moved from `#214` on 2026-10-06)

**What exists.** Two guards already hold, and neither is per thread at the
queue:

- `dispatchMenuOperation` (`couchtty/menu.go:1785-1788`) drops any operator
  operation while `InFlight` is set. It is global and silent.
- `CommitStartClaim` (`couchcore/threadstore.go:564-592`) refuses a second
  occupant (`already has N incarnation(s)`, `open park transaction`) under the
  revision CAS. It is the invariant of last resort and stays as is.

**The gap.** Four paths enqueue onto the one `operationQueue` with keys that
carry no thread: operator menu operations (`console.go:1661`), the reattach pass
(`console_reattach.go:50`), continuations (`console_continuation.go:195`) and
remote socket `resume|reboot` (`couchcmd/slot_operations.go:141`). Only the
single worker (`console.go:629`) orders them per thread. Under a pool, a resume
landing inside relaunch's park→resume window would win the CAS and the relaunch
would fail `ParkedNotResumed`: safe, but confusing and unexplained.

**Rule (operator decision, 2026-09-08, carried from `#214`).** Exactly one
launch-class operation (`resume`, `relaunch`, `reboot`, `switch-agent`,
continuation launch, park/detach) may be in progress per thread. A later request
for that thread is **refused**, not queued or coalesced, with a message naming
what is already running there. Different threads are unaffected. The guard may
be relaxed later; for now, one.

**Where (revised 2026-10-06, see Revisions).** One in-memory `ThreadGate` on
`Couch`, held by every couchcore lifecycle entry where its address is final, with
re-entry for composites through the context. The queue is the wrong place: it
cannot name a remote job's or a path-addressed thread, nor `leave`'s many. The
durable plan has the entry table. ~~One admission map, keyed by (thread address, operation class), at
the point all four paths enqueue: beside `operationQueue.pending`, or a wrapper
around `Enqueue`.~~ Release happens on result delivery. Each path supplies its
address explicitly; a job without one (the global `leave`) is not
launch-class. M1 lands with the single worker still in place and is tested
there, so M2 changes only the worker count.

**Out of scope here** (stays in `#214`): naming the `binding lost` failure, and
the in-pane restart's unclaimed ledger launch.

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

#### Answers (2026-10-06, from the durable plan's ARCH-ORDER table)

The cells above, and the ones the code can actually produce, are answered here:

| Event (cannot be blocked by the caller) | Policy | Rollback |
|---|---|---|
| Second lifecycle op on a held thread (pass vs remote vs continuation vs operator) | **Refuse** with `ThreadBusyError{Address, Running}` | none: nothing ran |
| `Leave` reaches a thread a reattach attempt holds | **Wait** on the gate (ctx-bounded), then detach or park | none |
| `RecoverActiveParks` reaches a thread an operation holds | **Wait**, then recover | none |
| Operator quits mid-pass | Pass holds new attempts (cell 10, unchanged). At most `LifecycleParallelism` attempts are in flight; `Leave` waits on each thread it needs | none |
| Caller `ctx` cancelled mid-park | The caller returns at once (prompt cancel, an existing tested contract). The thread stays guarded while the park work runs on: the record's open park transaction is refused by every launch, detach, recovery and archive, and the park worker refuses a second transaction (revised in M1; was "submit waits") | park's recovery modes, unchanged |
| Pass completions arrive out of order | `finishReattach` matches by attempt number in the in-flight map, not "the" attempt | none |
| One of N `Leave` threads fails | Others continue; joined error plus partial result (D6) | none: already-detached threads stay detached |
| couch process dies mid-`Leave` | Unchanged from today: per-thread durable transitions; the next startup reconciles | n/a |
| A thread's agent exits on its own while its park is in flight | Unchanged: park already treats child death as completion evidence (`awaitCompletionAndChildDeath`). The gate stays held until park returns, so no other operation sees the half-torn-down thread | park's recovery modes |
| `Leave` waited behind a holder, so its snapshot row is stale | `leaveOne` takes `holdWait` **first**, then re-reads the record with `GetThread` and decides from that. Never from the snapshot | none |
| Completion-side effects run after release (`finishOperation` → `attach`/`AbortStarted` on the console goroutine) | `AbortStarted` can quiesce the thread's session **by address** (`couch.go:857`, cold-start shapes that own the session). Occupancy alone does not protect a newer operation: relaunch, detach, park, switch-agent and leave all act *on* live threads. A relaunch admitted in the window would park the aborted start's incarnation and start a new session on A, and the late abort would then kill that session. **Policy:** `AbortStarted` is a gated drain. It takes `holdWait` on the thread and re-enters through its caller's ctx inside composites. Under the hold, it re-checks that the record's live incarnation is still this start's PID and identity before any address-scoped quiesce. If not, it retires only its own actor record. This race is reachable today, between a remote or continuation job on the worker and the console's completion path. Test in Task 3 | none: the newer session is left alone |
| Boot: `StartInteractive` meets `RecoverActiveParks` | Cannot happen today: `RecoverActiveParks` starts only after the startup operation returns (`couchcmd/run.go:489-496`). If that order ever changes, the startup resume is refused busy; that is acceptable, and the next start succeeds | none |

### The startup reattach pass uses the same pool (scope revision 2026-10-04)

Couch's startup background pass (#206, `couchtty/menu_reattach.go`) is the
other direction of the same batch: detach on the way out, reattach on the way
back in. It is serialized twice, and the pool must lift both:

- **The pass holds one slot.** `advanceReattach` emits nothing while
  `pass.Loading` is set, so at most one attempt is in flight. Make that a set of
  in-flight attempts up to the pool bound; `finishReattach` releases the one
  that completed by its attempt number, not "the" attempt.
- **The queue worker is single.** Same `operationQueue` as park/detach; the pool
  above covers it.

What stays: every attempt is still `warm-only` and re-proves its thread at its
turn; the pass still never takes focus; it still holds while the operator has an
operation in flight (cell 10, which is what makes quit-mid-pass safe; a pool
that keeps starting attempts would break it); failures stay per row.

Why it is worth it now: the #206 probe measured raw `zellij attach` at about
55 ms with 8 concurrent attaches finishing in 121 ms total and no movement in
`zellij action` latency. After #228 a couch reattach costs about 280 ms, nearly
all of it proof snapshots, so a sequential pass grows linearly with the fleet:
about 5 s for 17 background threads at the 18 slots now in use.

## Done when

- Quit (`leave`, detach or park) of N threads runs them concurrently inside
  `Couch.Leave`, bounded, finishing siblings when one fails.
- ~~A batch park/detach of N threads completes in materially less wall-clock than
  N sequential operations; measured, with the working-agent count recorded
  (per `workshop/targets/workbench-latency.md`, a timing without its co-tenancy
  is not a measurement).~~ **Revised 2026-10-07 (operator):** accepted on the
  operator's live smoke test of the #205 build at their full fleet ("tested a
  couple of rounds, much more pleasant now"). The traced before/after did not
  capture, and the operator chose not to repeat it. Not measured; see Revisions.
- Concurrency is bounded; the bound is stated with its reason, not tuned by feel.
- No two operations ever run on one thread — asserted by a test, not inherited
  from the queue's current shape.
- The per-thread guard covers all four enqueue paths. A second launch-class
  request for a busy thread is **refused** with a message naming what is
  running; a request for a different thread is admitted.
- A refused second gesture leaves the thread resumable: a test fires
  resume-then-relaunch on one thread, sees one admitted and one refused, then
  resumes successfully.
- Every interleaving cell above has a stated answer in the issue and a test.
- A failing operation inside a batch leaves the other threads' outcomes intact
  and the failure visible.
- ~~`#204`'s suite gains a counted invariant for the batch path — batch park of N
  threads issues O(N) store writes, not O(N²).~~ **Revised:** #204's suite does
  not exist yet (the issue is open). The counted invariant is pinned locally
  instead: `TestLeaveSignalsEachThreadExactlyOnce`, plus the generated-sequence
  test's "attempted at most once".
- The startup reattach pass runs up to the pool bound in parallel, and it still
  holds while the operator has an operation in flight (quit mid-pass leaves no
  extra attempt behind, by test). ~~Its wall-clock at N threads is measured
  before and after with the working-agent count~~ **Revised:** operator smoke
  test, as above.
- ~~`TestAReattachedChildKeepsItsTrackingMode` (#196) passes unmodified, and a
  variant runs N reattaches at once.~~ **Revised:** that test no longer exists.
  #196's invariant now lives in per-child state
  (`TestCouchParentCaptureStableAcrossChildMouseModes`, passing). Adoption still
  runs on the console goroutine one completion at a time, so N parallel
  reattaches arrive as N serial adoptions and create no new interleaving.

## Plan

Durable plan: `workshop/plans/000205-batch-park-and-detach-run-in-parallel-plan.md`.

- [x] M1 — Per-thread `ThreadGate` in couchcore on every lifecycle entry; refusal
      reaches the operator; nothing outlives its hold (plan Tasks 1–5).
- [ ] M2 — Bounded parallelism: `Leave` fan-out, park worker bound, reattach pass
      in-flight set, console queue workers; measured before/after (plan Tasks 6–12).

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: greenfield-go-module   design=0.10 impl=0.32
item: cross-cutting-refactor design=0.20 impl=0.20
item: smaller-go-module      design=0.05 impl=0.16
item: tui-screen             design=0.10 impl=0.32
item: milestone-review       design=0.00 impl=0.20
item: atlas-docs             design=0.05 impl=0.04
item: smaller-go-module      design=0.05 impl=0.16
item: smaller-go-module      design=0.05 impl=0.20
item: tui-screen             design=0.10 impl=0.40
item: real-api-discovery     design=0.00 impl=0.24
item: milestone-review       design=0.00 impl=0.20
item: atlas-docs             design=0.05 impl=0.04
design-buffer: 0.15
total: 3.34
```

The items in order:
- **M1:**
  - `ThreadGate` (greenfield);
  - gating about 13 couchcore entries, plus `AbortStarted` gaining a ctx and
    `Couch.Park` (cross-cutting);
  - `parkWorker` ordering and the capacity wait;
  - the console busy path, the off-goroutine abort and the continuation cleanup
    (tui);
  - the M1 review;
  - the atlas update.
- **M2:**
  - `Leave` fan-out;
  - the registry mutex across about 20 sites;
  - the reattach pass in-flight set, the generated-sequence test, placeholders
    and queue workers (tui);
  - the live before/after measurement (discovery budget);
  - the M2 review;
  - the atlas update.

Design hours are discounted because the durable plan already settles the
decisions. Implementation hours are 40% of the v2 table, per v3.1.

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.* (`sdlc estimate-source` flags the doc as stale versus the ledger, so the numbers are provisional.)

## Log


- 2026-10-07: closed M1 — M1 per-thread ThreadGate, plus boundary-review round 1 fixes (BR-1 park join never begins unheld; BR-2 registry lock pulled into M1, all ~20 accesses through registry()/mutateRegistry(), -race test red without the lock; BR-3 refusal undoes only its own expected-exit marks; BR-4 wait tests for RecoverActiveParks and AbortStarted, cancelled abort wait). Tests: gate decision/acquire/wait/reentry/stale-token; every lifecycle entry refuses a held thread; resume-during-relaunch and relaunch-during-resume refused naming the holder; Leave waits and decides after waiting; abort identity paths; couchtty busy skip/retry/notice; GoTracked. Unsandboxed clean-env go test: couchcore, couchtty, couchcmd ok. Full-suite residue is pre-existing or environmental (verified vs merge base): gcruntime archive locator, artifactpath classification (only generated runtime-bundle files and pre-existing console_messages.go), couchsingleton selection-size under long TMPDIR, test-changelog and test-pair-embedded-runtime under session env.; review verdict: FIX-THEN-SHIP
### 2026-09-06

Operator request, split from `#206` (auto-reattach at startup) at their
instruction — the two share a motivation but not a risk profile. This half is
well-founded: the store's concurrency model, the dedup invariant and the park
future all already exist, so the change is small and lands on seams built for it.

`#206` is the one that needs a measurement before its approach is chosen.

### 2026-10-04

The operator now runs 18 active slots and restarts have become slow. They asked
for this issue to cover the startup reattach pass as well; see Revisions.

### 2026-10-06: design reading for M1 (findings that move the Spec)

- **Global detach is one job, serial inside couchcore.** Quit dispatches
  `leave`, one queue request; `Couch.Leave` (`couchcore/park.go:170`) loops every
  record and detaches or parks each in turn ("Serial by choice"). A worker pool
  in `operationQueue` would not parallelize it at all. Leave-park also funnels
  through `parkWorker`, created with capacity 1 (`park.go:549`).
- **A per-address guard already exists for park.** `parkWorker`
  (`couchcore/parkworker.go`) keeps an `active` map keyed by `ThreadAddress`,
  refuses "another park transaction already owns this address", and bounds
  admission. That is the M1 shape, today scoped to park only (ARCH-DRY).
- **The queue is the wrong place for the guard.** A remote job's address is only
  known after `prepare` runs on the worker (`console_remote.go`), path-addressed
  `resume`/`reboot` resolve their thread inside couchcore, and `leave` touches
  every thread from one job. All launch paths converge on a few couchcore
  entries instead: `ResumeContextWith`, `Relaunch`, `Detach`, `SwitchAgent`,
  `Continue`, `RetryContinuation`, `Reboot`, the `PairLifecycle` park entries,
  and `Leave`'s per-thread step. Composites call leaves (`Relaunch` → park +
  `ResumeContext`; `RetryContinuation` → `ResumeContextWith`), so a leaf-level
  guard needs re-entry for the holder.

### 2026-10-06: M1 Tasks 1–2

- Task 1 landed the gate (`9eff1a00`). Re-entry now keys on the hold's identity
  token: the stale-context test showed address-keyed re-entry lets an outlived
  context into a later holder's hold.
- Task 2: the plan's D4 ("submit waits for the park after cancellation")
  contradicted a deliberate, tested contract (prompt cancel), and the durable
  park transaction plus the worker entry already guard the thread. D4 was
  dropped, the `parkWorker` delete-before-close ordering kept, and a test now
  pins that guard. The ordering stress test (1000 runs) passed even before the
  fix, so it is a regression guard, not a reproduction. Plan Revisions record
  both changes.

### 2026-10-06: M1 Tasks 3–5

- Every lifecycle entry holds the gate (`f365eca6`). The console carries
  `MenuEvent.Busy` and runs the abort off its goroutine (`f63e6793`).
- `make runtimebundle-generate` was needed once in this fresh checkout before
  couchtty would build.
- Deviations, recorded in plan Revisions:
  - `Couch.Park` joins an open park transaction, keeping the deliberate
    recovery/retry coalescing.
  - `AbortStarted` waits first and cleans up exactly once per path.
  - The late-abort and console-wait tests were taken in a narrower form.
- Mutation checks:
  - A gate that admits everything turns the entry table, both relaunch-order
    tests, and the leave tests red. `TestLeaveWaitsForAHolder` needed its
    window widened to 150 ms to fail reliably under load.
  - Removing the busy re-arm turns the continuation test red.
  - Removing `event.Busy` from the reattach skip turns its test red.
- Sandboxed couchcore run: the only failures are PTY tests the sandbox blocks
  (`ptychild: start … operation not permitted`), a known environment limit.
- Atlas: `atlas/couch.md` gains "One lifecycle operation per thread".
- Spec: the interleaving answers are copied in.

### 2026-10-07: M1 boundary review, round 1 (FIX-THEN-SHIP, 4 Important + 2 Minor)

The first `milestone-close` hit the 30-minute review timeout and refused
without writing. It was re-run with `WF_REVIEW_TIMEOUT=75m`. Fixes, each for
the class the finding named:
- **BR-1:** `Couch.Park`'s joining path maps begin-capable modes to `retry`
  (`parkJoinMode`), so a stale "open transaction" read fails closed instead of
  beginning a park unheld.
- **BR-2:** the registry mutex (plan Task 8b) is pulled into M1. All ~20
  `c.reg`/`c.names` accesses go through `registry()` and `mutateRegistry()`.
  `TestConcurrentRegistryWritersLoseNoUpdate` turns red under `-race` with the
  lock removed.
- **BR-3:** continuation watches record the expected-exit marks their own
  operation added (`markThreadExitsLocked`); a busy refusal removes only those.
  The test now proves a park's mark survives.
- **BR-4:** wait tests for `RecoverActiveParks`, `AbortStarted` (matching path),
  and a cancelled abort wait.
- **Minor:** `startStillOwnsThread` needs a matching incarnation, since zero
  incarnations is not ownership.
- **Minor:** the cancelled-park test asserts its refusal causes. That surfaced
  that resume refuses on its own live-incarnation rule before reaching the
  open-park check; the test now requires a structured resume refusal.

Lessons are added to `workshop/lessons.md`.

### 2026-10-07: M1 closed (round 2 FIX-THEN-SHIP), fixes bundled

Round 2 disposed BR-1..4 and BR-6 and found three things still open. All are
fixed in the M1 close commit:
- **BR-5:** a test for a thread with zero incarnations.
- **Exit marks:** a repeat of the "cleanup wider than its own effect" family.
  Expected-exit marks are now owner-counted (`exitMarks`): undo decrements, a
  real exit consumes them all. A test covers both orderings and fails if undo
  deletes outright.
- **Doc comment:** `withoutDead`'s comment is back on its function.

Mutation checks are red as expected. couchtty and couchcmd pass unsandboxed.

### 2026-10-07: M2 started, operator decisions

- Bound: "the 4 should be half of CPU cores, not a fixed number", so
  `LifecycleParallelism = max(1, NumCPU/2)`.
- The M1 live smoke test is skipped: the same-thread collision is hard to
  trigger by hand. One live check at the end of the issue covers both
  milestones.
- The issue lands whole (M1+M2), not M1 alone.
- The baseline is the operator's running main-built couch, measured with
  `COUCH_TRACE` on its next restart.
- Task 8: the park worker waits for capacity instead of refusing.

### 2026-10-07: M2 implemented; operator smoke test; rebased

- **M2 tasks:**
  - Task 8: the park worker waits for capacity.
  - Task 7: `Leave` fan-out.
  - Task 9: the reattach pass's in-flight set.
  - Task 10: queue workers, sized at the limit plus one for operator headroom.
  - Counted invariant and atlas.
- **The traced A/B did not capture.** The `/tmp/couch205-*` trace, input and
  exit files were never written. The only latency window that completed
  (12:02–12:12) read `zellij action` p50 18 ms, p95 22 ms, max 65 ms at 53
  sessions and 11–12 agents, with an unknown overlap with the restarts.
- **Operator decision:** close on the live smoke test, "tested a couple of
  rounds, much more pleasant now".
- The branch was rebased onto main (35 commits, including #362 peek) before the
  `pair:0` test. The couch packages are green after the rebase.
- `TestColdResumeOfAParkedPrimaryRegistersFromBothOrigins` flakes under `-race`
  about 50% of the time on main as well (4 of 8 at the merge base). It is
  pre-existing and not #205.

### 2026-10-07: close-time evidence

- **Latency sampler, 12:02–12:32,** covering the operator's base and #205
  restarts. `zellij action` stayed at p50 18–19 ms and p95 22–24 ms, with maxima
  of 65, 129 and 148 ms and no errors. The context was 53 sessions, 11–12
  agents and 12 cores. Parallel reattach did not visibly hurt interactive
  latency. The restarts can't be pinned to sub-windows without the traces.
- **Measured actual is unreliable after the rebase.** The same window read
  4.69 h at the M1 close and 3.70 h after rebasing, because the rebase rewrote
  committer timestamps. M2 closes with `--no-actual`. The issue close adopts
  the engine's measured total, and this note explains it.
- **Full verification** at branch HEAD, in a scratch worktree, unsandboxed:
  - The couch packages are green; couchcore needs a timeout above the 10-minute
    default under full-suite load (553 s alone).
  - Remaining failures are pre-existing or environmental: gcruntime archive
    locator, artifactpath classification (only the pre-existing
    `console_messages.go`), the cold-resume `-race` flake, and the TMPDIR and
    session-env shell tests.

### 2026-10-07: M2 review round (FIX-THEN-SHIP): BR-9 and three minors

**BR-9, concurrency cells untested above bound 1.** Real tests now run above
bound 1:
- `TestStopMidPassCancelsEveryParallelAttemptAndStartsNoMore` (couchtty, bound
  3, 5 threads, end to end through the worker pool).
- `TestLeaveCancelledMidFanOutFinishesStartedAndStartsNoMore` (bound 2).

The second one surfaced two things:
- **A wrong policy sentence.** Started threads see the cancellation through
  their shared ctx and stop at their own safe points; they do not "finish".
  This is the serial behaviour, and a detach that stops mid-way destroys
  nothing.
- **A real race.** `select` picks at random when the context is done and a
  unit is free, so a cancelled `Leave` sometimes started a thread.
  `TestLeaveCanceledDoesNotRetireDeadIncarnation` flaked on it. The fix checks
  `ctx.Err()` before and after the select.

The dispatch check is an equivalent mutant at the observable level, because
`Detach` refuses a cancelled ctx before signalling.

Cells covered by an existing mechanism test rather than a new one, with
reasons:
- **Leave-park beside another park.** The worker-level capacity tests run
  concurrent submitters through the same `Submit` that `Leave`'s park path
  calls. The lifecycle fake keeps one `lastRequest` and is not safe for
  concurrent use.
- **Late abort against a relaunch under N workers.** The gate-level tests
  (`TestAMismatchedAbortStillClosesItsOwnHandle`,
  `TestAbortStartedWaitsForAHolderThenQuiesces`) exercise the race on the
  couchcore objects that hold it. The console only routes.
- **A burst of N reattaches (#196).** Adoption runs on the console goroutine,
  one completion at a time.

**Minors:**
- `Leave` errors name their thread.
- The queue-worker and atlas wording now says "the pass alone".
- The park-worker capacity tests synchronize on a wait hook instead of
  sleeping.

## Revisions

### 2026-10-04: scope adds the startup reattach pass

**Reason.** The operator, at 18 active slots: restarting has become slow. No
issue owned making the startup reattach pass parallel: #206 chose a sequential
pass, #320 defers parallelism here, and this issue covered only park and detach.

**Delta.** Added a Spec subsection, two Done-when bullets and three Plan steps
that bring the #206 reattach pass under the same bounded pool. The pass's own
single `Loading` slot is the second serialization to lift. The operator-in-flight
hold and the warm-only, per-row-failure rules are kept. Dependency on #214 is
unchanged; it matters as much here, since a parallel pass makes a racing
resume more likely.

Line drift noted: the single worker is now started at `console.go:629`
(`c.operationQueue.Run(c.stop)`), not `:538` as the Problem says.

### 2026-10-06: per-thread guard moved in from `#214`

**Reason.** Re-reading `#214` against current code showed its named race
already blocked within one console, and that the per-thread guard is only
needed once this issue removes the single worker. The operator chose to land
the guard here, so the old ordering and its replacement ship together.

**Delta.** Dependency on `#214` removed. Added the "Per-thread admission guard"
Spec subsection, two Done-when bullets carried from `#214`, and split the Plan
into M1 (guard, single worker) and M2 (pool for batch park/detach and the
startup reattach pass). `#214` keeps the non-concurrency half.

### 2026-10-06: guard moves to couchcore; quit parallelises inside `Leave`

**Reason.** Design reading (Log, 2026-10-06): `leave` is one queue job with a
serial loop inside `Couch.Leave`, and the queue cannot name the thread for
remote, path-addressed or `leave` work. The operator approved the couchcore
design.

**Delta.** The Spec's "Where" now names a couchcore `ThreadGate` (the old
sentence is struck through). Done-when gains a bullet for `Leave` running
concurrently. The Plan points at the durable plan, whose M1/M2 split replaces
the inline sub-steps. One shared bound, `LifecycleParallelism = 4`, is chosen
with its reason in the plan (D5).

### 2026-10-06: durable plan reviewed

Fresh-context plan review ran five rounds and ended Approved. What it changed:
- `RecoverThread`, `Stop` and `AbortStarted` are gated.
- `Leave` decides only after waiting.
- `parkWorker` frees its address before signalling done.
- A registry mutex guards parallel workers. The race exists today between `Forget` and the worker.
- Busy reaches the console as `MenuEvent.Busy`, not as a resume diagnostic.
- A late abort holds the thread, re-checks identity, and never waits on the console goroutine.

Plan: `workshop/plans/000205-batch-park-and-detach-run-in-parallel-plan.md`.

### 2026-10-07: measurement bullets accepted on the operator's smoke test

**Reason.** The traced before/after restarts did not produce their files, and
the operator closed on a live smoke test of the #205 build instead: "no,
consider smoke test passed, close this issue and merge."

**Delta.**
- The two measurement Done-when bullets are struck and annotated as
  operator-observed.
- The #204 counted-suite bullet is re-pointed to the local counted tests.
- The #196 test bullet is re-pointed to its current form, with the reason.
