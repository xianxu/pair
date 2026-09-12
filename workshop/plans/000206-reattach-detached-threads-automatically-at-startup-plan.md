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
  - A background attach adds its pane without touching focus.

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
  a thread it owns is not selectable, so the operator cannot dispatch a second
  resume of it from the switcher; and `DecideResume` still refuses an occupied
  incarnation for any other route.
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
(`launcher.RequireAttachState`). **Decided by the operator before M2 started:
first inventory is fine** (2026-09-11). No first-frame seeding.

## Decisions (the Spec's open cells, answered)

**The operator's UX for pending threads (2026-09-11), which decisions 3-5
implement:** *"when we attempt to start a thread, we would add placeholder of
it in the couch status bar, with a spinner ... it's not clickable ... if user do
search in switcher, same thing, that line is grayed, and not selectable."* And,
asked about queued threads: **all pending, none selectable.**

1. **Order.** Most recently active first (`LastActiveAt` descending), ties by
   `(RepoScope, Tag)`.
2. **What is reattached.** Seeded from rows that are `ThreadDetached` **or**
   resume-shaped `unknown`, minus the startup root, which the pass subtracts
   explicitly. `unknown` is included because it means "the proof could not be
   asked", not "not detached" (see cell 4). `warm-only` decides at attempt
   time, so an `unknown` row that is not really detached is skipped, not
   started.
3. **Every pending thread appears at once, as a placeholder, and none is
   selectable.** When the pass seeds, each thread it will reattach shows up
   greyed in the status bar and in the switcher; the one currently starting
   carries a spinner. A placeholder cannot be clicked (it records no chip span)
   and a pending switcher row cannot be selected (the cursor and auto-select
   skip it). There is no queue-jumping and no adoption: since #228 each
   reattach takes about 0.3-0.7 s, so ten threads fill in within seconds, and
   the operator chose simplicity over reordering.
4. **A placeholder resolves when its thread does.** On success it becomes an
   ordinary chip and row, in place: attached chips are drawn in attach order
   and placeholders after them in pass order, so the thread that just attached
   takes the column its placeholder held. On a skip (cell 6) the placeholder
   disappears and the row returns to whatever the inventory says.
5. **A reattach that fails** drops its placeholder from the status bar, and its
   switcher row reads `reattach failed: <code>` -- no longer pending, so
   selectable. Enter on it is an ordinary manual resume and clears the mark.
6. **A thread that is no longer warm** when its turn comes -- parked, attached
   elsewhere, or its session gone -- is **skipped silently**, not marked
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
10. **Cell 14 covers archive, park and relaunch, not rename.** A pending row is
   not selectable in the switcher, so these reach a queued thread only from the
   CLI (`couch park <tag>`); the queue entry goes, and the ordinary effect runs.
   Renaming changes a label, not resumability, so the entry stays.
11. **Every pass failure carries a diagnostic code.** `ResumeDiagnosticOf`
   returns empty for failures that are not refusals (a spawn error, a
   registration timeout), so the row would read `reattach failed: `. Those get
   a generic `reattach-failed` code, and the row shows the error's first line
   beneath it.
12. **Rows are rendered from the pass while its own mutation is in flight**
   (cell 13), because the inventory passes through states -- stale incarnation,
   then Busy, then Live-but-unhosted -- that would otherwise show as
   "stale..."/"parking..." for a thread the pass is still bringing back.

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
| `StatusActor.Placeholder`, `RenderStatusRow` (no chip span for a placeholder) | `cmd/internal/couchtty/reserve.go` | modified |
| `menuRowSelectable` (cursor and auto-select skip pending rows) | `cmd/internal/couchtty/menu.go` | new |
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
  resolves. It is the union of what the readers of startup's rows filter on.
  **The reader list lives on `startupAsks`' doc comment in `startup.go`** and
  is not restated here: every restated copy of it drifted (M1 review, round 2).

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

  Attempt identities come from `MenuState.OperationSequence`, the counter the
  operator's own operations draw from, so a pass completion can never be
  mistaken for one of theirs; `finishReattach` matches it against
  `LoadingAttempt`. There is no adoption, so a pass attempt never occupies the
  operator's in-flight slot. `cloneMenuState` deep-copies `Queue`, `Attached`
  and `Failed`, keeping a nil map nil, because tests compare whole `MenuState`
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
| `paintNow` (placeholder chips from the pass) | `couchtty/console.go` | modified | status bar model |
| status-row spinner tick | `couchtty/console.go` (Run loop) | new | a timer, repaint |
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
    sets it from `completed.origin.Background`, and a background completion
    NEVER takes focus: with no adoption, there is no background completion
    that is the operator's own landing.
- **The status-row spinner needs its own tick.** The existing spinner runs
  only while the switcher is focused AND a progress notice shows, but the
  operator spends the pass in their own thread. So the Run loop gets a second
  timer, armed only while the pass has a `Loading` thread, that advances a
  status-row spinner phase and requests a repaint of that row -- coalesced
  through the existing `paintPending`, and stopped with the console. It is the
  same seam #231's clock needs (a periodic status-row repaint while the
  operator is in a thread); #231 extends it rather than adding another.
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

While the pass owns a row, the inventory is BEHIND -- a thread mid-reattach
reads Detached, then stale-incarnation, then Busy, then Live-but-unhosted, and
an attached one reads Detached until the next refresh lands. Every reader that
consulted the inventory first would get a stale answer, and three plan-review
rounds each found another such reader.

So there is **one pure function**, `passViewOf(pass, address) (PassView, bool)`,
the only place that knows what the pass means for a row.

**It is applied where rows are LOOKED UP, not in each reader.** The view is
overlaid inside the lookups every reader already goes through
(`findMenuThread`, `selectedMenuThread`, `visibleRootThreads`), which return a
menu-local row carrying the pass's state alongside the inventory row.

**And a guard makes "a new reader fails" true rather than hoped for:** a test
that parses `couchtty`'s non-test sources and fails on any read of
`state.Inventory` outside those lookups.

The view's contract, per pass state -- this table is the contract's one home:

| pass state | switcher row | selectable | status bar | clickable |
|---|---|---|---|---|
| Queued | `queued`, greyed | no -- the cursor and auto-select skip it | the label, greyed | no |
| Loading | `reattaching...`, greyed, with the spinner | no | the label, greyed, with the spinner | no |
| Failed | `reattach failed: <code>` | yes -- Enter is an ordinary resume and clears the mark | nothing (not attached, not pending) | -- |
| Attached | `live` | yes -- Enter switches | the ordinary chip, from its pane | yes |
| not in the pass | today's | today's | today's | today's |

**Post-conditions, because the callers rely on them:**
- **A pending row is never the selection.** `menuRowSelectable` is the one
  predicate `moveRootSelection` and `reconcileRootSelection` both consult, so
  Enter, click and Tab never reach a pending row. That also keeps
  `reduceRootKey`'s unguarded `items[0]` out of reach for it.
- **If every visible row is pending** -- a filter that matches only pending
  threads -- there is no selection, and Enter reports "no selection" as it
  does today.
- **A placeholder records no chip span**, so `ColumnToActor` cannot resolve a
  click on it: unclickable by construction, not by a check at the click site.

The generated-sequence test (Task 6 Step 0) asserts after every step that no
pending address is the selection and that no Enter, click or Tab ever
dispatches for one -- so a reader added later that forgets the view is a
failing invariant, not a visual regression nobody notices.

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
| 9 | any | Operator resume of a failed X | `Failed[X]` cleared | the ordinary resume effect |
| 10 | Running | any operator operation in flight | **advance holds** | none until the slot clears |
| 11 | Running | Inventory newer than `Attached[X]`'s generation | `Attached[X]` dropped, whatever that inventory says about X | advance |
| 12 | any | Operator archive/park/relaunch of a queued X (CLI only) | X removed from Queue | the ordinary effect |
| 13 | any | Stop (leave, SIGHUP, SIGTERM, last pane) | the console ends | attempt in flight cancelled; queue dropped |

There is no "operator resume of a pending X" cell, because the view makes a
pending row unselectable: that event cannot arise from the switcher. The
generated-sequence invariants assert it does not.

**Cell 10 is what makes quitting safe.** The queue worker starts the next
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
worker under `WithCancel(c.lifetime)`. The pass spawns no goroutine; the
status-row spinner is a timer on the Run loop, armed only while a thread is
`Loading`. How long an operator operation can wait behind one attempt is
stated once, under "Operating envelope" below -- not restated here, because the
last restatement of it drifted (it named one zellij query where a warm attempt
makes two, PQ-6).

**Nondeterminism** enters at the Run loop's `select`. The reducer is pure, so
each order is reproduced by feeding events in that order; no test races to
reach a cell.

## Operating envelope (`ARCH-CONSTRAINTS`)

- **Keystroke path unchanged.** The pass holds no lock across IO and never
  takes the in-flight slot: a pass attempt is never the operator's operation.
- **Worst-case wait behind one reattach** -- the ONE statement of it; the
  transitions section's Extent points here rather than restating it. An
  operator operation queued behind a pass attempt waits for that attempt, and
  the attempt's bounds add up: `StartBlocked`'s 10 s spawn bound, the
  registration wait, and TWO `DetachedSessions` queries -- `ResumeContext`'s and
  `confirmStillDetached`'s -- each bounded at zellij's 5 s query timeout. About
  20 s. The ordinary case is about 0.3 s in the probe and roughly 0.7 s on real
  detached sessions. Task 12 measures the real distribution rather than
  asserting this. (Only switcher-dispatched operations queue; typing into the
  operator's own thread, ctrl+return and ctrl+backspace never do.)
- **Time to first frame (M1).** Before: `ResolveEstablished` per resume-shaped
  record plus C `list-clients`, plus the cwd resume. After: the cwd candidate
  and any layout-conflicting candidate only. M1 closed on the counted invariant
  -- zellij candidates AND binding resolutions, which covers the ledger reads a
  zellij count alone would miss -- plus the operator's real-stack smoke (~1.5x).
  A measured first frame waits for M2's `COUCH_TRACE`; see the M1 timing
  Revision.
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

### Task 5: `warm-only` resume -- DONE (`01de3aa7`)

- [x] `ResumeNotDetached`; `ResumeOptions{WarmOnly}` on `ResumeContextWith`,
  refusing a verified park BEFORE the binding is resolved and a thread with no
  detached session BEFORE `CommitStartClaim`; the `warm-only` implicit arg,
  dispatched through the operation table; the CLI refuses it as an unknown
  flag. Pinned directly AND through the operation table -- a direct-call test
  survived the dispatcher dropping the argument. 5 of 5 mutations killed.

### Task 6: the pure pass -- DONE

**Files:** create `couchtty/menu_reattach.go`, `menu_reattach_test.go`

- [x] **Step 0: the invariants, as properties over generated event
  sequences.** Drive a few thousand random sequences of {inventory ok,
  inventory error, completion success/refusal/failure, operator resume of a
  failed row, operator op start/finish, cursor up/down, filter keystroke,
  Enter, click, Tab, arm, tick} through `ReduceMenu` and assert after every
  step:
  - at most one `Loading`; `Root` never in `Queue`; `Queue` never longer than
    at seeding; `Queue`, `Attached` and `Failed` pairwise disjoint;
  - no effect emitted while `InFlight.Operation != ""`;
  - **no pending address is ever the selection, and no Enter, click or Tab
    ever dispatches for one.**
- [x] **Step 1: red.** `TestReattachPassTransitions`, one row per cell 1-13.
  Named extras: `TestReattachPassNeverExtendsItsQueue` (cell 4, including an
  inventory whose rows all read `session-gone`), `TestReattachPassExcludesTheRoot`,
  `TestReattachPassOrdersMostRecentFirst`,
  `TestReattachPassHoldsWhileAnOperatorOperationIsInFlight` (cell 10).
- [x] **Step 2:** implement `seedReattach`, `advanceReattach` (holds on cell
  10), `finishReattach`, `expireAttached`, `passViewOf`, and
  `pendingPlaceholders` (Loading then Queue, in pass order, for the status
  bar). All take and return `MenuState` by value; add the deep copies to
  `cloneMenuState`.

### Task 7: route the pass through `ReduceMenu` -- DONE

**Files:** modify `couchtty/menu.go`

- [x] **Step 1: red.** Pure tests: the cursor skips a pending row in both
  directions; a filter that leaves only pending rows selects nothing, and Enter
  then reports "no selection"; Enter on a failed row dispatches an ordinary
  resume and clears the mark; a background result advances the pass and leaves
  `InFlight` and the notice alone. **A background success does NOT set
  `ProjectionPending`:** no row reader consults it, so it would only put
  "refresh pending" on the notice line for the whole pass.
- [x] **Step 2:** `MenuEventInventory` seeds (Armed) or expires `Attached`
  (Running), then advances, returning the effects. `MenuEventOperationResult`
  with `event.Background` goes to `finishReattach` and never into
  `reduceOperationResult` -- a pass attempt never held the operator's slot.
- [x] **Step 3: apply `passViewOf` inside the row lookups** --
  `findMenuThread`, `selectedMenuThread`, `visibleRootThreads` -- and add
  `menuRowSelectable`, consulted by `moveRootSelection` and
  `reconcileRootSelection`. Then the guard: a test parsing `couchtty`'s
  non-test sources that fails on any read of `state.Inventory` outside those
  lookups.
- [x] **Step 4:** add `MenuEventReattachArm`, `MenuEvent.Background`,
  `MenuEvent.Diagnostic`.

### Task 8: console wiring -- DONE

**Files:** create `couchtty/console_reattach.go`; modify `console.go`, `console_menu.go`; test `console_reattach_test.go`

- [x] **Step 1: red.** Fixture tests:
  1. the first inventory produces exactly one queued resume with
     `warm-only=true`; completing it attaches a pane and queues the next;
  2. **a background attach never takes focus** -- drive `onExit` of the last
     pane with the switcher focused (so `c.active == ""`), then complete a
     background attempt: `c.focus` is still the panel and `c.tracker` was not
     seeded;
  3. an operator switch waits behind at most the running attempt;
  4. **placeholders:** after seeding, the status model carries one placeholder
     per pending thread, the loading one flagged; a click on a placeholder's
     columns resolves to no actor; when the loading thread attaches, its chip
     occupies the column its placeholder held;
  5. **the status-row tick** runs only while a thread is `Loading`, repaints,
     and stops when the pass finishes and when the console stops;
  6. a failure drops the placeholder and marks the row, and the pass continues;
  7. `Stop` mid-attempt cancels it (via `SetOperationDispatcher`) and runs no
     further attempt;
  8. an unarmed console emits no background effect and draws no placeholder.
- [x] **Step 2:** `ArmReattachPass`; `finishMenuRefresh` and `finishOperation`
  dispatch the effects `ReduceMenu` returns (both discard them today, and
  neither event kind produced any before, so nothing else starts dispatching).
- [x] **Step 3:** `runMenuOperation`'s background branch goes **first**,
  before the attention-capture block and the no-dispatcher path, both of which
  address the operator's in-flight slot. Key: `reattach\x00<attempt>`.
- [x] **Step 4:** declare the `background` implicit arg on `attach` in
  `ops.go` (bump the expected attach arity from 2 to 3 in `couchcmd`'s
  `TestOperationArityMatchesExpectation`). `ExecuteConsoleOperation` reads it;
  `installObservedThreadActor` takes `background` and skips focus and tracker
  seeding. `finishOperation`'s focus steal becomes
  `origin.Operation == "resume" && err == nil && startedHandleID != "" && !origin.Background`.
- [x] **Step 5:** `paintNow` appends `pendingPlaceholders` after the attached
  chips; the status-row spinner timer joins the Run loop's `select`.
- [x] **Step 6:** `TestAReattachedChildKeepsItsTrackingMode` and the whole
  `./cmd/internal/couchtty/` suite still pass.

### Task 9: rendering -- DONE

**Files:** modify `couchtty/menu_render.go`, `reserve.go`; tests `menu_render_test.go`, `reserve_test.go`

- [x] **Step 1: red.**
  - Switcher: queued renders `queued` greyed; loading renders
    `reattaching...` greyed with the current spinner frame; failed renders
    `reattach failed: <code>`; attached-but-not-yet-in-inventory renders
    `live`; a Detached row outside the pass renders `detached · <age>`.
  - Status bar: a placeholder renders its label greyed, the loading one with
    the spinner frame; **no placeholder contributes a `ChipSpan`**; attached
    chips keep exactly the columns they had without placeholders present.
- [x] **Step 2:** implement, reading the pass **before** the inventory state
  for rows the pass owns. Keep the vocabulary guard green.

### Task 10: `couchcmd` arms the pass -- DONE

**Files:** modify `couchcmd/run.go`; test `run_test.go`

- [x] `runConsole` arms with `start.Record.Thread` after
  `dispatchInitialAttach` succeeds, never when it fails, and never for
  `couch resume <tag>` (decision 9).
  Done in `beginConsole`, which is split out of `runConsole` so the ordering
  can be tested with a fake dispatcher. With a real child the ordering is
  untestable: an exited child fails the attach first. 4 of 4 mutations killed.

### Task 11: the trace -- DONE

**Files:** modify `couchtty/inputtrace.go`, `couchcmd/run.go`

- [x] Extend the existing trace plumbing with `startup`, `first-frame`,
  `pass-seeded`, `reattach-start` and `reattach-done`; `couchcmd` reads
  `COUCH_TRACE`. It also gained `inventory`, and the shared plumbing is now in
  `couchtty/trace.go`; see the Revisions entry for Task 11. Mutations: 23 of 23
  killed, 3 of them only after tests were added for them.

### Task 12: measure, document, smoke, close

- [ ] Add `PAIR_PROBE_SAMPLE_SECS` to `cmd/probes/reattachcost`, bounded like
  the rest of the probe.
- [ ] **Measurement (operator-assisted: couch needs a real terminal).** With
  N detached threads and co-tenancy recorded: run the sampler for 30 s while
  the operator starts `COUCH_TRACE=... couch`. Record first-frame time, pass
  duration, per-attempt times, the refresh count during the pass, and
  `zellij action` p50/p95/max against the quiet baseline -- with the
  status-row spinner running, since it is a new periodic repaint.
- [ ] `atlas/couch.md`: the pass under the switcher's operation model (its
  decisions, not its cells), the placeholders, and the `COUCH_TRACE` format.
- [ ] **Mutation sweep:** one row per cell with a named test, plus: the focus
  steal ignoring `Background`; `installObservedThreadActor` seeding focus on a
  background attach; the pass emitting without `warm-only`; cell 10's hold
  removed; the Root exclusion removed; a placeholder recording a `ChipSpan`;
  the cursor landing on a pending row; the status-row tick running with
  nothing `Loading`.
- [ ] Unsandboxed `make test`.
- [ ] **Operator smoke.** Restart couch with several detached threads: every
  pending thread appears at once, greyed, in the status bar and the switcher;
  the starting one spins; none can be clicked or selected; they fill in within
  seconds without moving the cwd thread's focus or screen; quitting mid-pass
  leaves the rest detached.
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

### 2026-09-11 — M1 review round 2: three rules, applied as rules

**Reason.** The review converged (SHIP) but raised three advisory families, each
its second instance. Each is fixed as a rule, not at the named sites.

- **Do not restate, point.** "Grep every restatement in the same edit" was
  stated in round 1, and the next two commits still produced six stale copies:
  "three readers" in code, the test header and the atlas while the test
  asserted four, plus a dangling "and" and a promised measurement that had been
  replaced. So the reader list now has ONE home, `startupAsks`' doc comment,
  and every other site names it instead of repeating a number.
  - Swept: `startup.go`, `startup_proof_test.go`, `atlas/couch.md`, and this
    plan's Core concepts entry and envelope.
- **A reach claim is pinned by the test the comment names.** `DetachedSessions`'
  doc said an unreadable scope "contributes no bindings", and after the
  union-of-reads refactor the code no longer did that: it iterated scopes
  rather than reads, so a failed scope's threads took their names from legacy
  rows another read replayed. The comment stated the right rule -- the
  unreadable file may hold a newer row, so the legacy name could be one the
  thread has left -- so the code now follows it.
  `TestDetachedSessionsBindsNothingForAnUnreadableScope` pins it, the comment
  names that test, and removing the guard fails it.
- **One derivation of a thread's current session name.** `lookupSessionName`
  and `effectiveBindings` computed the same fact, for `PairSession` and
  `DetachedSessions` respectively. `lookupSessionName` is deleted and
  `PairSession` calls `effectiveBindings` over its one read, so the two cannot
  judge a thread by different names. The test helper `claimsOf`, a third copy
  of the counting rule, now counts through `claimsFromBindings`.


### 2026-09-11 — the operator's UX for pending threads: placeholders, none selectable

**Reason.** The operator specified how pending threads look and behave: a
placeholder in the status bar with a spinner while a thread starts, not
clickable; the same row greyed and not selectable in the switcher. Asked about
queued threads, they chose **all pending shown at once, none selectable.**

**Delta, applied to the body rather than only recorded here** (the gate has
flagged stale bodies three times):
- **Dropped: adoption and queue-jumping** -- old decisions 3-4 and cells 9-10.
  This removes the most intricate part of M2: the verb that claimed the
  in-flight attempt, the `adopted` focus rule, and the review findings that
  kept finding readers that bypassed it. A background completion now never
  takes focus, full stop.
- **Added: placeholders.** `StatusActor.Placeholder`, drawn greyed after the
  attached chips in pass order, with the spinner on the loading one, and no
  chip span -- unclickable by construction. A resolved placeholder becomes its
  chip in the same column.
- **Added: non-selectable rows.** `menuRowSelectable`, the one predicate the
  cursor and auto-select consult.
- **Added: a status-row spinner tick** on the Run loop, armed only while a
  thread is `Loading`. The existing spinner cannot serve: it runs only while
  the switcher is focused. This is also the seam #231's clock would extend.
- The transitions table is renumbered 1-13; the Decisions, the pass view and
  Tasks 6-12 are rewritten whole.
- **Swept:** Scope (#214 bullet), the Done-when deviation (now decided), the
  Decisions, `ReattachPass`'s attempt-identity note, the pure-entity and
  integration tables, the `finishOperation` focus note, the pass view, the
  transitions, and Tasks 5-12.

**Estimate unchanged at 3.13.** Removing adoption shrinks Task 7; placeholders
and the tick grow Tasks 8-9. The two are about even, and re-costing a settled
estimate to match a design change would be back-fitting.

### 2026-09-11 — estimate calibration evidence at M2's start

**Reason.** The estimate-quality judge (INFO, no refusal) asked that M1's
measured actual be recorded against its items, and the M2-forward remainder
stated. That is calibration evidence, not re-costing, so `estimate_hours`
stays 3.13.

**The numbers, and why they do not compare directly.**
- **M1's items** summed to about 1.04 h: design 0.24 x 1.15 plus impl 0.76.
- **M1 closed at a measured 5.06 h.** That is #206's whole share since the
  2026-09-10 claim, so it carries all of the issue's up-front design: the
  measure-first probe, the durable plan, and five plan-gate rounds that
  covered BOTH milestones. The estimate's items were implementation. Most of
  the gap is design the items did not cost, not M1 running five times over.
- **M2's items** are the remaining 2.09 h. The judge expects two of them to
  run over:
  - Task 8, the console wiring, slotted at 0.43 h: expect 0.8-1.2 h. It is the
    wiring shape that produced #230's four review rounds.
  - Task 12, the close-out, slotted at 0.11 h: expect 0.4-0.6 h.

  So M2 will likely land about 1 h over its items, and that should be visible
  in the ledger at close rather than a surprise.

### 2026-09-11 — Tasks 8-9 as built: two decisions and one extraction

**Reason.** Implementation settled three things the plan left open or got
wrong.

- **A successful leave ends the pass.** The leave's result clears the
  operator's slot, and the pass would then advance: it enqueues one more
  reattach only for `Stop` to cancel it, which is the outcome cell 10 exists to
  prevent. The rule is in the pure reducer (`ReattachDone` on a successful
  leave), pinned by `TestASuccessfulLeaveEndsThePass`. The transitions table
  gains it implicitly under cell 13's "Stop".
- **The switcher's "reattaching..." row carries no spinner.** The switcher's
  spinner advances only while a progress notice shows, and the pass shows
  none, so a glyph would sit frozen. The ellipsis says "in progress"; the live
  animation is on the status bar, where the operator is while the pass runs.
  Both surfaces share one spinner table (`spinnerGlyph`) and one grey
  (`placeholderSGR`).
- **`statusModelLocked` is split out of `paintNow`.** The model is where a
  placeholder either appears or silently does not. Without the split, nothing
  checked that `paintNow` adds them from the pass: the placeholder test handed
  `RenderStatusRow` a model built by hand.

**One equivalent mutant.** Moving the pass's queue key from `reattach\x00<n>`
into the operator's namespace survives. The pass and the operator draw attempt
numbers from one counter, so keys cannot collide whatever the prefix. The
comment on `runBackgroundOperation` that claimed the prefix "can never
collide" was corrected to say where uniqueness actually comes from.

### 2026-09-11 — Task 11: the trace gains an `inventory` event, and a `trace.go`

**Reason.** Task 12's measurement records "the refresh count during the
pass", and none of the five planned events measures it.

**Delta.**
- `COUCH_TRACE` also writes `inventory` for every inventory that lands, as
  `rows=N` or `error`. That is a count, never content, as Trust requires.
- The shared file plumbing is `traceFile`, in a new `couchtty/trace.go`
  alongside the event tracer. `inputtrace.go` keeps the keystroke probe, now
  built on `traceFile` (`ARCH-DRY`).
- Each event is recorded at its one source in the console, rather than derived
  from every `ReduceMenu` call site:
  - `pass-seeded` in `finishMenuRefresh`, the only place an inventory is
    reduced;
  - `reattach-start` in `runBackgroundOperation`, which every background
    effect passes through;
  - `reattach-done` in `finishOperation`, for a Background origin;
  - `first-frame` at the first `paintNow`. Nothing paints before `Run`: no
    attach path calls `paintNow`, and `switchTo`'s callers are all on the
    Run loop;
  - `startup`, stamped with the process start that couchcmd reads before the
    console exists.
