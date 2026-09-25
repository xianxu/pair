---
id: 000329
status: open
deps: []
github_issue:
created: 2026-09-25
updated: 2026-09-25
estimate_hours:
---

# Detach kills an unbound session watcher

## Problem

A thread that starts a new conversation needs a turn before it can be
relaunched. The `pair session-watch` sidecar waits for the agent's first
completed turn and writes the ledger binding. For a Couch-launched pair, that
sidecar stays in the actor's process group on purpose
(`cmd/internal/launcher/osruntime.go:461`), so Couch can clean up a failed
start.

Detach ends that group (`cmd/internal/couchcore/detach.go:27`: "the
session-watcher and title-poller sidecars sharing its process group go with
them"). Reattach is a `pair resume <tag>` attach: it restarts the title poller
(`cmd/internal/launcher/lifecycle.go:108`) but never the watcher. Only the
create path starts one (`cmd/internal/launcher/createflow.go:784`).

So a thread detached (or through a Couch restart) before its first completed
turn stays unbound for good. Alt+n refuses it with "its agent has not
completed a turn yet", however many turns it has taken since.

Observed 2026-09-25 on three live threads: `ariadne` (codex), `ariadne-slot1`
and `ariadne-slot2`. In each, the ledger's current launch has no binding, and
no session watcher is running for it. `ariadne-slot1` (`couch-6ab16bffb21c607a`)
launched claude at 2026-09-24 15:00 with `--session-id 30512fb8…`. Its first
message came at 19:48, followed by hours of turns, and it's still unbound.

## Spec

Options (to decide at design time):

- **Resolve on demand (preferred).** When relaunch finds the current launch
  unbound, run the watcher's own check right then: a completed round after the
  launch boundary, matched to a Pair send and corroborated by the live agent
  process. For a live thread, all of that exists when the operator presses
  alt+n. The watcher becomes a way to have the answer ready sooner, not the
  only thing that can produce it.
- **Run the watcher inside the zellij session**, as `pair wrap` already does
  for a fresh conversation (`cmd/internal/wrapcmd/wrap.go:2443`). It then lives
  as long as the agent, not the client. Couch's failed-start cleanup needs a
  different mechanism.
- **Restart the watcher on attach when the current launch is unbound.** This
  must pass the ORIGINAL launch's start time as `--pid-not-before`. Otherwise
  the old agent-pid file reads as stale, the watcher finds no agent and quits
  after its 60 s startup deadline.

## Done when

- Regression test: launch, detach before the first turn, reattach, complete a
  turn, and the thread must be relaunchable.
- The three stuck threads above (or their equivalents) become relaunchable
  after one turn, without manual repair.

## Plan

- [ ]

## Log

### 2026-09-25

- Found while diagnosing #328. The tools thread's refusal was #328, a
  different cause that leads to the same message.
