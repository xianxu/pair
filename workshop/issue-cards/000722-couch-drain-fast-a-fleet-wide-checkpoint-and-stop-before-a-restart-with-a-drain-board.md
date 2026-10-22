---
id: 000722
status: codecomplete
created: 2026-10-19
updated: 2026-10-22
estimate_hours: 18.3755
github_issue:
started: 2026-10-19T23:16:03-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
actual_hours: 33.40
tracker:
    version: 1
    completion:
        token: close-1e4728003a6a
        repository: github.com/xianxu/pair
        reviewed_head: b2c541b239de1db1c2feace42e731b16d8cd1069
        evidence_commit: 351bf4d2905c07d09dc44d03b0ef1646bc7b78a6
---

# couch --drain [--fast]: a fleet-wide checkpoint-and-stop before a restart, with a drain board

## Problem

There is no safe, on-demand moment to restart the machine or Couch. Work never ends, so "safe to restart" has to be a
state the fleet can reach on request. Today a restart is a crash (10-19 15:55) or the TL pinging slots one by one,
and nobody can read off whether it's safe. Operator item 15 (10-19), discussion:
ops `teams/devinfra/tl-status-xian-xu.side/2026-10-19-15-fleet-drain.md`.
