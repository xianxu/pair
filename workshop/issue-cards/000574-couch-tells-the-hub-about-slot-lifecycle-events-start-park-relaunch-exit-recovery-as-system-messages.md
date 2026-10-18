---
id: 000574
status: open
created: 2026-10-15
updated: 2026-10-17
estimate_hours:
github_issue:
started: 2026-10-17T18:43:29-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
---

# Couch tells the hub about slot lifecycle events (start, park, relaunch, exit, recovery) as system messages

## Problem

The hub (TL) isn't told about slot lifecycle events. When the operator parks a slot (Alt+n), relaunches it, brings a
new slot up (parley.nvim:3 on 2026-10-15), or a slot's agent exits, the TL learns it only by polling `couch --actors`,
by peeking, or from a worker's own report. Tonight's evidence: stalled slots went unnoticed for hours, and messages to a
relaunched slot were lost. The reverse direction already exists: on connect, Couch sends a worker a "Couch restarted
and the hub is available…" notice, attributed to ops:0.
