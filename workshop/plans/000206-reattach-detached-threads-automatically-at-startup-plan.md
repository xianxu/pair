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

**#230 is merged (PR #124), so M2 is unblocked.** It was the prerequisite: six
post-acknowledgement failure routes deleted the zellij session a warm reattach
was reattaching. The route a background reattach actually takes is #230's route
6 -- a warm reattach's registration check is satisfied by the pre-existing
session, so a `pair resume` client that dies early is first noticed when the
console's attach fails, which calls `AbortStarted`. A failed background
reattach now leaves its thread detached and reattachable.

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
(`launcher.RequireAttachState`). **Put to the operator before M2 starts, not
at the close** — it is a Done-when bullet, and building M2 against an
interpretation they have not accepted is the expensive order to discover a
disagreement in.

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
9. **`couch resume <tag>` in a terminal does NOT arm the pass.** It runs
   through `runConsole` like a bare `couch`, so it would otherwise inherit the
   pass. Arming belongs to the startup gesture: an operator who named one
   thread asked for that thread. Only the bare-start path arms.
10. **Cell 14 covers archive, park and relaunch, not rename.** Renaming a
   queued thread changes its label, not its resumability, so its queue entry
   stays — the pass holds addresses, not rows. Done-when bullet 2 is satisfied
   because the thread is still reattached.
11. **Every pass failure carries a diagnostic code.** `ResumeDiagnosticOf`
   returns empty for failures that are not refusals (a spawn error, a
   registration timeout), so the row would read `reattach failed: `. Those get
   a generic `reattach-failed` code, and the row shows the error's first line
   beneath it.
12. **Rows are rendered from the pass while its own mutation is in flight**
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

- **`ProjectDetachedSessions`'s duplicate-name rule counts claims over the
  scope's whole index, not over the caller's candidate list.** It used to count
  `claims[name]++` across the bindings passed in. Two addresses bound to one
  session name is its fail-closed case, and narrowing the candidate list
  silently removed that protection: ask about one of the two and its name looks
  unique, so a thread whose binding is contested is reported Detached and
  startup resumes it. A latent weakness #228's narrowing introduced; M1 depends
  on it and fixes it first.
  - **The unit of counting is a THREAD, at its effective binding.** The index
    is append-only and merged across files, so a thread has many entries and
    only its NEWEST one binds. `effectiveBindings` reduces the reads to one
    current name per address, and `claimsFromBindings` counts distinct addresses per name. Counting
    raw lines would be wrong in both directions: a thread that re-registered
    under one name would contest its own session, and a name a thread has since
    moved off would contest the thread that holds it now. Both are pinned
    (`TestDetachedSessionsCountsAThreadOnce...`,
    `TestDetachedSessionsIgnoresANameItsThreadHasLeft`).
  - **The fake goes through the same function**, rather than answering from its
    own map, so the rule is not invisible to fake-backed tests (`ARCH-MOCK`).
- **`startupAsks(requested Layout, repoScope, workingPath string) func(ThreadRecord) bool`**
  decides which resume-shaped candidates startup's blocking inventory
  resolves. It is the union of what the readers of startup's rows filter on:
  - `SelectResumableRoot`, `PathHoldsUsableThread` and
    read only rows at the cwd (`Address.RepoScope == repoScope &&
    WorkingPath == workingPath`). `PathHoldsUnreadableThread` scans the whole
    scope rather than one path, but only for records the store could not
    decode -- which never reach the resume-shaped branch this predicate gates,
    so it is unaffected;
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
      Attached       map[couchcore.ThreadAddress]uint64   // landed -> the refresh generation current at the time
      Failed         map[couchcore.ThreadAddress]string   // diagnostic code per row
  }
  ```

  Attempt identities come from `MenuState.OperationSequence`, so an adopted
  attempt matches the in-flight slot through the existing
  `menuOperationMatches`. `cloneMenuState` deep-copies `Queue`, `Attached` and
  `Failed`, keeping a nil map nil, because tests compare whole `MenuState`
  values with `DeepEqual`.
  - **`Attached`** is what makes cell 13 work: the pass, not the lagging
    inventory, is the authority for a row it just mutated. It maps an address
    to the refresh GENERATION current when its attach landed, and an entry is
    dropped by the first inventory admitted after that generation — the same
    idiom `ProjectionAfterGeneration` already uses.
    - Expiring on "an inventory shows it Live" would strand the row: a pane
      that exits before the next refresh is never Live in any inventory, so the
      row would render `live` forever, for a thread that is gone. A generation
      bound is one refresh whatever happened to the pane.
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
  - **The channel for that flag is a declared `background` arg on the `attach`
    operation**, `Implicit` like `tag` (so the CLI cannot send it), read by
    `ExecuteConsoleOperation` and passed through. `finishOperation` reaches
    `installObservedThreadActor` only by dispatching `attach` through the
    declared table — it holds no direct call — so without this arg the flag has
    no way across, and the guard above cannot be reached. `finishOperation`
    sets it from `completed.origin.Background && !adopted`: an ADOPTED
    completion is the operator's own landing and takes focus normally.
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

## The pass view: one authority, every reader (`ARCH-DRY`, `ARCH-PURPOSE`)

While the pass owns a row, the inventory is BEHIND — a thread mid-reattach
reads Detached, then stale-incarnation, then Busy, then Live-but-unhosted, and
an attached one reads Detached until the next refresh lands. Five readers ask
about a row, and every one of them that consults the inventory first gets a
stale answer:

| reader | stale answer without the view |
|---|---|
| `rootStateText` | `detached · 3m` for a row that is loading, `live` for one that is gone |
| `reduceRootKey` (Enter) | verb `switch` on a loading row, so adoption never fires; a second `resume` on an attached one, which `DecideResume` refuses as occupied |
| `MenuEventMouseSwitch` (click) | the same, by the same route |
| `menuThreadActionable` | refuses Enter entirely while the row reads Busy or stale |
| `menuActionItems` (Tab) | returns its Busy branch before the actionable check |

So there is **one pure function**, `passViewOf(pass, address) (PassView, bool)`,
the only place that knows what the pass means for a row.

**It is applied where rows are LOOKED UP, not in each reader.** The table above
names five readers, but that list is not the class: the confirmation's Enter
checks `!thread.Live()`, reconcile makes the same check, the leave
confirmation counts live rows, Escape reads state, and the age colouring reads
`LastActiveAt`. Patching five named functions leaves a sixth reading the stale
row -- and three plan-review rounds in a row found another one. So the view is
overlaid inside the lookups every reader already goes through
(`findMenuThread`, `selectedMenuThread`, `visibleRootThreads`), which return
a row with the pass's state applied.

**And a guard makes "a sixth reader fails" true rather than hoped for:** a test
that parses `couchtty`'s non-test sources and fails on any read of
`state.Inventory` outside those lookups. Without the guard the lookup
convention is a comment; with it, a new reader that goes around the view fails
the build's tests.

The view's contract, per pass state:

| pass state | text | Enter/click verb | actionable | Tab items |
|---|---|---|---|---|
| Queued | `queued` | `resume` (jumps the queue) | yes | resume, park, archive |
| Loading | `reattaching…` | **adopt** — claim the attempt, dispatch nothing | yes | name, describe (the Busy items) |
| Failed | `reattach failed: <code>` | `resume` (and clears the mark) | yes | resume, park, archive |
| Attached | `live` | `switch` | yes | the live-row items |
| not in the pass | today's `rootStateText` | today's `enterOperationFor` | today's rule | today's items |

**Post-conditions every overridden reader keeps**, because the callers rely on
them and a violation is a crash on the Run loop, not a wrong row:
- **Tab items are never empty.** `reduceRootKey`'s Tab branch does
  `SelectedItem: items[0]` unguarded, and every branch of `menuActionItems`
  returns at least two items today. An earlier draft of this table gave the
  Loading row none, which would have panicked the Run loop and taken couch down
  on one keypress. Loading therefore gets the Busy items, which are the ones
  that make sense while an operation is in flight.
- **The verb is one `enterOperationFor` could return** ("resume" or "switch"),
  or the adopt sentinel the reducer handles before dispatch.

The generated-sequence test's alphabet includes Tab and actions-menu selection,
so a future empty list fails there rather than at the operator's keyboard.

Two consequences worth stating, because they are what the table buys:
- **Adoption can actually fire.** It hangs off the resume verb, and the
  loading row is never `Resumable()` in the inventory, so without the view the
  one row adoption exists for would always dispatch `switch`.
- **An attached row is switchable immediately**, rather than offering a resume
  that couchcore refuses. `Attached` expires on a generation (cell 13), so the
  view yields back to the inventory within one refresh whatever happened to the
  pane.

The generated-sequence test (Task 6 Step 0) drives Enter and click through this
table rather than asserting rendered strings, so a reader added later that
forgets the view is a failing invariant, not a visual regression nobody notices.

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
| 13 | Running | Inventory newer than `Attached[X]`'s generation | `Attached[X]` dropped, whatever that inventory says about X | advance |
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

### Task 1: the duplicate-name rule counts the whole index — DONE

**Files:**
- Modify: `cmd/internal/couchcore/detachedsessions.go`, `artifactcollision.go`, `artifactcollision_fake.go`
- Test: `detachedsessions_test.go`, `artifactcollision_zellij_test.go`, `artifactcollision_fake_test.go`

- [x] **Step 1: red.** At `sandboxedChecker`: two addresses indexed onto one
  session name, then `DetachedSessions` asked about **one** of them. It
  returned an observation; it must return none
  (`TestDetachedSessionsRefusesANameTwoThreadsClaim`).
- [x] **Step 2:** `ProjectDetachedSessions` takes `claims map[string]int`,
  built by the IO shell from the index it already reads. The pure function
  keeps taking data.
- [x] **Step 3: the unit of counting is a THREAD at its effective binding.**
  `effectiveBindings` reduces the reads to one CURRENT name per address --
  entries are append-only and merged, so a thread's binding is its newest
  entry -- then counts distinct addresses per name. Counting raw lines is
  wrong in both directions, and both are pinned:
  - a thread that re-registered under one name would contest its own session
    (`TestDetachedSessionsCountsAThreadOnceHoweverOftenItRegistered`);
  - a name a thread has since MOVED OFF would contest the thread that holds it
    now (`TestDetachedSessionsIgnoresANameItsThreadHasLeft`) — the case that
    survived two mutation attempts before it was written.
- [x] **Step 4: the fake goes through the same function** rather than its own
  map, so the rule is not invisible to fake-backed tests
  (`TestFakeDetachedSessionsAppliesTheDuplicateNameRule`, `ARCH-MOCK`).
- [x] **Step 5:** pure rows for a claim count exceeding the bindings passed
  (`TestProjectDetachedSessionsRefusesAContestedName`); three mutations killed
  as named.

### Task 2: Red — startup's work is independent of the other threads — DONE

**Files:**
- Create: `cmd/internal/couchcore/startup_proof_test.go`

- [x] **Step 1: count.** A detached thread at the cwd and `k` detached threads
  elsewhere, all in couch's layout, sessions indexed, host also holding
  `withOthers(…, 3)`. `list-clients` is 3 at k=2 and k=12 (startup's proof,
  `DetachedSessions`, `confirmStillDetached`).
- [x] **Step 2: binding resolutions.** A counter on the fake resolver:
  `ResolveEstablished` is called only for the asked candidates, so it does not
  grow with k.
- [x] **Step 3: the guard keeps its reach.** One other-path detached thread in
  a different layout → `StartInteractive` refuses, and that thread's session
  was asked.
- [x] **Step 4: equivalence.** Over a record table, compute startup's rows
  both ways and assert `ResolveLayoutConflicts`, `SelectResumableRoot`,
  `PathHoldsUsableThread` and `PathHoldsUnreadableThread` answer identically.
  Rows: detached at cwd; parked at cwd; detached elsewhere same layout;
  detached elsewhere other layout; unreadable layout elsewhere; cwd session
  gone; **and a cwd thread sharing a session name with an unasked thread**
  (the case Task 1 makes safe), run at `sandboxedChecker` because the fake
  does not model the duplicate-name rule.
- [x] **Step 5:** run; Steps 1, 2 and 4 fail.

### Task 3: Implement the narrowing — DONE

**Files:**
- Modify: `cmd/internal/couchcore/actionableinventory.go`, `startup.go`

- [x] **Step 1:** `gatherThreadEvidence` gains `ask func(ThreadRecord) bool`
  (nil = every candidate, as today), applied **before** `ResolveEstablished`,
  reading `snapshot.Records[i]` after physicalization. A skipped candidate
  keeps `ProofUnresolved`. `ActionableThreadInventoryContext` passes nil.
- [x] **Step 2:** add `startupAsks` beside `SelectResumableRoot`, its comment
  naming the four readers and the equivalence test.
- [x] **Step 3:** `StartInteractive` builds its rows through
  `startupInventory(ctx, startupAsks(...))`, and the guard's comment says
  which candidates were asked and why that is exactly its set.
- [x] **Step 4:** Tasks 1–2 green, then `go test ./cmd/internal/couchcore/`.

### Task 4: M1 measure, docs, mutations, close

- [ ] **Timing evidence is the operator's smoke, not a new trace.** M1's
  promise is the counted invariant, which Task 2 already asserts at two store
  sizes (`workbench-latency`: prefer counts to timings). The `COUCH_TRACE`
  facility stays in M2 Task 11, where the background pass needs it; building
  it here only for M1 would be scope with no reader. The operator installs M1,
  restarts couch with ≥10 threads in the store, and reports whether startup
  feels faster -- which is the complaint M1 answers.
- [ ] `atlas/couch.md`: startup proves the cwd candidate and any
  layout-conflicting candidate, and nothing else.
- [ ] **Mutation sweep**, each killed by name: `ask` ignored; the layout arm
  dropped; the path arm dropped; `ask` applied after `ResolveEstablished`
  (Step 2's counter); `WorkingPath` compared before physicalization; the
  claim count reverted to the candidate list (Task 1).
- [ ] Unsandboxed `make test`; `sdlc milestone-close --issue 206 --milestone M1`.

## M2 — The background reattach pass

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
- [ ] **Step 3:** declare the `warm-only` implicit arg in `ops.go`; pass it
  from `operationdispatch.go`.
- [ ] **Step 4:** `couchcmd` tests — update `TestOperationArityMatchesExpectation`
  (resume gains an argument) and assert `couch resume <tag> --warm-only`
  fails with "unknown flag".
- [ ] **Step 5:** `go test ./cmd/internal/couchcore/ ./cmd/internal/couchcmd/`.

### Task 6: the pure pass

**Files:** create `couchtty/menu_reattach.go`, `menu_reattach_test.go`

- [ ] **Step 0: the invariants, as properties over generated event
  sequences.** The hand-written cells below each sample one interleaving;
  these hold over all of them. Drive a few thousand random sequences of
  {inventory ok, inventory error, completion success/refusal/failure, operator
  resume, operator op start/finish, arm, tick} through `ReduceMenu` and assert
  after every step: at most one `Loading`; `Root` never in `Queue`; `Queue`
  never longer than at seeding; no effect emitted while `InFlight.Operation !=
  ""`; and `Queue`, `Attached` and `Failed` pairwise disjoint.
  - **Enter and click go through `passViewOf`, not through rendered text.**
    After each step, for every address the pass owns, assert the verb the
    reducer would pick equals the view's — which is what makes a stale reader
    a failing invariant rather than a visual regression.
- [ ] **Step 1: red.** `TestReattachPassTransitions`, one row per cell 1–15,
  each applying one event through `ReduceMenu` and asserting the next
  `Reattach` and the effects. Named extras:
  `TestReattachPassNeverExtendsItsQueue` (cell 4, including an inventory whose
  rows all read `session-gone`), `TestReattachPassExcludesTheRoot`,
  `TestReattachPassOrdersMostRecentFirst`,
  `TestReattachPassHoldsWhileAnOperatorOperationIsInFlight` (cell 12).
- [ ] **Step 2:** implement `seedReattach`, `advanceReattach` (holds on cell
  12), `finishReattach`, `claimReattach`, `expireAttached`, and `passViewOf`. All
  take and return `MenuState` by value, following the clone discipline; add
  the deep copies to `cloneMenuState`.

### Task 7: route the pass through `ReduceMenu`

**Files:** modify `couchtty/menu.go`

- [ ] **Step 1: red.** Pure tests: Enter adopts on the loading row (and the
  adopted branch **still returns the pass's own effects**, so the pass does
  not stall); click adopts through `MenuEventMouseSwitch`; Enter jumps on a
  queued row; a background result that matches the adopted slot clears it and
  advances; one that does not leaves `InFlight` and the notice alone.
  **A background success does NOT set `ProjectionPending`.** No row reader
  consults it -- only the notice line and the row budget do -- so setting it
  would put "refresh pending" on the operator's notice line for the whole pass,
  while `Attached` is what actually keeps the row correct.
- [ ] **Step 2:** `MenuEventInventory` seeds (Armed) or clears `Attached`
  (Running), then advances, returning the effects. `MenuEventOperationResult`
  with `event.Background` routes to `finishReattach` **before** the in-flight
  early return, and continues into `reduceOperationResult` only when adopted.
- [ ] **Step 3:** `dispatchMenuOperation` calls `claimReattach` for a resume,
  after the in-flight refusal check so a refused dispatch claims nothing.
  Enter, click and the actions menu all reach it.
- [ ] **Step 4:** **apply `passViewOf` inside the row lookups** —
  `findMenuThread`, `selectedMenuThread`, `visibleRootThreads` — so every
  reader gets the pass's view without knowing the pass exists. Then add the
  guard: a test parsing `couchtty`'s non-test sources that fails on any read of
  `state.Inventory` outside those lookups. The guard is what stops a future
  reader from going around the view.
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
- [ ] **Step 4:** declare the `background` implicit arg on `attach` in
  `ops.go` (and bump the expected attach arity from 2 to 3 in
  `couchcmd/run_test.go`'s `TestOperationArityMatchesExpectation`).
  `ExecuteConsoleOperation` reads it; `installObservedThreadActor` takes
  `background` and skips focus and tracker seeding. `finishOperation` computes `adopted` under `c.mu`
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

### 2026-09-11 — M1's timing evidence: operator smoke, not a pulled-forward trace

**Reason.** Task 4 pulled M2's `COUCH_TRACE` forward to time M1's first frame.
That adds a trace facility M1 has no other use for, touching `couchtty` and
`couchcmd` for a milestone that otherwise lives in `couchcore`.

**Delta.** M1 closes on the counted invariant, which Task 2 asserts at two
store sizes, plus the operator's real-stack smoke. The trace stays in M2 Task
11, where the pass's per-attempt timings need it.

### 2026-09-11 — plan gate round 3: the union of reads, a crash, and the lookup rule

**Reason.** Two blocking findings and a third instance of a recurring class.

- **PQ-1, a real bug in the Task 1 code (not the plan).** `DetachedSessions`
  reads the index once per scope, and every scoped read replays the shared
  legacy file before its own rows. Task 1 summed per-read claim counts, so a
  thread bound only by a legacy row counted once PER SCOPE ASKED: with two or
  more scopes, its own session read as contested, it classified session-gone,
  and the pass would never seed it. The reviewer measured 56 legacy-only
  bindings on the operator's host. Now `effectiveBindings` merges over the
  union of reads -- a thread's own-scope read is authoritative, and a
  legacy-only row is identical in every read -- and `claimsFromBindings` counts
  each thread once. The fake shares `claimsFromBindings`.

  **The test that mattered was the second one.** A pure test of
  `effectiveBindings` passed, and reintroducing the summed-per-read bug in
  `DetachedSessions` SURVIVED it, because every other seam test reads one
  scope, where summing and not summing agree.
  `TestDetachedSessionsCountsALegacyThreadOnceAcrossScopes` asks about two
  scopes in one call with a legacy-bound thread, and that one kills the
  mutation.
- **PQ-11, a crash.** The pass view gave the Loading row no Tab items, and
  `reduceRootKey`'s Tab branch indexes `items[0]` unguarded -- a panic on the
  Run loop, taking couch down on one keypress. Loading now gets the Busy items,
  and the table states the post-conditions every overridden reader keeps.
- **The reader class, third instance.** Three rounds each found another reader
  bypassing the pass. The rule is now applied where rows are looked up, with a
  source-parsing guard that fails on any `state.Inventory` read outside those
  lookups -- which is what makes "a sixth reader fails" true.
- **Dropped:** setting `ProjectionPending` on a background success. No row
  reader consults it; it would only have put "refresh pending" on the notice
  line for the whole pass.

### 2026-09-11 — M1 milestone review: what the tests actually cover

**Reason.** The M1 review (FIX-THEN-SHIP) found the equivalence test shipped
four of this plan's seven rows and three of its four readers, plus three places
the plan's prose had not followed the code.

**Delta.**
- **The equivalence test now has all seven rows and all four readers**,
  including `PathHoldsUnreadableThread`, and compares layout conflicts by WHICH
  addresses conflict rather than how many. The added rows are the cwd thread
  parked, the cwd thread's session gone, and the cwd thread sharing its session
  name with an unasked thread.
- **The shared-name row runs through the fake, not `sandboxedChecker` as Task 2
  Step 4 said.** That was written before the fake answered through production's
  `ProjectDetachedSessions` and `claimsFromBindings` (PQ-8). It does now, so the
  row composes the narrowing with the duplicate-name rule end to end. The
  index-file merge that feeds those claims in production is pinned at the real
  checker by `TestDetachedSessionsCountsALegacyThreadOnceAcrossScopes`.
- **The "conflicting layout" row tested nothing new.** It used `layout1`, which
  `ParseLayoutMode` rejects, so it was the unreadable row twice. It uses
  `layout3` now, and the refusal test covers both a valid different layout and
  an unreadable one.
- **An equivalence test cannot catch a predicate that asks too much**, because
  over-asking never changes an answer, only its cost. A mutation dropping the
  scope half of the cwd arm survived it for exactly that reason.
  `TestStartupDoesNotProveAForeignScopeRecordAtTheCwdPath` is a count, and it
  kills the mutation.
- `sessionNameClaims` is now `effectiveBindings` plus `claimsFromBindings`;
  Tasks 2 and 3 are ticked.
