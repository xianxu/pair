---
id: 000176
status: open
deps: []
github_issue:
created: 2026-09-02
updated: 2026-09-02
estimate_hours:
---

# Switch or restart an actor's agent from the panel

## Problem

Changing which agent drives a thread is possible but awkward, and only from
inside pair: quit with `alt+x`, relaunch under the same tag, and answer the
startup prompt about carrying work over. Restarting the same agent has its own
separate chords (`ChordAltN → ActionRestartPair`, `ChordAltShiftN →
ActionRestartAgent`). None of it is reachable from couch's panel, which is where
the operator is already looking at the actor they want to change.

**These are one operation.** Restart is "switch to the same agent" — stop the
current driver, start a new one on the same thread, and give the new session a
way to pick up where the old one was.

**The prior art matters here.** `pair#115` (done) established the model — *the
tag identifies the work; the agent is an exclusive, replaceable driver* — and
shipped switching for an **exited or recent** tag via continuation-backed
recovery, preserving draft pane, sent-prompt history, queue, and native
per-agent conversations. `pair#135` (open) is the remaining case, taking over a
**currently live** session, and it records exactly why the first attempt was
abandoned:

> the acceptance fake modeled cleanup behavior real zellij did not provide, so
> the source could be destroyed and then time out … Quiescence evidence must be
> observable by the coordinator itself, not only by an acknowledgment emitted
> from a process that may already be gone.

**couch dissolves that.** The coordinator was inside the thing being killed,
which is why the proof was unsound. couch is *outside* it: it owns the child's
pty, and its liveness test is already a closed channel rather than `kill -0` —
deliberately, because `kill -0` succeeds for a zombie and would report an
exited-but-unreaped child as running. That is a first-party quiescence proof of
exactly the kind #135 said was missing.

## Spec

**One panel operation, `switch-agent`, with the target agent as an argument.**
Restart is the same call with target == current. Do not build two commands that
diverge.

Declare it beside the others in `couchcore/ops.go` with a row in the
declaration table (`ops_declarations_test.go`). Shape: `ExecuteLiveOwner`,
`EffectProcess`, **`ConfirmRequired`** — it destroys a live session, the same
grounds on which `stop` is the one existing `ConfirmRequired` operation.
`#159` made the TUI the public CLI, so declaring it once yields both surfaces.

**Sequence:**

1. Quiesce the current driver, with couch observing the exit itself — not an
   acknowledgment from the process being stopped.
2. Start a new pair under the **same thread and tag**. Identity is the thread;
   the agent is a replaceable driver (`#115`).
3. Hand the new session a pointer to the old transcript.

### Context carrier: the transcript pointer, not a continuation

`#115` carries context via a continuation — distilled, but it costs a model turn
and **requires the source agent to still work**. That is precisely wrong for the
motivating cases, which #135 lists as a degraded provider, exhausted quota, or
an agent that "can no longer produce the continuation document itself."

So the carrier here is a **path to the prior session's transcript**, handed to
the new session to read if it wants. Free, requires nothing of the dying agent,
and raw rather than distilled — a worse summary and a strictly more available
one. Where the source agent is healthy, a continuation is still the better
carrier; the two are complementary, and this issue owns the fallback.

**The hazard to solve, not discover:** the scrollback paths are per-tag
(`PAIR_SCROLLBACK_RAW_PATH`, `PAIR_SCROLLBACK_ANSI_PATH`,
`PAIR_LOG_PATH`). A new session under the *same tag* resolves to the *same
paths* and will write over them. Pointing the new session at "the old
transcript" therefore names a file the new session is already overwriting.
Rotate or snapshot before the new driver starts, and hand over the rotated path.

## Done when

- One panel operation switches a live actor from claude to codex and back,
  under the same thread, tag, and working path — no new thread record.
- The same operation with target == current agent restarts in place.
- Quiescence is proven by couch observing the child's exit, not by an
  acknowledgment from the stopped process; a source that dies without
  acknowledging still completes cleanly. This is #135's failure, so it is the
  acceptance test, not a note.
- The new session is told the path of the **prior** transcript, and that file
  still holds the prior session's content after the new one has started writing.
- A failed switch leaves the thread usable — no lock or claim that permanently
  disables it (`#135`'s requirement, carried forward).
- The operation is `ConfirmRequired` and declared in the same table as the
  others; no second command path for "restart".

## Plan

- [ ] Declare `switch-agent` in `ops.go` + the declaration table.
- [ ] Quiesce-and-observe on couch's side; test a source that dies silently.
- [ ] Transcript rotation before the new driver starts, and the handover of the
      rotated path.
- [ ] Restart = same call, target == current; assert one code path.
- [ ] Failure path: assert the thread is still startable after a failed switch.

## Log

### 2026-09-02

Raised as two commands — "switch agent" and "restart" — and collapsed to one on
the operator's own observation that restart is switching to the same agent.

The load-bearing insight is that couch's outer-shell position supplies the
quiescence proof `#135` could not build from inside pair. `#135` should be
re-evaluated when this lands: its spec is still right about *what* must be
proven, but its coordinator lives in the wrong process. Not repointed here —
that is a call for the operator, since #135 also carries requirements about
tag-scoped state that this issue inherits rather than replaces.
