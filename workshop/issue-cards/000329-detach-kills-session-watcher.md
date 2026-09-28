---
id: 000329
status: open
created: 2026-09-25
updated: 2026-09-25
estimate_hours:
github_issue:
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
