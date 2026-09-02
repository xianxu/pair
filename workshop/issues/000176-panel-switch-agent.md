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

**The prior transcript is already preserved — two mechanisms, neither needing
work.** Tags are opaque and stable across a restart (the tag *is* the work
identity), so the collision question is settled elsewhere:

- Per-agent artifacts carry the agent in the filename —
  `scrollback-<tag>-<agent>.raw` — so a claude→codex switch writes a different
  file by construction. Tag-scoped artifacts (draft, log, ledger, queue) are
  deliberately shared, which is the `#115` model.
- pair's quit path already **moves** the scrollback to a timestamped
  `parked-scrollback-<tag>-<ts>` base — "move on quit, copy on compaction"
  (`launcher/osruntime.go:841-856`, `runtime.go:219`) — so a same-agent restart
  also leaves the prior content intact.

So the work is **selection, not rotation**: those snapshots accumulate per tag,
and the handover must name *which* `parked-scrollback-<tag>-<ts>` belongs to the
session just replaced. Reuse `ParkedScrollbackArtifacts` rather than adding a
second snapshot path (`ARCH-DRY`).

## Done when

- One panel operation switches a live actor from claude to codex and back,
  under the same thread, tag, and working path — no new thread record.
- The same operation with target == current agent restarts in place.
- Quiescence is proven by couch observing the child's exit, not by an
  acknowledgment from the stopped process; a source that dies without
  acknowledging still completes cleanly. This is #135's failure, so it is the
  acceptance test, not a note.
- The new session is told the exact `parked-scrollback-<tag>-<ts>` of the
  session it replaced — not merely the newest file matching the tag — asserted
  with earlier snapshots already present for that tag.
- A failed switch leaves the thread usable — no lock or claim that permanently
  disables it (`#135`'s requirement, carried forward).
- The operation is `ConfirmRequired` and declared in the same table as the
  others; no second command path for "restart".

## Plan

- [ ] Declare `switch-agent` in `ops.go` + the declaration table.
- [ ] Quiesce-and-observe on couch's side; test a source that dies silently.
- [ ] Identify the replaced session's `parked-scrollback-<tag>-<ts>` via
      `ParkedScrollbackArtifacts` and hand that exact path over.
- [ ] Restart = same call, target == current; assert one code path.
- [ ] Failure path: assert the thread is still startable after a failed switch.

## Log

### 2026-09-02

Correction, recorded so it is not re-derived: an earlier draft of this issue
claimed the new session would overwrite the prior transcript and asked for
rotation. Wrong on both paths — the agent suffix separates a cross-agent
switch, and quit already moves the scrollback to a timestamped
`parked-scrollback-` base. What remains is picking the right snapshot.

Raised as two commands — "switch agent" and "restart" — and collapsed to one on
the operator's own observation that restart is switching to the same agent.

The load-bearing insight is that couch's outer-shell position supplies the
quiescence proof `#135` could not build from inside pair. `#135` should be
re-evaluated when this lands: its spec is still right about *what* must be
proven, but its coordinator lives in the wrong process. Not repointed here —
that is a call for the operator, since #135 also carries requirements about
tag-scoped state that this issue inherits rather than replaces.
