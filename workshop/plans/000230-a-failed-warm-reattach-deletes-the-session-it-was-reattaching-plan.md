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
   `(owns, helper, presence, phase)` into one of
   `quiesce / rollback / retire / mark-unknown`. It is exhaustively table
   tested, the way `ReconcileStart` is.
2. **Ownership comes from couch's own record of the start**, never from a
   struct a caller relays back.
3. **The fake models the deletion**, so a test that asserts "the thread is
   still resumable" can actually fail.

**Tech Stack:** Go, `cmd/internal/couchcore`.

---

## The class, enumerated

`quiescePostAckStart` (`couch.go`) is the only caller of `Artifacts.Quiesce`,
which deletes the session (`artifactcollision.go` →
`launcher.QuiesceThreadSession` → `zellij delete-session --force` plus a kill
of that session's server). `reconcileInterruptedStarts` rolls back or promotes
records and never quiesces. The routes in:

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
| `StartCleanupAction` | `cmd/internal/couchcore/startcleanup.go` | new |
| `StartCleanupInput` | `cmd/internal/couchcore/startcleanup.go` | new |
| `DecideStartCleanup` | `cmd/internal/couchcore/startcleanup.go` | new |
| `ActorRecord.Warm` | `cmd/internal/couchcore/registry.go` | modified |
| `StartResult.Warm` | `cmd/internal/couchcore/ops.go` | modified |
| `ResumeStart` | `cmd/internal/couchcore/resume.go` | new |

- **`DecideStartCleanup(StartCleanupInput) StartCleanupAction`** is the whole
  rule, in one pure function:

  ```go
  type StartCleanupAction string

  const (
      CleanupQuiesce     StartCleanupAction = "quiesce"      // delete the session this start created
      CleanupRollback    StartCleanupAction = "rollback"     // remove the start claim (creating phase)
      CleanupRetire      StartCleanupAction = "retire"       // retire the live incarnation, thread goes detached
      CleanupMarkUnknown StartCleanupAction = "mark-unknown" // cannot prove; leave it recoverable
  )

  type SessionPresence uint8

  const (
      PresenceUnobserved SessionPresence = iota // the question could not be asked
      PresenceAbsent
      PresencePresent
  )

  type StartCleanupInput struct {
      OwnsSession bool            // this start created the session
      HelperDead  bool            // the exact helper process is proven gone
      Presence    SessionPresence // only consulted when !OwnsSession
      LiveRecord  bool            // routes 5-6 (a live incarnation) vs 1-4 (a claim)
  }
  ```

  The rule, in order:
  - `OwnsSession` → `CleanupQuiesce` (today's behaviour, unchanged).
  - `!HelperDead` → `CleanupMarkUnknown`. Nothing durable is undone while a
    process that may still be writing is unaccounted for.
  - `!LiveRecord` → `CleanupRollback`. Presence is not consulted: rollback
    removes only this start's own claim. If the session survived, the thread
    reads Detached again; if it died independently, the thread honestly reads
    `session-gone`.
  - `LiveRecord && Presence == PresencePresent` → `CleanupRetire`.
  - otherwise → `CleanupMarkUnknown`. Unknown is recoverable; a deleted
    session is not.
  - **Relationships:** 1 call per post-ack failure. No IO.
  - **DRY rationale:** six routes asked the same question in two different
    functions. Precedent: `ReconcileStart` (`starttransaction.go`).
  - **Future extensions:** a park-shaped cleanup would add a phase, not a
    branch at each call site.
- **`ActorRecord.Warm`** is couch's own record that this start attached to a
  pre-existing session. `launchTrackedThread` sets it from `in.Warm`, which it
  derives from its own detached proof. **`AbortStarted` reads the registry's
  record, not the `StartResult` the caller hands back**, so a caller that
  relays a zero value cannot select the destructive branch. `AbortStarted`
  already refuses a record that does not match the registry.
  - **`StartResult.Warm`** exists only so the console can tell a warm landing
    from a cold one. Nothing destructive reads it.
- **`ResumeStart(ctx, address) (StartResult, error)`** is `ResumeContext`
  returning the whole start. `ResumeContext` wraps it, so `relaunch.go` and the
  existing tests keep their call shape.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `quiescePostAckStart(address, h, owns)` | `couch.go` | modified | helper signalling, `Artifacts.Quiesce` |
| `applyStartCleanup` | `couch.go` | new | thread store, `PairSession` |
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
  performs the one action. `CleanupRetire` uses
  `RetireIncarnation(address, revision, helperIdentity, current.LastActiveAt)`
  — the operation `Detach` uses — passing the thread's existing
  `LastActiveAt`, because a failed reattach is not activity and
  `MonotonicLastActiveAt` then leaves it unchanged. It retries a revision
  conflict the bounded way `Detach` does.
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

## Open question, recorded rather than answered

The non-owning branch still ends the helper with `handleCleanup`, which
escalates to an unconditional SIGKILL of the helper's process group. `Detach`
deliberately avoids exactly that, on the grounds that it can truncate an agent
mid-write (`detach.go`). Whether that risk is real for a `pair resume` client
whose agent lives in the zellij **server** is not established here, and this
issue does not change the signal policy it inherited. If the risk is real it
is a separate issue against every rollback path, not just the warm one. The
Log records this; `TestSessionDetachLive` pins only the gentle case.

## Tasks

### Task 1: the pure decider

**Files:**
- Create: `cmd/internal/couchcore/startcleanup.go`, `startcleanup_test.go`

- [ ] **Step 1: red.** `TestDecideStartCleanupTable` enumerates **all 24**
  combinations of the input (2 owns × 2 helper × 3 presence × 2 phase) with
  an expected action for each, asserting that owning always quiesces, that a
  live helper never destroys anything, and that no non-owning row quiesces.
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
  6. a successful warm `ResumeStart`, then `AbortStarted(start, cause)`.

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
- Modify: `registry.go` (`ActorRecord.Warm`), `ops.go` (`StartResult.Warm`)
- Modify: `resume.go` (`ResumeStart`; `ResumeContext` wraps it)
- Modify: `launch_existing.go` (routes 1–5 pass `!in.Warm`; `launchTrackedThread` records `Warm` on the `ActorRecord`)
- Modify: `couch.go` (`quiescePostAckStart`, `applyStartCleanup`, `failPostAckStart`, `AbortStarted`)
- Modify: `startup.go`, `operationdispatch.go` (call `ResumeStart`, so the console gets `Warm`)

- [ ] **Step 1:** add the `owns` parameter. Every existing caller passes
  `true` except the warm routes, so the compiler enumerates the sites.
- [ ] **Step 2:** `AbortStarted` resolves ownership from the registry record
  it already validates, not from the relayed `StartResult`.
- [ ] **Step 3:** `applyStartCleanup`, replacing the ad-hoc tails of
  `failTrackedPostAckStart` and `failPostAckStart`.
- [ ] **Step 4:** Task 3 green, then `go test ./cmd/internal/couchcore/ ./cmd/internal/couchcmd/ ./cmd/internal/couchtty/`.

### Task 5: verify and close

- [ ] **Mutation sweep** (apply-asserted, named failures only, restore from
  saved bytes), each killed by name:
  - `DecideStartCleanup` returning `CleanupQuiesce` for a non-owning input →
    Task 1 and every warm row;
  - `owns` forced true at each of the six routes → that row;
  - `AbortStarted` reading `start.Warm` instead of the registry → route 6
    with a zeroed relayed struct;
  - the decider ignoring `HelperDead` → the live-helper rows;
  - `CleanupRetire` substituted by `CleanupMarkUnknown` → the Detached-again
    assertion;
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
