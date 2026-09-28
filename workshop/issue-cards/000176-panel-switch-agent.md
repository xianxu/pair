---
id: '000176'
status: wontfix
created: 2026-09-02
updated: 2026-09-07
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

**And restart's purpose is refreshing the pair subsystem, not the agent.** The
value is a genuinely new zellij session, nvim panes, and `pair-wrap` — the agent
restarting is a consequence. pair's own chord table already draws this line:
`ChordAltN → ActionRestartPair` versus `ChordAltShiftN → ActionRestartAgent`
(`workbenchshortcut/shortcut.go:124,128`). The panel operation is the *former*.
An implementation that replaces only the agent child would satisfy every other
sentence in this issue and miss the point of it (`ARCH-PURPOSE`).

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
