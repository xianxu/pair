---
id: 000739
status: codecomplete
created: 2026-10-20
updated: 2026-10-21
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
actual_hours: 0.39
tracker:
    version: 1
    completion:
        token: close-7a8e08cfad10
        repository: github.com/xianxu/pair
        reviewed_head: 6792c49527b9cb76acd7efb4dbe53b717bbd9b33
        evidence_commit: 22a0a28abe53304c9b04ccc1a348e26fc58d4a17
---

# couch: start-up safe mode: no automatic injection into the hub, restart kind reported not acted on, one generic start notice per worker

## Problem

After a restart, Couch acts on its own: it injects system notices into every slot, the hub included ("Couch
restarted and the hub is available…"), and recovery has been attempted automatically at times. The operator (10-20,
drain redesign, ops side doc `2026-10-20-17-drain-redesign.md`): no automatic recovery; a restart may be for
something else, or the setup may be wedged; the operator tells the TL "start recovery" in one line.
