---
id: 000280
status: working
deps: []
github_issue:
created: 2026-09-17
updated: 2026-09-17
estimate_hours:
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

- [ ] A live thread with a retained failed continuation renders as live in the
      switcher, with the retry still discoverable.
- [ ] A test pins the precedence across state × continuation-phase, so a future
      status cannot re-take the column by being added to the switch.
- [ ] The retained request is never auto-retired by the thread's later liveness;
      an explicit operator dismissal exists, or the issue records why not.
- [ ] `Pending`/`Running` precedence is settled on purpose and the reasoning is
      in the code.

## Plan

- [ ] Confirm the `pair` request's phase and provenance in the thread record
      (expected: the `#256` M2 close continuation, accepted, never `complete`).
- [ ] Decide the composed row shape — state column plus marker — and how it
      renders at narrow widths.
- [ ] Implement the precedence change; table test over state × phase.
- [ ] Dismissal gesture, or a recorded decision not to have one.

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
