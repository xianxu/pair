---
id: 000673
status: working
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
github_issue:
started: 2026-10-18T12:05:17-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
---

# Couch attention list: a top-right panel of what's waiting on the operator, each item jumps to its slot

## Problem

Ghostty skips OSC desktop notifications sent from the focused tab (confirmed by the operator 2026-10-18), and Couch's hub is nearly always the focused tab, so `couch --notify` never reaches the operator in practice. The operator wants Couch to manage attention itself.
