---
id: '000182'
status: done
started: 2026-09-04T09:16:38-07:00
created: 2026-09-03
updated: 2026-09-04
estimate_hours: 6.20
actual_hours: 10.96
---

# Relaunch an actor: restart Pair in place, keeping the agent conversation

## Problem

Developing Pair while working inside it has no clean restart. Rebuild the
binary and the running actor keeps the old code; the only way to pick up a new
Pair is to lose the session, and losing the session means losing the agent
conversation with it.

couch already has the symmetric move one level up: `alt+d` detaches everything
and leaves, so rebuilding and re-running `couch` picks up new couch code with
every agent still running behind its zellij session. The actor level has no
equivalent -- which is the level where Pair development actually happens.

Pair's own `Alt+n` (`ActionRestartPair`, "reload pair -- kill and re-launch the
workbench in place") is close but not the same thing: it restarts the workbench
*inside* the session couch's child already handed off to. The process couch
spawned is not replaced, so a rebuilt Pair binary is not what comes back.

Concretely: `launcher/restart.go` writes a restart marker, touches the quit
marker and execs `kill-session`; the already-running Pair process's outer loop
then regains control and re-enters its create flow. The zellij session is
replaced, the PROCESS is not.

**The gesture is the point.** `Alt+n` is already the operator's reflex for
exactly this intent -- "reload pair, keep the conversation". Inside couch it
silently does the weaker thing, and the operator gets no signal that the chord
means something different here. Making `Alt+n` mean the same thing one layer up
is the feature; the operation below it is what makes that honest.
