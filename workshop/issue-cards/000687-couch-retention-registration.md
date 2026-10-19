---
id: 000687
status: working
created: 2026-10-18
updated: 2026-10-18
estimate_hours: 2.015
github_issue:
started: 2026-10-18T21:13:11-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
---

# Recognize Couch supervisor retention registration

## Problem

During #677's real diagnostic preview, the running Couch supervisor PID45023 was classified as an ownerless legacy runtime and blocked253 current diagnostic rows containing23,462,164,824bytes. Those live current generations were correctly retained; no process intervention was performed. Closed unheld old generations now have separate reader-only proof.

Couch already calls storagegc.Coordinator.RegisterRuntime in couchcore/couch.go, but diagnosticlog.OSInspection/VerifyWriters recognizes PAIR_RETENTION_PROTOCOL=1 or the particular log's writer registrations. The supervisor has no per-thread PAIR_DATA_DIR/tag and its runtime lease is not consumed by that proof, so being registered in the coordinator does not establish protocol awareness there. Missing protocol means unrecognized registration, not an old start time.
