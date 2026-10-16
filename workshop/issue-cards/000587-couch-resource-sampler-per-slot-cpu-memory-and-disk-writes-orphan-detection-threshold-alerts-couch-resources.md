---
id: 000587
status: done
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
actual_hours: 4.46
tracker:
    version: 1
    completion:
        token: close-e6c417ceaf4a
        repository: github.com/xianxu/pair
        reviewed_head: d477e4a815847e3aaea8ad4450bb4d36df0fda2f
        evidence_commit: 82ec7d00d6df396e5d2a72bc841a48e80e236b4b
        landed_commit: 034b04ba489d908c8355724dac9e3ae9b2db3828
---

# Couch resource sampler: per-slot CPU, memory and disk writes, orphan detection, threshold alerts, couch --resources

## Problem

Every resource incident so far was noticed by a human seeing a slow or full machine: a 22 GB orphan `nvim --embed`,
fseventsd at ~90 GB, #579's 61× write amplification (System Monitor), the scratch volume filling, the 15 s ledger
decode (#584). Nothing measures resource use per slot, so problems surface days late.

Part of the ops project resource-observability (`ops/teams/devinfra/projects/resource-observability/project.md`).
