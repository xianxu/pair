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
actual_hours: 0.38
tracker:
    version: 1
    completion:
        token: close-be31f5cf218d
        repository: github.com/xianxu/pair
        reviewed_head: 63babdc3e4dec229d2035224cc77000da456b7ea
        evidence_commit: 7fea67d661ed856948b0c9882caf49e491325031
---

# Resource monitor: a slot's detached test suite (sdlc local-check) is reported as orphans; attribute it to the slot

## Problem

pair#587's sampler alerts the hub about "orphans" that are really a slot's own test suite: 2026-10-15 23:5x,
"pair:6 pid=39601 name=couchcore.test orphan-rss 338 MB for 60 s". The process was a test binary of pair:6's recorded
suite (`sdlc local-check run --name suite`), which runs detached from the slot's process tree, so its test processes
aren't under the slot's wrapper. It exited a minute later. Every full suite on every slot will raise these.
