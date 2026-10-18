---
id: 000574
status: done
created: 2026-10-15
updated: 2026-10-17
estimate_hours: 3.19
github_issue:
started: 2026-10-17T18:43:29-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
actual_hours: 2.36
tracker:
    version: 1
    completion:
        token: close-03b900c614be
        repository: github.com/xianxu/pair
        reviewed_head: f962a74b9031836f2783f9d472cc395ff3fd5b57
        evidence_commit: 825175b3655885f65818d39f0d2522f6298f1ede
        landed_commit: 6d6e345763bcb00bc044e071695ad6287b12a4d3
---

# Couch tells the hub about slot lifecycle events (start, park, relaunch, exit, recovery) as system messages

## Problem

The hub (TL) isn't told about slot lifecycle events. When the operator parks a slot (Alt+n), relaunches it, brings a
new slot up (parley.nvim:3 on 2026-10-15), or a slot's agent exits, the TL learns it only by polling `couch --actors`,
by peeking, or from a worker's own report. Tonight's evidence: stalled slots went unnoticed for hours, and messages to a
relaunched slot were lost. The reverse direction already exists: on connect, Couch sends a worker a "Couch restarted
and the hub is available…" notice, attributed to ops:0.
