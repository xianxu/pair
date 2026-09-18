# Dismiss continuation, and a failed request that composes with its thread (#280) Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A retained `Failed` continuation stops replacing its thread. The row
shows the thread's real state with a continuation marker. The operator can
**Dismiss** the request, which deletes it from the record, so relaunch, cold
resume and switch-agent admit the thread again. **Retry** from the switcher
works, where today it probably fails before reaching its thread.

**Architecture:** Dismissal is a NAMED store transition,
`ThreadStore.DismissFailedContinuation`. It CAS-clears `record.Continuation`
only when the request is `Failed` and its ID matches (#256 lesson: *"a mutation
callback is not a transition"*). A declared operation `dismiss-continuation`
(direct-store, metadata effect, row action, internal presentation) reaches it
from the switcher and from `couch --internal dismiss-continuation <ref>`. The
switcher composes the state text and offers dismiss beside retry.
`continuationGuard` stays fail-closed, and its refusal names both exits.

**Operator decision (2026-09-17):** dismissal deletes the request, with no
terminal phase and no audit trail. Records decode with `DisallowUnknownFields`
(`strictjson/decode.go:23`), so any new phase value or field would make
pre-change binaries (long-running `pair` helpers) reject the whole record.
Deletion is also the only skew-safe shape.

**Tech Stack:** Go: `cmd/internal/couchcore`, `couchtty`, `couchcmd`.

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `checkpoint.CheckDismissible` | `cmd/internal/checkpoint/request.go` | new |
| `checkpoint.Exits` | `cmd/internal/checkpoint/request.go` | new |
| `checkpoint.AllPhases` | `cmd/internal/checkpoint/request.go` | new |
| `withContinuationExits` | `cmd/internal/couchcore/continuation.go` | new |
| `continuationGuard` | `cmd/internal/couchcore/continuation.go` | modified |
| `dismiss-continuation` declaration | `cmd/internal/couchcore/ops.go` | new |
| `continuationArguments` | `cmd/internal/couchcore/ops.go` | modified |
| `rootStateText` | `cmd/internal/couchtty/menu_render.go` | modified |
| `menuActionItems` | `cmd/internal/couchtty/menu.go` | modified |
| `ContinuationRefuses` | `cmd/internal/couchcore/continuation.go` | new |

- **`DismissFailedContinuation(address, expectedRevision, requestID)`**: the
  one transition that retires a failed request. It follows its siblings: the
  store method takes the expected revision (`updateExistingThread`,
  `threadstore.go:293,313`), and the caller reads the record through
  `requestRecord` (`continuation.go:140-152`), retrying on a stale revision the
  way `advanceContinuation` does. It sets `Continuation = nil` and refuses,
  writing nothing, when:
  - there is no request, or the id is obsolete: `requestRecord`'s existing
    messages, reused rather than reworded;
  - the phase is not `Failed` (*"continuation … is running; only a failed
    continuation can be dismissed"*).

  An empty `requestID` (the CLI) means "the retained one".
- **`Couch.DismissContinuation(ctx, address, requestID)`**: the couchcore
  entry point. It resolves nothing itself; dispatch already did. It returns the
  updated thread.
- **`continuationGuard`**: still refuses every unfinished request. For `Failed`,
  the message names both exits (retry re-delivers, dismiss drops) in Couch and
  as `couch --internal …`. `Pending`/`Running` messages are unchanged: they are
  in flight, and dismiss is not offered for them.
- **`dismiss-continuation`**: `ExecuteDirectStore`, `EffectMetadata` (a record
  write, no process), `ConfirmNone` (it stops nothing and the checkpoint file
  survives, the same as retry, which also has no confirmation; this also avoids
  the confirm frame's live-only admission at `menu.go:787-791,1566-1573`),
  `ResultThread`, `PresentationInternal`, `RowAction: true`.
  - Arguments follow `name`'s shape: `ref` OPTIONAL (CLI positional), `tag`
    implicit, `repo-scope` required implicit, `request-id` implicit optional.
    The switcher passes `tag` and `request-id`, never `ref`.
- **`continuationArguments(bootstrap)`**: the bootstrap `ref` becomes OPTIONAL
  (the retry fix). Today it is `Required`, so the switcher supplies `ref` on top
  of `threadEffect`'s `tag`, and `resolveOperationThread` refuses both
  (`operationdispatch.go:421-423`). The CLI still binds it positionally;
  omitting it resolves to *"thread reference is required"*.
- **`rootStateText`**: EVERY non-complete phase composes as
  `<state text> · <label>`, whatever the state (see the close-review revision).
  The text originally planned here said `Pending`/`Running` may displace the
  state because they are bounded in time. That is false: the 30s deadline and
  the owner's scan only run while a live owner watches the address, so a request
  whose owner died reads `continuing…` indefinitely.
- **`menuActionItems`**: a `Failed` request COMPOSES with the row's actions,
  like the state text. The row keeps what its state offers, minus exactly the
  operations `continuationGuard` refuses, plus `retry-continuation` and
  `dismiss-continuation`.
  - A live row becomes `detach`, `retry-continuation`, `dismiss-continuation`,
    `park`, `name`, `describe`. Relaunch and switch-agent are refused by the
    guard (`relaunch.go:90`, `switchagent.go:128`).
  - Park and detach STAY. Neither reads the request (`park.go` never does, and
    `detach.go` only for archive). A later warm reattach is checked by
    `validateContinuationWarm`, and if it refuses, the detached row offers
    retry and dismiss, so detach is not a trap.
  - Rows with `Recovery` (non-live) already compose; dismiss joins them after
    retry.
  - In-flight phases keep today's sets, on purpose, because the continuation
    owns the thread mid-replacement. `Running` offers `retry-continuation`,
    `name` and `describe`: retry reconciles a stalled request. `Pending` offers
    only `name` and `describe`, as before #280 (pinned by
    `TestFailedContinuationComposesWithALiveRowsActions`). This is a statement
    of which actions are offered, not a time bound.
- **`ContinuationRefuses(operation string) bool`**
  (`couchcore/continuation.go`, beside `continuationGuard`): the ONE list of
  operations the guard refuses: relaunch, switch-agent, (cold) resume, start.
  The menu filters through it rather than restating it. A couchcore table test
  drives each listed operation on a thread with a `Failed` request and asserts
  the continuation refusal by message. It drives park and detach too, asserting
  they are NOT refused by it. So the list and the guard sites cannot drift.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `ThreadStore.DismissFailedContinuation` | `cmd/internal/couchcore/continuation_store.go` | new | the file-backed thread store (applies `CheckDismissible` in its CAS) |
| `Couch.DismissContinuation` | `cmd/internal/couchcore/continuation.go` | new | the store, via `writeRequestRecord`'s stale-revision loop |
| `DirectStoreExecutor` (dismiss case) | `cmd/internal/couchcore/operationdispatch.go` | modified | the file-backed thread store |
| switcher dispatch (`dispatchMenuOperation`, refresh) | `cmd/internal/couchtty/menu.go` | modified | the declared-operation dispatcher |
| CLI owner/scope policy | `cmd/internal/couchcmd/run.go:365-378` | modified | `couch --internal` |

- **Store (ARCH-MOCK):** the thread store is already a portable file-backed
  store. Tests run it in a temp directory and never touch the live store. The
  production executors (`DispatchOperation` + `DirectStoreExecutor` /
  `CouchLiveOwnerExecutor`) run in tests exactly as wired. That is the seam the
  retry bug hid behind, because its only test used a fake live-owner executor.

### Architecture principles applied

- **ARCH-ORDER.** Transitions:
  - (`Failed`, dismiss(id)) → no request.
  - Refused, writing nothing, from: no request, `Pending`, `Running`,
    `Complete`, and an obsolete id.

  Retry (`Failed`→`Running`) and dismiss race through the store's revision
  CAS. The first to commit wins, and the loser refuses on the new phase (dismiss
  sees `Running`) or on absence (retry sees no request). Both orders are tested
  with a controlled interleaving: commit one, then drive the other. The failure
  most likely to be mishandled is an obsolete id from a stale switcher row, and
  it is tested.
- **ARCH-FUNERAL.** Failed requests had no end until now; dismissal IS their
  end. Couch's own checkpoint copy (`<store>/continuation/<scope>/<tag>.md`,
  written at `continuation_store.go:92`) is deliberately LEFT. It is bounded at
  one per thread, overwritten by the next publish, and removed at archive
  (`threadstore.go:1211-1214`). The repo's `workshop/continuation/*.md` file
  belongs to that repo and is never touched.
- **ARCH-SECURE.** `request-id` arrives from a possibly stale switcher row and
  must match exactly. The record keeps being read strictly, and no new
  persisted shape is introduced.
- **ARCH-DRY.** Reuses `resolveOperationThread`, the continuation projection
  (`continuationStatus`), and retry's switcher plumbing as the template. The
  ~12 inline `Phase != Complete` sites stay: with no new phase, they still mean
  exactly what they say.
- **ARCH-PURE.** The transition's rules are pure checks inside the store CAS;
  the switcher functions are pure over `ActionableThreadSummary`.
- **ARCH-CONSTRAINTS.** An operator action is one CAS write, with no process
  effects. N/A beyond that.
- **ARCH-PURPOSE.** It delivers #280's four Done-when bullets, plus the
  relaunch unblock and the retry repair, which share the purpose ("a failed row's
  exits exist and work").

---

## Tasks

- [x] **Reproduce the retry bug (red), across the real seam.** Switch
  `TestContinuationFailedRowKeepsAnExplicitRetry`
  (`couchtty/console_continuation_test.go:17`) from its fake LiveOwner to the
  PRODUCTION dispatcher (`DispatchOperation` + `CouchLiveOwnerExecutor`) on a
  temp-dir store, as `operation_queue_test.go` already does. Expect the *"thread
  ref and exact tag cannot both be supplied"* refusal. If it does not
  reproduce, log why and drop the retry fix.
- [x] **Fix retry (green).**
  - Make `continuationArguments(true)`'s `ref` optional, and stop
    `dispatchMenuOperation` from adding `ref`. The same test then reaches
    `RetryContinuation`, asserted by its result or its own refusal, never by a
    bare `err == nil`. It stays the regression test: putting `ref` back turns
    it red.
  - Accepted knowingly: with `ref` optional, `couch --internal
    retry-continuation` given no argument now fails at dispatch (*"thread
    reference is required"*) after the supervisor lease, not at binding.
    It is an internal operation, the lease is released on exit, and the message
    names what is missing.
- [x] **Store transition (TDD).** `DismissFailedContinuation`: success clears
  and bumps the revision. Each of the five refusals is asserted by message AND by
  an unchanged revision (writes nothing). Both retry/dismiss orders are covered.
- [x] **Operation + couchcore entry.** Declare `dismiss-continuation`, add
  `Couch.DismissContinuation`, and add the `DirectStoreExecutor` case. Add rows
  to `TestOperationDeclarationsAreClosureFreeCompleteAndOwned`,
  `TestOperationArityMatchesExpectation`, `TestContinuationOperationsDeclared`,
  and the couchcmd owner/scope policy tests (`run.go:365-378`: current repo
  scope yes, live owner no, console no).
- [x] **Guard, its list, and the reported symptom.**
  - Add `ContinuationRefuses` and its table test (listed operations refused by
    message; park and detach not).
  - With a live thread and a `Failed` request, `Relaunch` refuses naming BOTH
    retry and dismiss. After `DismissContinuation`, `Relaunch` gets PAST
    `continuationGuard`, asserted by reaching the next refusal or the park (fake
    lifecycle).
  - Publishing after a dismissal: a stale publish from the old source
    generation is refused by `sameContinuationSource` (`continuation.go:105`),
    now the only guard once the old request is gone.
  - Add a comment in `checkpoint/request.go`'s `Advance` that couchcore's
    dismissal (deleting the request) is the other exit from `Failed`.
- [x] **Switcher.**
  - `rootStateText` table over `AllThreadStates` × {nil, Pending, Running,
    Failed, Complete}: every non-complete phase composes (revised at close
    round 1); `nil`/`Complete` show the plain state.
  - `menuActionItems` over state × phase: `Failed` composes (state actions
    minus `ContinuationRefuses`, plus retry and dismiss); `Running` offers
    retry, name and describe, and `Pending` offers name and describe.
  - `dispatchMenuOperation` passes `request-id` for dismiss, and never `ref`.
  - Dismiss joins the refresh list.
  - Extend `TestRowActionDeclarationsAndTheMenuAgreeInBothDirections` and add a
    `TestContinuationFailedRowKeepsAnExplicitRetry`-style dismiss test that
    crosses the PRODUCTION dispatcher.
- [x] **Docs.**
  - `atlas/couch.md`: the operation list, "four internal operations" → five,
    with the literal `couch --internal dismiss-continuation`, as
    `TestOperationPresentationDocs` requires; and the offered-vs-permitted
    table.
  - `README.md`'s Retry continuation section gains Dismiss.
- [x] **Verify.** `go test ./... -count=1` with the retention scrub and the
  sandbox off. `make -k test` (the known `test-changelog` failure; remember that
  its Go recipe is skipped). Build `bin/couch` and `bin/pair`.
- [x] **Operator smoke.** On the `pair` thread:
  1. The row reads `live · continuation failed` and offers Dismiss.
  2. Dismiss it. The row reads `live`, and the normal actions are back.
  3. Alt+n relaunch succeeds and keeps the conversation.

  Record the result in `## Log`.

## Revisions

### 2026-09-17 — plan-quality round 1

- **PQ-1:** the restricted action set was justified by a guard that does NOT
  cover park or detach. Actions now compose like the state text, filtered by a
  single-sourced `ContinuationRefuses`, which a table test ties to the guard
  sites.
- **PQ-2:** the retry red/green test is the existing switcher test, moved onto
  the production executor, not a couchcore copy of the switcher's arguments.
- **PQ-3:** the store transition takes the expected revision and reuses
  `requestRecord`'s messages.
- **PQ-4:** Couch's checkpoint copy is named, bounded and left in place.
- **PQ-5:** a comment in `Advance`, plus a test that a stale publish after
  dismissal is refused.
- **PQ-6:** the CLI error ordering is accepted, with the reason stated.

### 2026-09-17 — reconciled with the Log at close

- Every task is evidenced in the issue's `## Log` and ticked.
- The operator smoke's step 3 (Alt+n relaunch after dismissal) is ticked
  without a live observation. The record shows the dismissal (revision 38, no
  `continuation`), but its only incarnation dates from couch's startup
  reattach. The relaunch-after-dismissal path is pinned by
  `TestFailedContinuationRelaunchesOnceDismissed`, which asserts the
  `Relaunched` outcome on the production relaunch.

### 2026-09-17 — close review round 1 (REWORK)

- **BR-2 (Critical):** the Pure table listed the store transition and
  `Couch.DismissContinuation`, both of which do store IO. The pure rule is now
  `checkpoint.CheckDismissible`, tested on literal requests. Both IO entries
  moved to Integration points, and the tables above are corrected in place.
- **BR-3:** "Pending/Running are bounded, so they may displace the state" is
  false without a watching owner. Every phase now composes. The in-flight action
  set stays restricted, with an honest reason: the continuation owns the thread
  mid-replacement, and retry reconciles a stalled one.
- **BR-4:** "every refusal names both exits" held only for the guard. The class
  is every refusal a retained request causes: recovery (4 sites), archive (2)
  and warm reattach (1) now go through `withContinuationExits`. The wording is
  one function, `checkpoint.Exits`, which `launcher` also uses. Each site is
  driven by `TestEveryRefusalARetainedRequestCausesNamesBothExits`.
- **Minors:**
  - `menuLiveActions` was moved above `menuActionItems`' doc comment.
  - The duplicate exits helpers were merged, and the stale-revision loop is
    shared (`writeRequestRecord`).
  - `checkpoint.AllPhases` drives the state-text test.
  - `continuationArguments`' parameter is renamed `operatorFacing`.
  - The console prunes `menu.Orientation` with the watch.
  - The guard-agreement test also drives cold resume and cold start, and
    `ContinuationRefuses`' doc names exactly what is driven.

### 2026-09-17 — close review round 2 (REWORK)

- **BR-11 (Critical), in-memory state follows the record:** my round-1 prune
  deleted switch-agent's orientation prompt, because `menu.Orientation` has two
  producers and no provenance. Every writer now records its producer
  (`setOrientationLocked`), and every prune names one (`dropOrientationLocked`)
  or explicitly supersedes (`supersedeOrientationLocked`, a new switch-agent
  launch). That covers all five pre-existing and new sites.
  `TestSwitchAgentOrientationPromptSurvivesContinuationScans` pins both scan
  paths.
- **BR-4 / BR-12, refusals and test reach:** each retained-request check wraps
  ONCE at its boundary. `prepareAbsentContinuation`'s retained branch became
  `admitRetainedRecovery`, which also covers the missed `AdmitRecoveryGeneration`
  refusal. The routing test is one row per `withContinuationExits` call site
  (4), each driven, plus a source scan that the call sites ARE the rows. The
  guard-agreement oracle now matches the guard's own
  `continuation <id> is failed; ` prefix, not wording the wrapper shares.
- **BR-3:** the plan's `rootStateText` bullet no longer carries the "bounded in
  time" reason, and the plan and atlas state the actual per-phase action sets.
- **Minors:**
  - no `[]Phase{` literal outside `AllPhases` (two test lists replaced, and a
    repo scan added);
  - `pair continue --retry` uses the phase-neutral `Exits("", tag)`.
