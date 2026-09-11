# Reattach detached threads at startup — Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `couch` attaches the thread for its directory as fast as it can prove
that one thread, then reattaches every other detached thread one at a time in
the background, without taking the operator's operation slot or their focus.

**Architecture:** Two milestones.
- **M1** narrows startup's blocking work to the threads its readers act on:
  the cwd thread, and any thread whose layout differs from couch's. It also
  hardens the one proof whose answer narrowing would otherwise change.
- **M2** adds a **reattach pass**: a pure sub-reducer inside the menu's single
  transition authority (`MenuState.Reattach`).
  - Seeded once, from the first inventory after the console starts.
  - One attempt at a time, through the existing operation queue, with a
    background origin, and never while an operator operation is in flight.
  - Its resume carries `warm-only`, so the pass can never start an agent.
  - A background attach adopts a pane without touching focus.

**Tech Stack:** Go. `couchcore` (inventory, resume), `couchtty` (menu reducer,
console), `couchcmd` (wiring).

**Depends on #230.** Six post-acknowledgement failure routes delete the zellij
session a warm reattach was reattaching (#230's route table). The route a
background reattach actually takes is #230's route 6: a warm reattach's
registration check is satisfied by the pre-existing session, so a `pair resume`
client that dies early is first noticed when the console's attach fails, which
calls `AbortStarted`, which quiesces. M2 does not start until #230 has merged.

**#230 does not move the seam M2 builds on.** An earlier #230 draft introduced
`ResumeStart`; its plan gate showed nothing would read the field that motivated
it, so it was dropped. M2's `ResumeOptions{WarmOnly}` attaches to
`ResumeContext`, unchanged.

---

## Scope

**In:** the whole `## Done when` of #206, plus startup narrowing (the operator
asked for it on 2026-09-11; the #206 Revision of that date put it here).

**Out, each with a reason:**
- **Bounded-parallel loading (strategy C).** `zellij attach` measured about
  55 ms, and the operator works in the cwd thread while the pass runs.
- **An opt-out flag.** The Spec says "with no operator action". Add one when
  somebody asks.
- **#214's per-thread guard.** The pass keeps at most one attempt of its own;
  an operator resume of the loading thread is adopted, not duplicated; and
  `DecideResume` still refuses an occupied incarnation.
- **The first inventory's O(C) cost.** It runs on the refresh worker, off the
  critical path. #229 owns post-mutation refresh cost.

## Deviation from Done-when, for the operator to confirm

Done-when says: *"If B or C: the switcher shows every known thread
immediately."* This plan shows every known thread **when the first inventory
lands**, which is before any reattach attempt starts — so no row ever waits on
a reattach, which is what the bullet was protecting. It is not literally the
first frame: before that inventory, the switcher reads "thread inventory
unavailable", exactly as today.

Seeding the switcher from startup's rows instead was rejected: after M1 those
rows deliberately carry unasked candidates, and feeding unasked state to a
reader of attach state is what #228's close review closed off
(`launcher.RequireAttachState`). **Flagged in the close for the operator to
accept or reject.**

## Decisions (the Spec's open cells, answered)

1. **Order.** Most recently active first (`LastActiveAt` descending), ties by
   `(RepoScope, Tag)`.
2. **What is reattached.** Seeded from rows that are `ThreadDetached` **or**
   resume-shaped `unknown`, minus the startup root, which the pass subtracts
   explicitly. `unknown` is included because it means "the proof could not be
   asked", not "not detached" (see cell 4). `warm-only` decides at attempt
   time, so an `unknown` row that is not really detached is skipped, not
   started.
3. **Enter or click on a queued row: jump.** It leaves the queue and
   dispatches as an ordinary operator resume.
4. **Enter or click on the row being reattached: adopt.** No second attempt.
   The in-flight slot takes the pass's attempt identity; on completion it
   lands focus and reports like an operator resume.
5. **A reattach that fails** marks its row `reattach failed: <code>`. The pass
   moves on. Enter on that row is an ordinary resume and clears the mark.
6. **A thread that is no longer warm** when its turn comes — parked, attached
   elsewhere, or its session gone — is **skipped silently**, not marked
   failed. That is the `resume-not-detached` refusal, and `resume-session-gone`
   from `confirmStillDetached` joins it.
7. **A thread detached by the operator during the pass is not reattached.**
   The queue is never added to after seeding.
8. **Quit mid-pass.** The pass never advances while an operator operation is
   in flight (cell 12), so a queued leave is never followed by a new attempt.
   SIGHUP, SIGTERM and a last-pane exit cancel the attempt in flight; after
   #230 that leaves its session detached. Everything not yet reattached stays
   detached.
9. **Rows are rendered from the pass while its own mutation is in flight**
   (cell 13), because the inventory passes through states — stale incarnation,
   then Busy, then Live-but-unhosted — that would otherwise show as
   "stale…"/"parking…" and refuse Enter.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `ProjectDetachedSessions` (claim counting) | `cmd/internal/couchcore/detachedsessions.go` | modified |
| `startupAsks` | `cmd/internal/couchcore/startup.go` | new |
| `gatherThreadEvidence` (ask filter) | `cmd/internal/couchcore/actionableinventory.go` | modified |
| `ReattachPass` | `cmd/internal/couchtty/menu_reattach.go` | new |
| `ReduceMenu` (pass branches) | `cmd/internal/couchtty/menu.go` | modified |
| `MenuOperationOrigin.Background`, `MenuEffect.Background`, `MenuEvent.Background`, `MenuEvent.Diagnostic` | `cmd/internal/couchtty/menu.go` | modified |
| `rootStateText` | `cmd/internal/couchtty/menu_render.go` | modified |
| `ResumeNotDetached` (diagnostic) | `cmd/internal/couchcore/resume.go` | new |

- **`ProjectDetachedSessions`'s duplicate-name rule must count claims over
  every binding in the scope's index, not over the caller's candidate list.**
  Today it counts `claims[name]++` across the bindings it was passed. Two
  addresses bound to one session name is its fail-closed case, and narrowing
  the candidate list silently removes that protection: ask about one of the
  two and its name looks unique, so a thread whose binding is ambiguous is
  reported Detached and startup resumes it. The index is already fully read
  (`ReadSessionNameIndex`), so counting over it is free. This is a latent
  weakness #228's narrowing introduced; M1 depends on it and fixes it first.
- **`startupAsks(requested Layout, repoScope, workingPath string) func(ThreadRecord) bool`**
  decides which resume-shaped candidates startup's blocking inventory
  resolves. It is the union of what the readers of startup's rows filter on:
  - `SelectResumableRoot`, `PathHoldsUsableThread` and
    `PathHoldsUnreadableThread` read only rows at the cwd
    (`Address.RepoScope == repoScope && WorkingPath == workingPath`);
  - `ResolveLayoutConflicts` reads only rows whose
    `NormalizeLayout(string(record.Layout)) != requested`.

  A candidate outside both sets is left `ProofUnresolved` → `unknown`. No
  reader of startup's rows can act on it, and the rows never leave
  `StartInteractive` (`StartResult` carries none).
  - **Applied before `ResolveEstablished`, not only before the zellij ask.**
    `ResolveEstablished` runs per resume-shaped record and reads that thread's
    ledger, so filtering only the zellij call would leave time-to-first-frame
    growing with the store. That is the shape `workbench-latency` forbids.
  - **DRY rationale:** one predicate beside the readers it serves, pinned by
    an equivalence test so a new reader cannot silently need more.
- **`ReattachPass`** lives in `MenuState`, so the menu's single transition
  authority owns every interleaving:

  ```go
  type ReattachPhase uint8

  const (
      ReattachIdle    ReattachPhase = iota // no pass (not armed, or finished)
      ReattachArmed                        // waiting for the first inventory
      ReattachRunning                      // seeded
  )

  type ReattachPass struct {
      Phase          ReattachPhase
      Root           couchcore.ThreadAddress
      Queue          []couchcore.ThreadAddress            // ordered, not yet attempted
      Loading        couchcore.ThreadAddress              // zero when no attempt runs
      LoadingAttempt uint64
      Attached       map[couchcore.ThreadAddress]bool     // landed; awaiting an inventory that shows them Live
      Failed         map[couchcore.ThreadAddress]string   // diagnostic code per row
  }
  ```

  Attempt identities come from `MenuState.OperationSequence`, so an adopted
  attempt matches the in-flight slot through the existing
  `menuOperationMatches`. `cloneMenuState` deep-copies `Queue`, `Attached` and
  `Failed`, keeping a nil map nil, because tests compare whole `MenuState`
  values with `DeepEqual`.
  - **`Attached`** is what makes cell 13 work: the pass, not the lagging
    inventory, is the authority for a row it just mutated. An address leaves
    it when an inventory shows that thread Live.
- **`MenuEvent.Diagnostic`** carries `ResumeDiagnosticOf(err)` so cell 6 is
  decided by a code, not by matching error text.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `warm-only` on resume | `couchcore/resume.go`, `ops.go`, `operationdispatch.go` | modified | `ResumeContext` |
| `Console.ArmReattachPass` | `couchtty/console_reattach.go` | new | menu reducer |
| `runMenuOperation` (background) | `couchtty/console.go` | modified | `operationQueue` |
| `installObservedThreadActor` (background) | `couchtty/console.go` | modified | pane adoption |
| `finishMenuRefresh`, `finishOperation` (dispatch pass effects) | `couchtty/console_menu.go`, `console.go` | modified | menu effects |
| `runConsole` arms the pass | `couchcmd/run.go` | modified | console lifecycle |
| `COUCH_TRACE` timing trace | `couchtty/inputtrace.go` (extended), `couchcmd/run.go` | modified | append-only file |
| reattachcost `sample` mode | `cmd/probes/reattachcost/main.go` | modified | zellij CLI |

- **`warm-only`** is an `Implicit` `ArgSpec` on `resume`. What keeps it off
  the CLI is `bindArgs` skipping implicit arguments (`couchcmd/run.go`), with
  `validateOperationCall` as the second line; the test asserts `couch resume
  <tag> --warm-only` fails with "unknown flag".
  - `ResumeOptions{WarmOnly}` refuses **before** `resumeEvidence`, so a
    verified-parked thread is refused without resolving a binding or writing
    a catalog entry.
- **`installObservedThreadActor` gains a `background bool`.** With it set, it
  appends the pane and sets `c.active` only if empty, but **never** sets
  `c.focus` or seeds `c.tracker`. Without this, a pass completion arriving
  while `c.active == ""` — which `onExit` produces whenever the last pane
  exits while the switcher is focused — moves focus to a pane the operator
  never asked for, with no screen takeover, so their typing goes to an
  invisible agent.
- **The console seam** is the existing fixture. Cancellation tests use
  `SetOperationDispatcher`, not `setTestOps`, because `setTestOps` drops
  `call.Context`.
- **`COUCH_TRACE=<path>`** appends `<unix-ms>\t<event>\t<scope>/<tag>\t<detail>`
  for `startup`, `first-frame`, `pass-seeded`, `reattach-start`,
  `reattach-done`. It reuses `inputtrace.go`'s file plumbing rather than
  opening its own (`ARCH-DRY`).
- **The reattachcost `sample` mode** (`PAIR_PROBE_SAMPLE_SECS=N`) creates only
  the control session and samples `zellij action query-tab-names` for N
  seconds, printing p50/p95/max/n.

## The pass's transitions (`ARCH-ORDER`)

| # | State | Event | Next state | Effects |
|---|---|---|---|---|
| 1 | Idle | `ArmReattach(root)` | Armed | none |
| 2 | Armed | Inventory, error | Armed | none |
| 3 | Armed | Inventory, ok | Running, Queue = (Detached ∪ resume-shaped unknown) − Root, sorted; empty → Idle | advance |
| 4 | Running | Inventory, ok or error | unchanged. **The queue is never pruned or extended.** | advance |
| 5 | Running | Result for `Loading`, success | `Attached[X]`; head popped | advance |
| 6 | Running | Result for `Loading`, `resume-not-detached` or `resume-session-gone` | head popped, no mark | advance |
| 7 | Running | Result for `Loading`, other failure | `Failed[X] = code`; head popped | advance |
| 8 | Running | Result for another attempt | unchanged | none |
| 9 | Running | Operator resume of `Loading` | in-flight slot := the pass's attempt | none dispatched; the pass's own effects still returned |
| 10 | Running | Operator resume of a queued X | X removed from Queue | the ordinary resume effect |
| 11 | any | Operator resume of a failed X | `Failed[X]` cleared | the ordinary resume effect |
| 12 | Running | any operator operation in flight | **advance holds** | none until the slot clears |
| 13 | Running | Inventory showing X Live | `Attached[X]` cleared | advance |
| 14 | any | Operator archive/park/relaunch/rename of a queued X | X removed from Queue | the ordinary effect |
| 15 | any | Stop (leave, SIGHUP, SIGTERM, last pane) | the console ends | attempt in flight cancelled; queue dropped |

**Cell 12 is what makes quitting safe.** The queue worker starts the next
request as soon as it pushes the previous result, before `Run` has handled the
completion. Without the hold, a leave dispatched mid-pass would be followed by
one more reattach that is then cancelled. `advanceReattach` therefore emits
nothing while `InFlight.Operation != ""` and is retried when the slot clears.

**Cell 4 replaces pruning.** A failed `DetachedSessions` leaves every candidate
`unknown`, and a failed `list-sessions` reads as "no sessions at all", so one
bad refresh would otherwise empty a queue that is never refilled. The queue is
seeded once; `warm-only` re-proves each thread at attempt time, which is the
authority that cannot go stale.

**Extent.** One pass attempt in the queue at a time, on the queue's single
worker under `WithCancel(c.lifetime)`. The pass spawns no goroutine.
Worst-case wait for an operator operation behind one attempt: `StartBlocked`'s
10 s spawn bound plus zellij's 5 s query bound; the ordinary case is about
0.3 s in the probe, roughly 0.7 s on real detached sessions.

**Nondeterminism** enters at the Run loop's `select`. The reducer is pure, so
each order is reproduced by feeding events in that order; no test races to
reach a cell.

## Operating envelope (`ARCH-CONSTRAINTS`)

- **Keystroke path unchanged.** The pass holds no lock across IO and never
  takes the in-flight slot except by adoption.
- **Time to first frame (M1).** Before: `ResolveEstablished` per resume-shaped
  record plus C `list-clients`, plus the cwd resume. After: the cwd candidate
  and any layout-conflicting candidate only. M1 closes on both a zellij call
  count **and** a measured first-frame time, since a count alone would miss
  the ledger reads.
- **The whole pass** is N sequential reattaches at #228's constant
  `list-clients` cost — roughly 7 s for 10 threads, off the critical path.

## Trust (`ARCH-SECURE`)

- `warm-only` is `Implicit`: unreachable from the CLI, and its absence is
  today's behaviour.
- Addresses come from the proof-bearing inventory, and every attempt re-proves
  its thread inside the resume before any effect.
- `COUCH_TRACE` is the operator's own path, `O_APPEND|O_CREATE`, 0600,
  recording addresses and timings, never content.

---

## M1 — Startup asks only about the threads it can act on

### Task 1: the duplicate-name rule counts the whole index

**Files:**
- Modify: `cmd/internal/couchcore/detachedsessions.go`, `artifactcollision.go`
- Test: `cmd/internal/couchcore/detachedsessions_test.go`, `artifactcollision_zellij_test.go`

- [ ] **Step 1: red.** At `sandboxedChecker`: index two addresses onto one
  session name, then ask `DetachedSessions` about **one** of them. Assert no
  observation (the name proves nothing). Today it returns one.
- [ ] **Step 2:** `SessionNameBinding` gains the scope's full claim set, or
  `ProjectDetachedSessions` takes a `claims map[string]int` built by the
  caller from the index. Prefer the second: the pure function keeps taking
  data, and the IO shell keeps reading the index it already reads.
- [ ] **Step 3:** pure-level rows in `detachedsessions_test.go` for a claim
  count that exceeds the bindings passed.
- [ ] **Step 4:** `go test ./cmd/internal/couchcore/ -run 'Detached' -count=1`.

### Task 2: Red — startup's work is independent of the other threads

**Files:**
- Create: `cmd/internal/couchcore/startup_proof_test.go`

- [ ] **Step 1: count.** A detached thread at the cwd and `k` detached threads
  elsewhere, all in couch's layout, sessions indexed, host also holding
  `withOthers(…, 3)`. `list-clients` is 3 at k=2 and k=12 (startup's proof,
  `DetachedSessions`, `confirmStillDetached`).
- [ ] **Step 2: binding resolutions.** A counter on the fake resolver:
  `ResolveEstablished` is called only for the asked candidates, so it does not
  grow with k.
- [ ] **Step 3: the guard keeps its reach.** One other-path detached thread in
  a different layout → `StartInteractive` refuses, and that thread's session
  was asked.
- [ ] **Step 4: equivalence.** Over a record table, compute startup's rows
  both ways and assert `ResolveLayoutConflicts`, `SelectResumableRoot`,
  `PathHoldsUsableThread` and `PathHoldsUnreadableThread` answer identically.
  Rows: detached at cwd; parked at cwd; detached elsewhere same layout;
  detached elsewhere other layout; unreadable layout elsewhere; cwd session
  gone; **and a cwd thread sharing a session name with an unasked thread**
  (the case Task 1 makes safe), run at `sandboxedChecker` because the fake
  does not model the duplicate-name rule.
- [ ] **Step 5:** run; Steps 1, 2 and 4 fail.

### Task 3: Implement the narrowing

**Files:**
- Modify: `cmd/internal/couchcore/actionableinventory.go`, `startup.go`

- [ ] **Step 1:** `gatherThreadEvidence` gains `ask func(ThreadRecord) bool`
  (nil = every candidate, as today), applied **before** `ResolveEstablished`,
  reading `snapshot.Records[i]` after physicalization. A skipped candidate
  keeps `ProofUnresolved`. `ActionableThreadInventoryContext` passes nil.
- [ ] **Step 2:** add `startupAsks` beside `SelectResumableRoot`, its comment
  naming the four readers and the equivalence test.
- [ ] **Step 3:** `StartInteractive` builds its rows through
  `startupInventory(ctx, startupAsks(...))`, and the guard's comment says
  which candidates were asked and why that is exactly its set.
- [ ] **Step 4:** Tasks 1–2 green, then `go test ./cmd/internal/couchcore/`.

### Task 4: M1 measure, docs, mutations, close

- [ ] **First-frame timing.** Add the `COUCH_TRACE` `startup`/`first-frame`
  events (Task 11's plumbing, pulled forward) and record the time with a
  store holding ≥10 threads, before and after, with co-tenancy.
- [ ] `atlas/couch.md`: startup proves the cwd candidate and any
  layout-conflicting candidate, and nothing else.
- [ ] **Mutation sweep**, each killed by name: `ask` ignored; the layout arm
  dropped; the path arm dropped; `ask` applied after `ResolveEstablished`
  (Step 2's counter); `WorkingPath` compared before physicalization; the
  claim count reverted to the candidate list (Task 1).
- [ ] Unsandboxed `make test`; `sdlc milestone-close --issue 206 --milestone M1`.

## M2 — The background reattach pass (after #230 merges)

### Task 5: `warm-only` resume

**Files:** `couchcore/resume.go`, `ops.go`, `operationdispatch.go`; test `warmresume_test.go`

- [ ] **Step 1: red.** `TestWarmOnlyResumeNeverStartsAnAgent`, a table:
  verified-parked; parked **with a provisional binding** (so a late refusal
  would surface a binding error instead of the warm-only code); detached-shaped
  with its session gone; detached-shaped attached elsewhere. Each expects
  `ResumeDiagnosticOf(err) == ResumeNotDetached`, an unchanged revision, and
  no child. A genuinely detached thread still resumes warm.
- [ ] **Step 2:** add `ResumeNotDetached`; add `ResumeOptions` to
  `ResumeContext`; refuse `WarmOnly && VerifiedPark != nil` **before**
  `resumeEvidence`, and `WarmOnly && !detached` before `CommitStartClaim`.
- [ ] **Step 3:** declare the `warm-only` implicit arg; pass it from
  `operationdispatch.go`.
- [ ] **Step 4:** `couchcmd` tests — update `TestOperationArityMatchesExpectation`
  (resume gains an argument) and assert `couch resume <tag> --warm-only`
  fails with "unknown flag".
- [ ] **Step 5:** `go test ./cmd/internal/couchcore/ ./cmd/internal/couchcmd/`.

### Task 6: the pure pass

**Files:** create `couchtty/menu_reattach.go`, `menu_reattach_test.go`

- [ ] **Step 1: red.** `TestReattachPassTransitions`, one row per cell 1–14,
  each applying one event through `ReduceMenu` and asserting the next
  `Reattach` and the effects. Named extras:
  `TestReattachPassNeverExtendsItsQueue` (cell 4, including an inventory whose
  rows all read `session-gone`), `TestReattachPassExcludesTheRoot`,
  `TestReattachPassOrdersMostRecentFirst`,
  `TestReattachPassHoldsWhileAnOperatorOperationIsInFlight` (cell 12).
- [ ] **Step 2:** implement `seedReattach`, `advanceReattach` (holds on cell
  12), `finishReattach`, `claimReattach`, and `clearAttachedWhenLive`. All
  take and return `MenuState` by value, following the clone discipline; add
  the deep copies to `cloneMenuState`.

### Task 7: route the pass through `ReduceMenu`

**Files:** modify `couchtty/menu.go`

- [ ] **Step 1: red.** Pure tests: Enter adopts on the loading row (and the
  adopted branch **still returns the pass's own effects**, so the pass does
  not stall); click adopts through `MenuEventMouseSwitch`; Enter jumps on a
  queued row; a background result that matches the adopted slot clears it and
  advances; one that does not leaves `InFlight` and the notice alone; a
  background success sets `ProjectionPending` so the row is not read as stale.
- [ ] **Step 2:** `MenuEventInventory` seeds (Armed) or clears `Attached`
  (Running), then advances, returning the effects. `MenuEventOperationResult`
  with `event.Background` routes to `finishReattach` **before** the in-flight
  early return, and continues into `reduceOperationResult` only when adopted.
- [ ] **Step 3:** `dispatchMenuOperation` calls `claimReattach` for a resume,
  after the in-flight refusal check so a refused dispatch claims nothing.
  Enter, click and the actions menu all reach it.
- [ ] **Step 4:** the actionable check that gates Enter and click consults the
  pass first, so the loading row stays selectable while the inventory lags.
- [ ] **Step 5:** add `MenuEventReattachArm`, `MenuEvent.Background`,
  `MenuEvent.Diagnostic`.

### Task 8: console wiring

**Files:** create `couchtty/console_reattach.go`; modify `console.go`, `console_menu.go`; test `console_reattach_test.go`

- [ ] **Step 1: red.** Fixture tests:
  1. the first inventory produces exactly one queued resume with
     `warm-only=true`; completing it attaches a pane and queues the next;
  2. **a background attach never takes focus** — drive `onExit` of the last
     pane with the switcher focused (so `c.active == ""`), then complete a
     background attempt: `c.focus` is still the panel and `c.tracker` was not
     seeded;
  3. an operator switch waits behind at most the running attempt;
  4. adoption lands focus, with one attach rather than two;
  5. a failure marks the row and the pass continues;
  6. `Stop` mid-attempt cancels it (via `SetOperationDispatcher`) and runs no
     further attempt;
  7. an unarmed console emits no background effect.
- [ ] **Step 2:** `ArmReattachPass`; `finishMenuRefresh` and `finishOperation`
  dispatch the effects `ReduceMenu` returns (both discard them today, and
  neither event kind produced any before, so nothing else starts dispatching).
- [ ] **Step 3:** `runMenuOperation`'s background branch goes **first**,
  before the attention-capture block and the no-dispatcher path, both of which
  address the operator's in-flight slot. Key: `reattach\x00<attempt>`.
- [ ] **Step 4:** `installObservedThreadActor` takes `background` and skips
  focus and tracker seeding. `finishOperation` computes `adopted` under `c.mu`
  before reducing; the focus steal becomes `origin.Operation == "resume" &&
  err == nil && startedHandleID != "" && (!origin.Background || adopted)`.
- [ ] **Step 5:** `TestAReattachedChildKeepsItsTrackingMode` and the whole
  `./cmd/internal/couchtty/` suite still pass.

### Task 9: rendering

**Files:** modify `couchtty/menu_render.go`; test `menu_render_test.go`

- [ ] **Step 1: red.** Queued → `queued`; loading → `reattaching…`; failed →
  `reattach failed: <code>`; attached-but-not-yet-in-inventory → `live`;
  a Detached row outside the pass → `detached · <age>`.
- [ ] **Step 2:** implement, reading the pass **before** the inventory state
  for rows the pass owns. Keep the vocabulary guard green.
- [ ] **Step 3:** the `reattaching…` row is static unless a progress notice is
  showing (the spinner only advances then) — either render it without a
  spinner or drive the tick; decide and state which.

### Task 10: `couchcmd` arms the pass

**Files:** modify `couchcmd/run.go`; test `run_test.go`

- [ ] `runConsole` arms with `start.Record.Thread` after
  `dispatchInitialAttach` succeeds, never when it fails.

### Task 11: the trace

**Files:** modify `couchtty/inputtrace.go`, `couchcmd/run.go`

- [ ] Extend the existing trace plumbing with the five events; `couchcmd`
  reads `COUCH_TRACE`. (Pulled forward for M1's timing; only the pass events
  land here.)

### Task 12: measure, document, smoke, close

- [ ] Add `PAIR_PROBE_SAMPLE_SECS` to `cmd/probes/reattachcost`, bounded like
  the rest of the probe.
- [ ] **Measurement (operator-assisted: couch needs a real terminal).** With
  N detached threads and co-tenancy recorded: run the sampler for 30 s while
  the operator starts `COUCH_TRACE=… couch`. Record first-frame time, pass
  duration, per-attempt times, and `zellij action` p50/p95/max against the
  quiet baseline.
- [ ] `atlas/couch.md`: the pass under the switcher's operation model (its
  decisions, not its cells), plus the `COUCH_TRACE` format. Fix the stale
  "Resume an exact verified-parked work thread" summary.
- [ ] **Mutation sweep:** one row per cell with a named test, plus the focus
  steal ignoring `Background`; `installObservedThreadActor` seeding focus on a
  background attach; the pass emitting without `warm-only`; cell 12's hold
  removed; the Root exclusion removed; the adopted branch returning nil
  effects.
- [ ] Unsandboxed `make test`.
- [ ] **Operator smoke.** `make install`, restart couch with several detached
  threads: they come back unasked; the cwd thread is typeable at once; a
  switch to a queued row lands on it; quit mid-pass leaves the rest detached.
- [ ] **Ask the operator** to accept or reject the Done-when deviation above.
- [ ] `sdlc close --issue 206`.

## Estimate

Derived at `sdlc change-code`, after plan-quality.
