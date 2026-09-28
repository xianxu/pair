---
id: '000230'
status: done
started: 2026-09-11T09:55:31-07:00
created: 2026-09-11
updated: 2026-09-11
estimate_hours: 1.25
actual_hours: 2.42
---

# A failed warm reattach deletes the session it was reattaching

## Problem

**A warm reattach that fails after its child is acknowledged deletes the zellij
session it was reattaching, and with it the running agent.** A warm reattach
exists to preserve that agent.

The path, from the code:

1. `ResumeContext` proves the thread detached and calls `launchTrackedThread`
   with `Resume: true, Warm: true`. The child is a bare `pair resume <tag>`,
   which attaches a client to the surviving session.
2. After `h.Acknowledge()`, every failure goes to
   `failTrackedPostAckStart(in.Resume, …)` (`launch_existing.go`). That covers
   the context being cancelled, the 15 s Pair registration timeout, and a
   failed `AdvanceStart`. The function takes `Resume` but not `Warm`.
3. For any resume, it calls `quiescePostAckStart`, which ends the helper and
   then calls `c.Artifacts.Quiesce(address)`.
4. In production that is `launcher.QuiesceThreadSession`, which deletes the
   thread's bound session: `zellij delete-session --force`, plus a kill of
   that session's server (`session_quiescence.go`).

Quiescing is right for a **cold** resume, which created that session and owns
cleaning it up. For a warm one, the session predates the attempt.

**Evidence.** A temporary test at the fake seam (2026-09-11) cancelled a warm
`ResumeContext` from the runner's `AfterAcknowledge` hook. The fake checker
recorded a `Quiesce` of the thread's address. The resume returned `context
canceled`, and the binding was then absent. No existing test covers a warm
reattach failing after acknowledgement.

**Reachable today:**
- Closing the terminal (SIGHUP) or sending SIGTERM while an operator's resume
  waits for Pair to register: `Run` returns, `teardown` cancels the lifetime
  context, and the in-flight resume takes the post-ack path.
- A warm reattach whose `pair resume` takes longer than the registration
  timeout to register, for example on a loaded host. Startup's own resume of
  the cwd thread is exposed to this too.

**#206 makes it much more reachable.** Its background pass reattaches every
detached thread one after another for several seconds after each startup, and
the operator's quit gestures and the last-pane exit are not held off while it
runs.
