---
id: 000657
status: codecomplete
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
actual_hours: 1.13
tracker:
    version: 1
    completion:
        token: close-9f1571702966
        repository: github.com/xianxu/pair
        reviewed_head: 137152dbba61d1b5ccded1affccc45725c92a21e
        evidence_commit: 10618e266f975dbc8759c6d9c1a92bfd7474f7e6
---

# Couch reports native agent exit without mistaking attach disconnect

## Problem

#574 cannot report a native agent exit reliably from the existing Couch observations. Console.onExit observes the Pair attachment client, and registry EOF may be a reconnect or wrapper replacement. Neither proves that the native agent stopped.
