---
id: 000280
status: working
deps: []
github_issue:
created: 2026-09-17
updated: 2026-09-17
estimate_hours: 2.22
started: 2026-09-17T21:46:16-07:00
---

# A retained continuation failure masks a live thread's state in the switcher

## Problem

Operator report, 2026-09-17. The `pair` thread is **healthy and in active use**,
and the switcher renders it:

```
  brain        /Users/xianxu/workspace/brain        live
  tools        /Users/xianxu/workspace/tools        live
  parley.nvim  /Users/xianxu/workspace/parley.nvim  live
  ariadne      /Users/xianxu/workspace/ariadne      live
▸ pair         /Users/xianxu/workspace/pair         continuation failed — retry available
  astro        /Users/xianxu/workspace/astro        session gone
```

Couch itself classifies it live, at the same moment:

```
$ couch --show pair
pair                   /Users/xianxu/workspace/pair
  address: e108517d46ab4575/couch-c945633f5c806f21
  recorded: live     pid 80199
  live
```

So the state is right everywhere except the surface the operator actually reads.

### Cause — an unconditional precedence, one line up from the state

`couchtty/menu_render.go:414`, `rootStateText`:

```go
func rootStateText(thread couchcore.ActionableThreadSummary, now time.Time) string {
	if request := thread.Continuation; request != nil {
		switch request.Phase {
		case checkpoint.Pending:  return "continuation queued"
		case checkpoint.Running:  return "continuing…"
		case checkpoint.Failed:   return "continuation failed — retry available"
		}
	}
	switch thread.State { … }
}
```

The continuation request is consulted **before** `thread.State` and returns
outright, so any retained request shadows the row's real state for as long as it
is retained. The thread does not have to be unusable, parked or even idle — this
one is live and being typed into.

The retention itself is correct and deliberate. `pair#249` made a failed or
unconfirmed continuation survivable on purpose: *"Acceptance is not completion.
Registration proves a fresh target exists; `complete` requires the matching
orientation `submitted` receipt. A failed, canceled, or unconfirmed delivery
remains recoverable, and text may already be present in the target."* What was
never bounded is how long that fact outranks everything else the row could say.
This thread's request most likely dates from the `#256` M2 close continuation
(`731f99b7`), whose restart worked — the conversation continued — while the
`submitted` receipt never landed, so the request sits in `Failed` indefinitely.

**This is the same external/internal mismatch family as `pair#272` and
`pair#275`, from the other end:** not a stale liveness witness, but a stale
*request* outliving the situation it described.

## Spec

The row's state column shows the thread's state. A retained continuation is
additional information about that thread, not a replacement for it.

- **Compose, don't replace.** A live thread with a retained failed continuation
  reads as live *and* carries a marker that a retry is available. The fix is the
  precedence, not the message — "continuation failed — retry available" is the
  right words in the wrong slot.
- **Do NOT auto-resolve the request.** The tempting fix — a thread that has since
  gone live supersedes its request, so close it — is wrong and lossy. Per
  `pair#249`, a failed delivery may hold a checkpoint body that never reached
  any target; retry exists to observe or reattach rather than resubmit. Silently
  retiring the request would discard an undelivered handoff with no trace.
- **Give the operator an explicit dismissal instead.** If the row is going to
  carry the marker until the request reaches a terminal phase, there must be a
  gesture that says "I don't need this continuation" and records that decision.
  Otherwise the only exits are a successful retry or living with the marker
  forever.
- `Pending` and `Running` are arguably a different case: those describe an
  operation actually in flight, where displacing the state is defensible for the
  seconds it lasts. Decide that deliberately rather than by inheriting this
  switch's shape; the bug is `Failed`, which is not transient.

## Done when

- [x] A live thread with a retained failed continuation renders as live in the
      switcher, with the retry still discoverable.
- [x] A test pins the precedence across state × continuation-phase, so a future
      status cannot re-take the column by being added to the switch.
- [x] The retained request is never auto-retired by the thread's later liveness;
      an explicit operator dismissal exists, or the issue records why not.
- [x] `Pending`/`Running` precedence is settled on purpose and the reasoning is
      in the code.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only. `sdlc estimate-source` reports the calibration
source as stale, so the number is provisional. Only in-window work is counted:
the window started at the 21:46 claim. Line by line:

- issue-spec: the relaunch diagnosis, the operator decision and the plan.
- smaller-go-module ×4, each with ×0.2 design because the plan pre-resolves it:
  - the store transition, the operation and the couchcore entry;
  - the retry fix plus its production-seam test;
  - the switcher composition and its state × phase tables;
  - `ContinuationRefuses` and the guard tests.
- cross-cutting-refactor: the operation-table harness rows and the CLI policy.
- atlas-docs (×0.2).
- milestone-review: the close review.
- real-api-discovery: the operator's smoke on the live `pair` thread.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.50 impl=0.08
item: smaller-go-module design=0.06 impl=0.14
item: smaller-go-module design=0.06 impl=0.14
item: smaller-go-module design=0.06 impl=0.14
item: smaller-go-module design=0.06 impl=0.14
item: cross-cutting-refactor design=0.10 impl=0.14
item: atlas-docs design=0.03 impl=0.06
item: milestone-review design=0.10 impl=0.14
item: real-api-discovery design=0.00 impl=0.12
design-buffer: 0.15
total: 2.22
```

## Revisions

### 2026-09-17 — scope widened to the lifecycle refusal; dismissal decided

`## Spec` and `## Done when` stand. Deltas:

- **Dismissal = deletion** (operator decision, see `## Log`). The Spec's *"and
  records that decision"* is withdrawn: dismissing clears the request, and the
  checkpoint file is the only trace. The Done-when's dismissal bullet is met by
  the gesture existing.
- **Added to Done when:**
  - a live thread with a `Failed` request whose relaunch refuses names BOTH
    exits, and, once dismissed, relaunch passes the continuation guard;
  - Retry continuation from the switcher reaches its thread through the
    production dispatcher (a probable bug, logged 2026-09-17: the switcher
    sends `ref` and `tag`, which the resolver refuses).
- **Settled here, as Done-when bullet 4 asks:** `Pending`/`Running` keep
  displacing the state. They are in flight and bounded: `Running` fails at the
  30s submission deadline, and `Pending` is picked up by the owner's scan. Only
  `Failed` composes.
- **Kept out:** a `dismissed` phase, and a `Settled()` predicate over the ~12
  inline `!= Complete` sites. With no new phase, those sites still mean what
  they say.

### 2026-09-17 — close review round 1: two claims corrected

- **Settled differently from the first revision:** `Pending`/`Running` do NOT
  get to displace the state. "Bounded in time" holds only while an owner
  watches the address; a request whose owner died reads `continuing…` forever.
  Every phase now composes with the state. Done-when bullet 4 is answered by
  this rule and by the in-flight action set's stated reason (the continuation
  owns the thread mid-replacement).
- **"Every refusal names both exits"** is now true, not just asserted. The
  first implementation covered the guard, publish and `pair continue`, but not
  recovery, archive or warm reattach. All go through one wording
  (`checkpoint.Exits`), and each site is driven by a test.

## Plan

Durable plan: `workshop/plans/000280-dismiss-continuation-plan.md`. Single pass,
one boundary.

- [x] Confirm the `pair` request's phase and provenance: `failed`, 16:00:58,
      *"operator input interrupted automatic orientation"* (Log, 2026-09-17).
- [x] Reproduce the switcher retry bug through the production executor, then
      fix it (bootstrap `ref` optional; the switcher stops sending `ref`).
- [x] `ThreadStore.DismissFailedContinuation` + `dismiss-continuation`
      operation + `Couch.DismissContinuation`; refusals write nothing.
- [x] `continuationGuard` names retry and dismiss; a failed live thread
      relaunches past the guard once dismissed.
- [x] Switcher: `Failed` composes the state text; `Pending`/`Running` displace,
      on purpose; dismiss is offered beside retry; state × phase tables.
- [x] Atlas + README; full suite; operator smoke on the `pair` thread.

## Log

### 2026-09-17

- Filed from the operator's screenshot plus `couch --show pair` taken at the same
  moment; cause read directly from `menu_render.go:414`. No existing issue —
  `#150` is the only other open continuation issue and is unrelated.
- Surface divergence worth noting for **`pair#278`**: `couch --list` reports this
  thread `live` while the switcher reports `continuation failed`. Two operator
  surfaces, same store, different answers — #278 is already making `--list` the
  diagnostic view, and this is a second reason the two renderings need one rule.

### 2026-09-17 — claimed; a second, stronger symptom: the same request blocks relaunch

The operator could not relaunch this very `pair` thread (Alt+n). The refusal
comes from `continuationGuard` (`couchcore/continuation.go:366-371`), which
refuses whenever the retained request is not `Complete`: *"continuation … is
failed; use Retry continuation in Couch, or `couch --internal
retry-continuation …`"*. That guard gates relaunch (`relaunch.go:90`), cold
resume (`resume.go:453`), switch-agent (`switchagent.go:128`) and every non-warm
start claim (`threadstore.go:500-503`).

The record, read directly (`threadstore/records/e108517d46ab4575/couch-c945633f5c806f21.json`):
the request's phase is `failed`, created 16:00:58, with failure *"continuation
delivery cancelled: operator input interrupted automatic orientation; inspect
the existing target before retrying"*. The replacement agent DID start (target
pid 87027, launch ordinal 7). The operator typed into it before orientation
finished and has worked in the thread since. So retrying would push a
five-hour-stale handoff into a live conversation. Retry is the ONLY exit the state
machine has from `Failed` (`checkpoint/request.go`: `RetryAbsent` and
`RetryObserve`; no dismiss, abandon or superseded event).

**The class is wider than the row text.** A retained `Failed` request REPLACES
the thread in three places instead of composing with it:

1. **State text:** `rootStateText` returns before consulting `thread.State`
   (`menu_render.go:414-423`), as filed.
2. **Action set:** `menuActionItems` returns ONLY `retry-continuation`, `name`
   and `describe` for a non-unusable row (`menu.go:1228-1232`). A live thread
   loses detach, relaunch, park and switch-agent in the switcher.
3. **Lifecycle admission:** `continuationGuard`, above.

(3) is #249's deliberate fail-closed rule (*"Failed requests retain that
snapshot and require explicit retry"*). It is right to require an explicit
decision; it is wrong that retry is the only decision on offer. The missing
dismissal this issue's Spec already asks for is therefore not just cosmetic:
it is what makes the thread relaunchable again without a stale re-delivery.

**A DRY finding under all three:** about a dozen production sites spell "this
request still constrains the thread" as an inline `Phase != checkpoint.Complete`
(`continuation_recovery.go:14`, `recovery_execute.go:177,263,304`,
`continuation.go:187,367`, `continuation_store.go:22`, `threadstore.go:500`,
`detach.go:432`, `console_continuation.go:115,215`, `menu.go:1212,1228`). A new
terminal phase added site by site would leave the next one to find the same
trap. The terminal set belongs in `checkpoint`, as one predicate.

### 2026-09-17 — plumbing map for a dismiss operation, and a probable retry bug

**Probable existing bug: switcher Retry continuation cannot reach its thread.**
Read, not yet reproduced:
- `threadEffect` puts `tag` in the effect (`couchtty/menu.go:1887-1891`);
- `dispatchMenuOperation` then adds `ref` for `retry-continuation` (`:1773`);
- the production path (`wireResolver` → `CouchLiveOwnerExecutor` →
  `resolveOperationThread`) refuses both together: *"thread ref and exact tag
  cannot both be supplied"* (`couchcore/operationdispatch.go:421-423`).

The only switcher retry test (`console_continuation_test.go:17`) uses a fake
live-owner executor, so it never reaches that refusal. If this reproduces, a
failed row's one offered exit is dead too. Reproduce it before building on retry,
and do not copy the ref+tag pattern for dismiss.

**What a new operation must touch** (mapped by an exploration pass):
- **Declaration:** `ops.go:247-251`, argument helper `ops.go:423-429`.
- **Dispatch:** `operationdispatch.go:275-285`.
- **Switcher:** offer `menu.go:1211-1232`, confirm `:722-741`, label
  `:1375-1392`, result `:1663`, own-child `:1683-1689`, refresh `:1695-1704`,
  arguments `:1772-1777`, notice `:1827`; console effect `console.go:1583-1597`.
- **CLI:** generic through `cli.go:121-139`; owner policy `run.go:365-398`.
- **Harnesses that fail without a row:**
  - `TestOperationDeclarationsAreClosureFreeCompleteAndOwned`
    (`ops_declarations_test.go:8`);
  - `TestOperationArityMatchesExpectation` (`run_test.go:589`);
  - `TestRowActionDeclarationsAndTheMenuAgreeInBothDirections`
    (`menu_action_sweep_test.go:76`);
  - `TestAtlasDocumentsEveryTypedOperation` and `TestOperationPresentationDocs`
    (`readme_test.go:184,197`), which need `couch --internal <op>` in
    `atlas/couch.md`;
  - the hand lists `TestContinuationOperationsDeclared`
    (`continuation_ops_test.go:5`) and the owner-policy tests in
    `couchcmd/continuation_test.go`.
- **Confirmation pitfall:** a confirming operation on a NON-live row is dropped
  by the confirm guard (`menu.go:787-791`) and the refresh check
  (`:1566-1573`), which admit only live rows other than archive.
- **Untested today:** `rootStateText`'s continuation labels. No test sets
  `Continuation` (`menu_render_test.go:390,413`).

**Design constraint found while planning:** thread records decode with
`DisallowUnknownFields` (`strictjson/decode.go:23`, via
`threadrecord.DecodePersisted`), and the embedded request is validated on read
(`threadrecord/record.go:114-116`). A new phase VALUE or a new FIELD therefore
makes any pre-change binary reject the whole record. That includes the
long-running `pair` helpers, and the dismiss-then-relaunch flow parks through
one. Clearing the request is the only skew-safe dismissal. Operator decision
pending: clear the request, or add a `dismissed` phase.

### 2026-09-17 — operator decision: Dismiss continuation deletes the request

Operator: *"yes, add a 'Dismiss continuation' … keep it simple. if user choose
to dismiss, we don't need to keep track of previous behavior or failures."*
Dismissal CLEARS `record.Continuation`. There is no terminal phase, no audit
field and no diagnostics trail. The checkpoint's markdown file is left where it
is, untouched. This is also the only skew-safe shape (see the entry above).
Supersedes this issue's Spec wording *"records that decision"*, and the
Done-when's "or the issue records why not" is answered: it exists.

### 2026-09-17 — implemented; verification

**Retry bug, reproduced first.** `TestContinuationFailedRowKeepsAnExplicitRetry`
was moved onto the PRODUCTION dispatcher (`DispatchOperation` +
`CouchLiveOwnerExecutor`) over a real temp-dir store holding a `Failed` request.
It went red with *"thread ref and exact tag cannot both be supplied"*: the
switcher's Retry continuation could never reach its thread. Fixed by making the
bootstrap `ref` optional (name's shape) and dropping the switcher's `ref`. Both
continuation exits now address the row by its exact tag alone.

**Dismissal.** `ThreadStore.DismissFailedContinuation` (CAS, exact failed
request, deletes it) → `Couch.DismissContinuation` (the `requestRecord` +
stale-revision loop) → declared `dismiss-continuation` (direct store, metadata
effect, row action, internal). Every refusal of a failed request now names both
exits:
- `continuationGuard`;
- `PublishContinuation`, which had also been blocking the session-continuity
  writer on this thread;
- `pair continue --retry` on a hosted thread.

**Composition.** `rootStateText`: a `Failed` request renders
`<state> · continuation failed`; `Pending`/`Running` displace the state on
purpose (in flight, bounded). `menuActionItems`: a live `Failed` row keeps its
actions minus `couchcore.ContinuationRefuses` (relaunch, switch-agent, cold
resume, start), plus retry and dismiss, giving `detach, retry, dismiss, park,
name, describe`. Composition applies to live rows only, per the plan-quality
advisory: archive and warm reattach have their own admissions.

**Tests:**
- store transition refusals, each asserted by an unchanged revision;
- both retry/dismiss orders;
- the harness row in `TestEveryLifecycleTransitionIsDrivenIntoItsOwnRefusal`;
- `TestFailedContinuationRelaunchesOnceDismissed`: the refusal names both
  exits, and after dismissal the outcome is `Relaunched`;
- `TestContinuationRefusesMatchesTheGuardForEveryRowAction`: every declared row
  action is driven through the production dispatcher (listed ⇒ refused BY THE
  GUARD; unlisted ⇒ SUCCEEDS) or exempt with a reason;
- `TestPublishAfterDismissalAcceptsOnlyTheCurrentSource`;
- the switcher's state × phase text and action tables;
- a dismiss from the switcher that deletes the request on the real store.

**Mutation-checked**, each turning its test red:
- dropping the store's `Failed` precondition;
- removing relaunch from `ContinuationRefuses`;
- adding park to it;
- the switcher no longer filtering;
- the switcher sending `ref` again;
- the failed text replacing the state.

**Suites:** `go test ./... -count=1` passed (71 packages, exit 0, retention
scrub, sandbox off). `make -k test` failed only on the pre-existing
`test-changelog`; its Go recipe is skipped, which is why the suite above ran
separately. `make build` rebuilt `bin/couch` and `bin/pair` for the smoke.

### 2026-09-17 — operator smoke on the live `pair` thread

The operator killed the pre-change couch (pid 76239, from 20:02), relaunched
`couch` from this branch, and walked the smoke: *"ok, worked."* The live record
confirms the part that matters: `couch-c945633f5c806f21` is at revision 38 with
NO `continuation`. The failed 16:00 request is dismissed, and the thread is
`live` (incarnation pid 3683, `pair resume`, 22:27:33, couch's startup
reattach). A separate Alt+n relaunch after dismissal is not visible in the
record: the only incarnation dates from couch's start. That path is pinned by
`TestFailedContinuationRelaunchesOnceDismissed` (outcome `Relaunched`), and
the refusal it used to hit is gone with the request.

### 2026-09-17 — close review round 1 (REWORK): fixed by class

- **BR-2 (Critical):** the pure dismissal rule is extracted as
  `checkpoint.CheckDismissible`, tested on literal requests; the two IO entries
  moved to the plan's Integration points.
- **BR-3:** every continuation phase composes with the state. The in-flight
  action set keeps its restriction, for the reason stated in `## Revisions`.
- **BR-4:** every refusal a retained request causes names its exits through
  `checkpoint.Exits`:
  - guard, publish and `pair continue` (already);
  - plus recovery ×4, archive ×2 and warm reattach, via
    `withContinuationExits`.

  All are driven by `TestEveryRefusalARetainedRequestCausesNamesBothExits`,
  and cold resume and cold start by the guard-agreement test.
- **Minors:**
  - `menuLiveActions` placement;
  - one exits helper, and a shared `writeRequestRecord` loop;
  - `checkpoint.AllPhases` drives the phase table;
  - `operatorFacing` rename;
  - the console prunes `menu.Orientation` when a request vanishes;
  - `ContinuationRefuses`' doc names what is driven.
- **Mutation-checked (6 more, all red):**
  - the wrapper dropping the exits;
  - an unwrapped archive site;
  - orientation not pruned;
  - running displacing the state;
  - the pure rule admitting running;
  - resume unlisted.
- `go test ./... -count=1` passes (71 packages).
