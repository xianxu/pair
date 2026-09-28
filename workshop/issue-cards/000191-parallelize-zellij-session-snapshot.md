---
id: 000191
status: open
created: 2026-09-02
updated: 2026-09-11
estimate_hours:
github_issue:
---

# Parallelize the zellij session snapshot

## Problem

`ZellijSource.Snapshot` polls each live session serially for its client count.
Measured on a 19-session host (6 exited, 13 live) on 2026-09-02:

- whole snapshot: **1.49 s**
- two `list-sessions` runs: ~16 ms each
- one `action list-clients` per live session: ~100 ms each, 13 of them = 1.43 s

Since `pair#170` M2, Couch's actionable inventory takes this snapshot whenever a
detach candidate exists, and M3 put that inventory on the **blocking** startup
path: `StartInteractive` must decide resume-vs-new before it attaches anything,
so `couch` in a directory pays ~1.4 s before the first frame. M2 also made
*detached* the normal resting state, so this is the ordinary case rather than an
edge one.

The switcher itself is unaffected — refreshes are event-driven and run on the
single-flight worker while the menu renders its last-good projection — so this
is a startup-latency issue, not a keystroke-budget one.
