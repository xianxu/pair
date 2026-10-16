---
id: 000606
status: done
created: 2026-10-15
updated: 2026-10-16
estimate_hours:
github_issue:
started: 2026-10-16T12:04:56-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
actual_hours: 0.87
tracker:
    version: 1
    completion:
        token: close-3ae8d21db883
        repository: github.com/xianxu/pair
        reviewed_head: 3532f6d70fdb4d4af46f6fecdbbb7fe591e66664
        evidence_commit: c0495c9be73f735da9f1b75d3d03cf857b9afcbc
        landed_commit: a75e0ae456b881681de9fda8bd3a509a9fb1b16e
---

# Resource monitor: a slot's detached test suite (sdlc local-check) is reported as orphans; attribute it to the slot

## Problem

pair#587's sampler alerts the hub about "orphans" that are really a slot's own test suite: 2026-10-15 23:5x,
"pair:6 pid=39601 name=couchcore.test orphan-rss 338 MB for 60 s". The process was a test binary of pair:6's recorded
suite (`sdlc local-check run --name suite`), which runs detached from the slot's process tree, so its test processes
aren't under the slot's wrapper. It exited a minute later. Every full suite on every slot will raise these.
