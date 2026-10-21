---
id: 000747
status: done
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
github_issue:
started: 2026-10-20T17:17:34-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
actual_hours: 0.31
tracker:
    version: 1
    completion:
        token: close-0d58aa04536a
        repository: github.com/xianxu/pair
        reviewed_head: 0af77fd818b77daf156d5e2e879124e9db66c01e
        evidence_commit: 65b203efe127cd1aaa451e8ed85d994598c3803a
        landed_commit: ac6a85bd113700168eb83a85937f9e98360b20b1
---

# Recordless Fresh worker launch validates a reserved address as established

## Problem

The #744 shared native disposable fleet provisions a genuine numbered worker through real SDLC/Weave, starts a real steward and invokes couch --reboot fixture:1 --confirm --json from that steward's actual inherited environment. Recordless Fresh is admitted but Pair exits before registering: existing Couch thread registration does not match requested established address.
