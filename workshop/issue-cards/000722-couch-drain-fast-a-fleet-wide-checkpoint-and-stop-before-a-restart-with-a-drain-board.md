---
id: 000722
status: codecomplete
created: 2026-10-19
updated: 2026-10-20
estimate_hours: 9.6185
github_issue:
started: 2026-10-19T23:16:03-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
actual_hours: 8.64
tracker:
    version: 1
    completion:
        token: close-3ee41a2ef83b
        repository: github.com/xianxu/pair
        reviewed_head: 09140407b75d866a912c26cb32e435de2451e4ee
        evidence_commit: 63645ddd10146e9987ff18661f8920abaa3c30fd
---

# couch --drain [--fast]: a fleet-wide checkpoint-and-stop before a restart, with a drain board

## Problem

There is no safe, on-demand moment to restart the machine or Couch. Work never ends, so "safe to restart" has to be a
state the fleet can reach on request. Today a restart is a crash (10-19 15:55) or the TL pinging slots one by one,
and nobody can read off whether it's safe. Operator item 15 (10-19), discussion:
ops `teams/devinfra/tl-status-xian-xu.side/2026-10-19-15-fleet-drain.md`.
