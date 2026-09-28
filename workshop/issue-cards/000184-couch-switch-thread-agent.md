---
id: '000184'
status: done
started: 2026-09-13T12:04:32-07:00
created: 2026-09-04
updated: 2026-09-13
estimate_hours: 6.65
actual_hours: 6.03
---

# couch: switch a thread's agent

## Problem

A thread's agent is pinned by its saved launch profile. Every couch path that
brings a thread back -- resume, and the relaunch #182 is adding -- reads
`record.LatestLaunchProfile.Agent` and re-launches that same agent
(`couchcore/resume.go:187`, `CheckResumePreconditions`). There is no couch
gesture that says "same work, different driver".

The operator's escape today is to leave couch: park or quit the thread, drop to
a shell, and run `pair <agent>` -- which #115 taught to route an explicit-agent
request onto existing work. But #159 made the couch TUI the public CLI, so the
one place the operator actually lives is the one place the move is missing.
That matters exactly when it is needed: a degraded provider, an exhausted quota,
or a deliberate choice to put a different model on this particular thread.

Two neighbours bound this issue and neither covers it:

- **#115 (done)** built the launcher-level substrate -- repo-scoped per-agent
  launch defaults and explicit-agent picker routing. It runs from a shell, and
  deliberately refuses to take over a live foreign-agent session.
- **#135 (open)** is the live handoff: quiesce a running source agent and hand
  the tag to a target agent while it is still up. That is the hard, deferred
  case.

This issue is the cold, safe one, at the couch layer: park the thread, then
bring it back under a different agent, in place.
