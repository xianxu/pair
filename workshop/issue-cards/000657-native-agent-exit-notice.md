---
id: 000657
status: open
created: 2026-10-17
updated: 2026-10-18
estimate_hours:
github_issue:
started: 2026-10-18T07:05:50-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
---

# Couch reports native agent exit without mistaking attach disconnect

## Problem

#574 cannot report a native agent exit reliably from the existing Couch observations. Console.onExit observes the Pair attachment client, and registry EOF may be a reconnect or wrapper replacement. Neither proves that the native agent stopped.
