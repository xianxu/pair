# A failed warm reattach keeps its session — Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** no failure path after a start's helper is acknowledged may delete a
zellij session that the start did not create.

**Architecture:** every such path funnels into `quiescePostAckStart`, which
ends the helper and then quiesces the thread's durable session. Quiescing is
right only when the start **created** that session: a spawn, or a cold resume.
A warm reattach attached to a session that predates it.

Three pieces:
1. **A pure decider**, `DecideStartCleanup`, turns
   `(shape, helper, presence, phase)` into two independent answers: whether to
   delete the session, and which durable action to take. All 36 combinations
   are table tested, the way `ReconcileStart` is.
2. **Ownership comes from couch's own record of the start**, never from a
   struct a caller relays back.
3. **The fake models the deletion**, so a test that asserts "the thread is
   still resumable" can actually fail.

**Tech Stack:** Go, `cmd/internal/couchcore`.

---

## The class, enumerated

`Artifacts.Quiesce` deletes the session (`artifactcollision.go` →
`launcher.QuiesceThreadSession` → `zellij delete-session --force` plus a kill
of that session's server). It has exactly two callers:
- **`ArchiveThread`** (`detach.go`), which is **not** in this class and does
  not change. Archiving means removing the thread, so ending its session is
  the point, and its guard already refuses an occupied record before any
  effect.
- **`quiescePostAckStart`** (`couch.go`), the post-acknowledgement failure
  path, which is this issue.

`reconcileInterruptedStarts` rolls back or promotes records and never
quiesces. The routes into `quiescePostAckStart`:

| # | Route | Where | Start phase |
|---|---|---|---|
| 1 | acknowledge failed | `launch_existing.go`, the `h.Acknowledge()` exit | creating + nonce |
| 2 | context cancelled after acknowledge | the `ctx.Err()` exit after `Acknowledge` | creating + nonce |
| 3 | Pair registration failed or timed out | the `awaitResumeRegistration` exit | creating + nonce |
| 4 | registration promotion failed | the `StartRegistered` `AdvanceStart` exit | creating + nonce |
| 5 | registry persistence failed | `failPostAckStart` after `c.Store.Save` | live incarnation |
| 6 | the console could not attach the started child | `couchcmd/run.go` → `AbortStarted` → `failPostAckStart` | live incarnation |

Routes 1–4 reach it through `failTrackedPostAckStart`; routes 5–6 through
`failPostAckStart`.

**Route 6 is the one a warm reattach actually takes.** A warm reattach's
registration check is `awaitResumeRegistration`, which polls
`PairSession(address).Present` — that is satisfied by the session that already
exists, before the `pair resume` client has attached anything. So a client that
dies right after spawning passes registration, and is first noticed when the
console's attach finds the terminal gone. That is `AbortStarted`, which
quiesces. Routes 1–4 stay reachable (route 3 when the session itself dies
mid-reattach, so presence never returns), and the fix is one rule for all six.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `StartShape`, `DurableAction`, `SessionPresence` | `cmd/internal/couchcore/startcleanup.go` | new |
| `StartCleanupInput`, `StartCleanup` | `cmd/internal/couchcore/startcleanup.go` | new |
| `DecideStartCleanup` | `cmd/internal/couchcore/startcleanup.go` | new |
| `ActorRecord.Warm` | `cmd/internal/couchcore/registry.go` | modified |

- **`DecideStartCleanup(StartCleanupInput) StartCleanup`** is the whole rule,
  in one pure function:

  **Two dimensions, and a three-valued shape.** Whether to delete the session
  and what to do with the durable record are independent. And the input is not
  a warm boolean: the two OWNING shapes already had different tails, and
  merging them would have broken spawn. `launch_existing.go`'s
  `failTrackedPostAckStart` opens with `if !resume { return
  c.failPostAckStart(...) }`, so a spawn's claim-phase failure reconciles
  against registration evidence, where a cold resume's rolls back or marks
  unknown on the session's absence.

  ```go
  type StartShape uint8

  const (
      StartSpawn        StartShape = iota // created the thread and its session
      StartColdResume                     // relaunched a parked thread, new session
      StartWarmReattach                   // attached to a session that predates it
  )

  func (s StartShape) OwnsSession() bool { return s != StartWarmReattach }

  type SessionPresence uint8

  const (
      PresenceUnobserved SessionPresence = iota // could not be asked
      PresenceAbsent
      PresencePresent
  )

  type DurableAction string

  const (
      DurableRollback    DurableAction = "rollback"     // remove this start's claim
      DurableRetire      DurableAction = "retire"       // retire the live incarnation
      DurableMarkUnknown DurableAction = "mark-unknown" // cannot prove; stay recoverable
      DurableReconcile   DurableAction = "reconcile"    // today's spawn / live-record tail
  )

  type StartCleanupInput struct {
      Shape      StartShape
      HelperDead bool
      Presence   SessionPresence
      LiveRecord bool // routes 5-6 (a live incarnation) vs 1-4 (a claim)
  }

  type StartCleanup struct {
      Quiesce bool
      Durable DurableAction
  }
  ```

  `Quiesce` is `Shape.OwnsSession()`. `Durable` is:
  - **spawn, either phase** → `DurableReconcile` (today's tail, unchanged);
  - **any owning shape, live record** → `DurableReconcile` (today's
    `failPostAckStart` tail, unchanged);
  - **cold resume, claim** → `DurableRollback` when `HelperDead && Presence ==
    PresenceAbsent`, else `DurableMarkUnknown` (today's tail, unchanged);
  - **warm, claim** → `DurableRollback` when `HelperDead`, else
    `DurableMarkUnknown`. Presence is not consulted: rollback removes only
    this start's own claim. Session survived → detached again; session died
    independently → `session-gone`, honest either way;
  - **warm, live record** → `DurableRetire` when `HelperDead && Presence ==
    PresencePresent`, else `DurableMarkUnknown`.

  Nothing durable is undone while the helper is unaccounted for, except a
  spawn's reconcile, which reads registration evidence rather than the helper
  and is unchanged. Unknown is recoverable; a deleted session is not.
  - **Relationships:** 1 call per post-ack failure. No IO.
  - **DRY rationale:** six routes asked the same question in two different
    functions. Precedent: `ReconcileStart` (`starttransaction.go`).
  - **The owning rows are today's behaviour, restated in one place.** The
    existing spawn and cold-resume tests are what prove they did not change —
    `couch_test.go`'s post-acknowledgement table in particular.
  - **Future extensions:** a park-shaped cleanup would add a phase, not a
    branch at each call site.
- **`ActorRecord.Warm`** is couch's own record that this start attached to a
  pre-existing session. `launchTrackedThread` sets it from `in.Warm`, which it
  derives from its own detached proof. **`AbortStarted` reads the registry's
  record, not the `StartResult` the caller hands back**, so a caller that
  relays a zero value cannot select the destructive branch. `AbortStarted`
  already refuses a record whose identity does not match the registry, so the
  lookup it needs is one it already performs.
  - **No `StartResult.Warm`, and no `ResumeStart`.** An earlier draft added
    both so the console could relay warmness back. Once `AbortStarted` reads
    the registry instead, nothing reads that field — and an unused field whose
    zero value is the destructive branch is exactly what the gate flagged. So
    `ResumeContext` keeps its signature, and `startup.go` and
    `operationdispatch.go` are untouched. **#206 M2 therefore stays on
    `ResumeContext` too.**

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `quiescePostAckStart(address, h, shape)` | `couch.go` | modified | helper signalling, `Artifacts.Quiesce` |
| `applyStartCleanup`, `markLiveRecordUnknown` | `couch.go` | new | thread store, `PairSession` |
| `retireDetachedIncarnation` | `detach.go` | new | thread store, `PairSession` |
| `failTrackedPostAckStart`, `failPostAckStart` | `launch_existing.go`, `couch.go` | modified | the above |
| `AbortStarted` | `couch.go` | modified | reads the registry's `Warm` |
| `FakeThreadArtifactCollisionChecker.Quiesce` | `artifactcollision_fake.go` | modified | the fake's session state |

- **`quiescePostAckStart`** gains `owns bool`. With `false` it runs only the
  helper half of its retry loop, and returns once the helper is quiet. The
  helper half is **unchanged**: `handleCleanup`'s SIGTERM then unconditional
  SIGKILL of the helper's process group. This issue found a session deletion,
  not a signal policy, so the signal policy is out of scope — see Open
  question below.
- **`applyStartCleanup`** is the thin shell: it observes `HelperDead` and,
  when the decider needs it, `PairSession(address).Present` (an error or a
  missing observer is `PresenceUnobserved`), calls `DecideStartCleanup`, and
  performs the result.
  - **`DurableRetire` reuses `Detach`'s post-signal half rather than copying
    it.** `Detach` already owns "prove the session present, then retire this
    exact incarnation, retrying a bounded number of revision conflicts"
    (`detach.go`). That block is extracted as `retireDetachedIncarnation` and
    called from both. `detachedAt` is the thread's existing `LastActiveAt`,
    because a failed reattach is not activity and `MonotonicLastActiveAt`
    then leaves it unchanged.
  - **The extracted helper KEEPS `Detach`'s per-attempt `ctx.Err()` check**,
    which is that loop's only interrupt and is pinned by nothing — removing it
    would be an unobserved regression in `Detach`. The cancelled-context
    problem at route 5 is the caller's to solve: cleanup runs
    `context.WithoutCancel(ctx)`, because cleanup must complete precisely when
    the thing that failed was a cancellation. A cancelled context must not be
    able to turn a provable retire into a `DurableMarkUnknown`.
- **`FakeThreadArtifactCollisionChecker.Quiesce`** keeps its call log **and
  now models the effect**: it clears the address's detached session and marks
  its `PairSession` absent. Without this, the fake's quiesce is invisible to
  every later observation, so an assertion that the thread is still resumable
  passes whether or not the session was deleted (`ARCH-MOCK`: a fake laxer
  than production hides the bug).

## ARCH notes

- **`ARCH-PURPOSE`.** The filed finding named cancellation. The class is six
  routes, and the fix is a property every route reads, not a special case at
  one site. Lessons: "Define a category by its property, not by the gestures
  that produce it today".
- **`ARCH-PURE`.** The decision is a pure function with an exhaustive table;
  the IO shell only gathers the three observations and performs one action.
- **`ARCH-SECURE`.** Ownership is couch's own record. The one caller-relayed
  copy (`StartResult.Warm`) is read by nothing destructive.
- **`ARCH-ORDER`.** The Start transaction's states do not change. This fixes
  which cleanup effect a failure event takes in the creating and live phases.
- **`ARCH-MOCK`.** The fake models the deletion, so the guard can fail.
- **`ARCH-CONSTRAINTS`.** Failure paths only; the success path is untouched.

## Non-goals

Three adjacent behaviours this issue deliberately leaves alone, so a reviewer
does not read their absence as an oversight:

- **`ArchiveThread`'s quiesce** stays. Archiving is a request to remove the
  thread; deleting its session is the intent, not a side effect.
- **A cold resume's quiesce** stays. That start created the session, and
  leaving it behind after a failed start is the litter this cleanup exists to
  prevent.
- **The helper-ending signal policy** stays `handleCleanup`'s SIGTERM then
  SIGKILL — see the Open question below.

## Open question, recorded rather than answered

The non-owning branch still ends the helper with `handleCleanup`, which
escalates to an unconditional SIGKILL of the helper's process group. `Detach`
deliberately avoids exactly that, on the grounds that it can truncate an agent
mid-write (`detach.go`). Whether that risk is real for a `pair resume` client
whose agent lives in the zellij **server** is not established here, and this
issue does not change the signal policy it inherited. If the risk is real it
is a separate issue against every rollback path, not just the warm one.

The reason the SIGKILL is believed not to reach the agent: it signals the
**helper's** process group, and that group is the freshly spawned `pair
resume` client. The zellij server, and the agent inside it, predate that
process and sit outside its group — which is the same separation detach
relies on when it kills a client and keeps the session.
`TestSessionDetachLive` pins the SIGTERM case against the real binary; the
SIGKILL case is reasoned, not pinned, and the Log says so.

## Tasks

### Task 1: the pure decider

**Files:**
- Create: `cmd/internal/couchcore/startcleanup.go`, `startcleanup_test.go`

- [ ] **Step 1: red.** `TestDecideStartCleanupTable` enumerates **all 36**
  combinations (3 shapes × 2 helper × 3 presence × 2 phase). Its expected
  values are **written out as literals**, never recomputed from the same
  conditions the implementation branches on — a table that derives its own
  expectations mirrors the code and asserts nothing. Three property tests
  cross-check it over the whole input space: quiesce follows ownership;
  nothing destructive while the helper is unaccounted for (spawn's reconcile
  excepted, since it reads registration evidence); retire only on the warm
  path.
- [ ] **Step 2:** run it; it fails to compile.
- [ ] **Step 3:** implement `DecideStartCleanup`.
- [ ] **Step 4:** run it; green.

### Task 2: the fake models its own deletion

**Files:**
- Modify: `cmd/internal/couchcore/artifactcollision_fake.go`
- Test: `cmd/internal/couchcore/artifactcollision_fake_test.go`

- [ ] **Step 1: red.** `TestFakeQuiesceMakesAThreadUnresumable`: set a
  detached session and a present `PairSession`, call `Quiesce`, then assert
  `DetachedSessions` returns nothing and `PairSession` is absent.
- [ ] **Step 2:** implement, keeping the existing call log.
- [ ] **Step 3:** `go test ./cmd/internal/couchcore/` — a pre-existing test
  that relied on the fake's quiesce being invisible must be read, not
  reflexively updated: if one breaks, it was asserting a state production
  does not reach, and the fix goes in the test's comment.

### Task 3: red — every route, warm and owning

**Files:**
- Create: `cmd/internal/couchcore/warm_failure_test.go`

- [ ] **Step 1:** one table, six routes × {warm, owning}. Each row names its
  injection at the fake seam, and the table is the class enumeration:
  1. `BeforeAcknowledge` returns an error;
  2. `AfterAcknowledge` cancels the context;
  3. the session dies mid-reattach (`SetDetachedSession("")` and
     `SetPairSession(absent)` from `AfterAcknowledge`) with a short
     `resumeRegistrationTimeout`, so registration times out;
  4. a concurrent `UpdateExistingThread` from `AfterAcknowledge` bumps the
     revision, so the `StartRegistered` promotion conflicts;
  5. `env.Couch.Store = NewStore(<a file path>)`;
  6. a successful warm `ResumeContext`, then `AbortStarted(start, cause)`.

  The owning half of each row is the same injection against a
  verified-parked thread (cold resume), and for routes 5–6 also a `Spawn`.
- [ ] **Step 2:** assertions, per row:
  - **warm:** the address is absent from `env.Artifacts.Quiesces()`; the
    helper is not alive; routes 1–4 leave no incarnation and routes 5–6 leave
    none either (retired); and — for every row whose session survived —
    `ActionableThreadInventoryContext` classifies the thread `ThreadDetached`
    with `LastActiveAt` unchanged. Route 3's session died, so it asserts
    `ReasonSessionGone` instead.
  - **owning:** the address **is** in `Quiesces()`. This pins today's
    behaviour, which does not change.
- [ ] **Step 3:** run. Expected: every warm row fails on the `Quiesces()`
  assertion, and the owning rows pass.

### Task 4: carry ownership, apply the decision

**Files:**
- Modify: `registry.go` (`ActorRecord.Warm`)
- Modify: `launch_existing.go` (routes 1–5 pass `!in.Warm`; `launchTrackedThread` records `Warm` on the `ActorRecord`)
- Modify: `couch.go` (`quiescePostAckStart`, `applyStartCleanup`, `failPostAckStart`, `AbortStarted`)
- Modify: `detach.go` (extract `retireDetachedIncarnation`)

- [ ] **Step 1:** add the `owns` parameter. Every existing caller passes
  `true` except the warm routes, so the compiler enumerates the sites.
- [ ] **Step 2:** `AbortStarted` resolves ownership from the registry record
  it already validates, not from the relayed `StartResult`.
- [ ] **Step 3:** extract `retireDetachedIncarnation` from `Detach` (its own
  tests must stay green, unmodified), then `applyStartCleanup`, replacing the
  ad-hoc tails of `failTrackedPostAckStart` and `failPostAckStart`.
- [ ] **Step 4:** Task 3 green, then `go test ./cmd/internal/couchcore/ ./cmd/internal/couchcmd/ ./cmd/internal/couchtty/`.

### Task 5: verify and close

- [ ] **Mutation sweep** (apply-asserted, named failures only, restore from
  saved bytes), each killed by name:
  - `DecideStartCleanup` setting `Quiesce` for a warm input → Task 1 and every
    warm row;
  - `StartSpawn` folded into `StartColdResume` (the PQ-12 error) → the
    existing spawn tests;
  - the owning rows' `Durable` swapped for the warm one → the existing spawn
    and cold-resume tests;
  - `owns` forced true at each of the six routes → that row;
  - `AbortStarted` reading the relayed `StartResult` instead of the registry's
    `ActorRecord.Warm` → route 6 with a zeroed relayed struct;
  - the decider ignoring `HelperDead` → the live-helper rows;
  - `DurableRetire` substituted by `DurableMarkUnknown` → the Detached-again
    assertion;
  - cleanup's `context.WithoutCancel` removed → route 2, whose context is
    cancelled by construction;
  - the extracted helper's `ctx.Err()` check removed → `Detach`'s own
    interrupt test;
  - the fake's `Quiesce` reverted to log-only → Task 2.
- [ ] A comment on `quiescePostAckStart` states the rule and names the six
  routes by function, not line.
- [ ] `atlas/couch.md`, lifecycle section: a failed warm reattach leaves its
  session detached, and why.
- [ ] Record in the issue Log: the registration asymmetry that makes route 6
  the live one, and the SIGKILL open question.
- [ ] Unsandboxed `env -u PAIR_SESSION_ID -u PAIR_TAG make test`.
- [ ] `sdlc close --issue 230`, then `sdlc pr` and `sdlc merge --yes`.

## Estimate

Derived at `sdlc change-code`, after plan-quality.

## Revisions

### 2026-09-11 — what shipped, against what this plan described

**Reason.** The close review (BR-6, BR-7) found three places where this plan
and the tree disagree. Recorded here rather than left for the archive.

- **`StartCleanup` is gone; the decider returns a `DurableAction` only.** The
  plan had one function answering two questions. In the shell they are answered
  at different MOMENTS -- the session is ended first, and its absence
  afterwards is an input to the record's disposition -- so a single return
  value meant the `Quiesce` half was computed before it could be used and then
  never read. `StartShape.OwnsSession()` is now the named single authority for
  the session question, consumed once by `quiescePostAckStart`.
- **`applyStartCleanup` was built, and is where both entry points meet.** The
  plan named it; the first implementation instead left two ad-hoc tails, each
  consulting the decider differently. `failTrackedPostAckStart` and
  `failPostAckStart` are now one line each over the shared shell, and the
  live-record tail is the named `markLiveRecordUnknown`.
- **`retireDetachedIncarnation` is an INTEGRATION point, not a pure entity.**
  It observes `PairSession` and writes the thread store. The Core concepts
  table listed it under Pure entities, which is exactly the misclassification
  the table exists to catch.
- **`StartShape` is a string, and `ActorRecord.Shape` records it.** The plan
  carried a `Warm bool`. A string persists legibly and, more importantly, makes
  an unrecognised value answer "does not own" -- the direction that leaves a
  session behind rather than killing an agent.
