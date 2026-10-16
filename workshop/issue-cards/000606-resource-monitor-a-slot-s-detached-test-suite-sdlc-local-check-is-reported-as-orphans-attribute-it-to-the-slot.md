---
id: 000606
status: codecomplete
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
actual_hours: 0.32
tracker:
    version: 1
    completion:
        token: close-113584a35748
        repository: github.com/xianxu/pair
        reviewed_head: ee39876a52fc7fc8a51be8fdc89e2e5dd1dc8f84
        evidence_commit: 79f9d576804864857a2d7d1f6a42eca986f0838a
---

# Resource monitor: a slot's detached test suite (sdlc local-check) is reported as orphans; attribute it to the slot

## Problem

pair#587's sampler alerts the hub about "orphans" that are really a slot's own test suite: 2026-10-15 23:5x,
"pair:6 pid=39601 name=couchcore.test orphan-rss 338 MB for 60 s". The process was a test binary of pair:6's recorded
suite (`sdlc local-check run --name suite`), which runs detached from the slot's process tree, so its test processes
aren't under the slot's wrapper. It exited a minute later. Every full suite on every slot will raise these.
