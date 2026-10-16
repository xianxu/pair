---
id: 000587
status: working
created: 2026-10-15
updated: 2026-10-15
estimate_hours: 4.883
github_issue:
started: 2026-10-15T16:54:15-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
---

# Couch resource sampler: per-slot CPU, memory and disk writes, orphan detection, threshold alerts, couch --resources

## Problem

Every resource incident so far was noticed by a human seeing a slow or full machine: a 22 GB orphan `nvim --embed`,
fseventsd at ~90 GB, #579's 61× write amplification (System Monitor), the scratch volume filling, the 15 s ledger
decode (#584). Nothing measures resource use per slot, so problems surface days late.

Part of the ops project resource-observability (`ops/teams/devinfra/projects/resource-observability/project.md`).
