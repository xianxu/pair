---
id: 000673
status: done
created: 2026-10-18
updated: 2026-10-19
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
actual_hours: 4.17
tracker:
    version: 1
    completion:
        token: close-833c2cca86c9
        repository: github.com/xianxu/pair
        reviewed_head: 6d00d20b44696671955e7d550d02c143c8de1e27
        evidence_commit: baa13bcdda422199338ec6b5c25fe0da6e749e8a
        landed_commit: 38bf8d5e20b3955ce2295c0291200b566fb62d4e
---

# Couch attention list: a top-right panel of what's waiting on the operator, each item jumps to its slot

## Problem

Ghostty skips OSC desktop notifications sent from the focused tab (confirmed by the operator 2026-10-18), and Couch's hub is nearly always the focused tab, so `couch --notify` never reaches the operator in practice. The operator wants Couch to manage attention itself.
