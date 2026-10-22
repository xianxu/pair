---
id: 000722
status: codecomplete
created: 2026-10-19
updated: 2026-10-22
estimate_hours: 14.947
github_issue:
started: 2026-10-19T23:16:03-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
actual_hours: 16.33
tracker:
    version: 1
    completion:
        token: close-bbd4eded21d1
        repository: github.com/xianxu/pair
        reviewed_head: 903a134c3603877e6eedc662effa1b56448c5117
        evidence_commit: a75ef8fb911db7e41f0fa754b909b0ed4252b539
---

# couch --drain [--fast]: a fleet-wide checkpoint-and-stop before a restart, with a drain board

## Problem

There is no safe, on-demand moment to restart the machine or Couch. Work never ends, so "safe to restart" has to be a
state the fleet can reach on request. Today a restart is a crash (10-19 15:55) or the TL pinging slots one by one,
and nobody can read off whether it's safe. Operator item 15 (10-19), discussion:
ops `teams/devinfra/tl-status-xian-xu.side/2026-10-19-15-fleet-drain.md`.
