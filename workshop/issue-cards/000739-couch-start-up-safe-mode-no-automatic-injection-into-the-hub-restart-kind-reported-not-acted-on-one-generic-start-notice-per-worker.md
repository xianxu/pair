---
id: 000739
status: codecomplete
created: 2026-10-20
updated: 2026-10-23
estimate_hours: 4.695
github_issue:
started: 2026-10-20T22:40:29-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
actual_hours: 5.79
tracker:
    version: 1
    completion:
        token: close-5a28598f1074
        repository: github.com/xianxu/pair
        reviewed_head: d2da63f3df609ea04cea240524d9c083d8c5c4db
        evidence_commit: 2e98acbf45cc4836866e6fa6448f9ae1e8329dc1
---

# couch: start-up safe mode: no automatic injection into the hub, restart kind reported not acted on, one generic start notice per worker

## Problem

After a restart, Couch acts on its own: it injects system notices into every slot, the hub included ("Couch
restarted and the hub is available…"), and recovery has been attempted automatically at times. The operator (10-20,
drain redesign, ops side doc `2026-10-20-17-drain-redesign.md`): no automatic recovery; a restart may be for
something else, or the setup may be wedged; the operator tells the TL "start recovery" in one line.
