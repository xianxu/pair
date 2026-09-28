---
id: 000329
status: done
created: 2026-09-25
updated: 2026-09-28
estimate_hours:
github_issue:
started: 2026-09-28T11:21:37-07:00
actual_hours: 0.64
tracker:
    version: 1
    completion:
        token: close-dd2b41c7fa9a
        repository: github.com/xianxu/pair
        reviewed_head: df4a9496db753dd994162cbc352b3f7828a9366b
        evidence_commit: 49658399a764c6445549ab7294cf2a415d5ae3e9
        landed_commit: 450c06b33abe0d34fbeff01a7acb498cb149740b
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
