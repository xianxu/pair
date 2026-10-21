---
id: 000751
status: done
created: 2026-10-20
updated: 2026-10-21
estimate_hours: 3.79
github_issue:
started: 2026-10-20T21:40:18-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:9
    worktree: /Users/xianxu/workspace/worktree/pair-slot9/pair
    repository: github.com/xianxu/pair
actual_hours: 2.04
tracker:
    version: 1
    completion:
        token: close-1c004f181433
        repository: github.com/xianxu/pair
        reviewed_head: db8caaeba35cb82f5d4e3ac8ba0ca20397f5c185
        evidence_commit: b3e26855a190b1f4fcbc09f459dfdc4f2c02e646
        landed_commit: 6a615f28c20aef8789243c7a2098ecb7e2b631ea
---

# cast-sketch: recording model, pressure paths and undo history

## Problem

Implement the approved cast-sketch recording model before its drawing consumers. Existing pointer trails must retain their current fading semantics; sketches are persistent, undoable design evidence.
