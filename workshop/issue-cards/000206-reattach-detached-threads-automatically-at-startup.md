---
id: '000206'
status: done
started: 2026-09-10T20:16:19-07:00
created: 2026-09-06
updated: 2026-09-12
estimate_hours: 3.13
actual_hours: 8.96
---

# reattach detached threads automatically at startup

## Problem

Starting couch does not bring back the threads that are still running behind
client-less zellij sessions. A detached thread's agent is alive — `couchcmd/run.go:641`
describes the state as *"detached (no client attached; the agent is still
running)"* — but the operator has to go find each one in the switcher and
reattach it by hand.

With the session populations this fleet actually runs (10–11 live sessions
observed 2026-09-06), that is the first thing the operator does every time, and
it is pure ceremony: the state needed to do it automatically is already in the
ThreadStore before anything launches.

**The feature is the auto-reattach. How it loads is a separate decision**, and
this issue deliberately does not pre-commit to one — see Spec.
