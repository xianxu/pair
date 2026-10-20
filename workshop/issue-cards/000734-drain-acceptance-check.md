---
id: 000734
status: wontfix
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
github_issue:
started: 2026-10-20T12:21:03-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
tracker:
    version: 1
    abandoned: {}
---

# Declare opt-in drain acceptance local check

## Problem

#722's approved re-plan needs a named opt-in disposable-Couch acceptance matrix recorded like a suite. The recorder reads declarations only from main. Adding that declaration changes protected local gates, so it belongs in a separate operator-verified configuration landing rather than #722's product implementation.
