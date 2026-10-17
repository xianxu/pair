---
id: 000626
status: codecomplete
created: 2026-10-16
updated: 2026-10-17
estimate_hours:
github_issue:
started: 2026-10-17T08:52:29-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
actual_hours: 0.32
tracker:
    version: 1
    completion:
        token: close-2facb1e5b7a7
        repository: github.com/xianxu/pair
        reviewed_head: 64bae95b018325198f47183bfecfd45ffd4d917d
        evidence_commit: 769c78c9244a3b17aa97b66a15cc854f561f4766
---

# Workbench activity invariant intermittently times out waiting for cold worker

## Problem

During #622 M1 affected-package verification at head aaf17569ea4005aa59502de3a3e47ecacb01dada, `go test ./cmd/internal/couchcmd ./cmd/internal/couchtty ./cmd/internal/ptychild ./cmd/internal/couchcore ./cmd/internal/artifactpath` failed `TestWorkbenchPerformanceIdleUITick` at workbench_activity_invariant_test.go:77: `activity worker did not finish`. Test duration 50.43s; couchtty package 101.425s. Couchcmd passed 132.400s and ptychild passed 1.078s; the command continued through the remaining packages.

The unchanged targeted repeat `go test ./cmd/internal/couchtty -run '^TestWorkbenchPerformanceIdleUITick$' -count=1 -v` passed the read/decode accounting with test duration 30.16s. Retained repeat log: /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair622-activity-investigation.bQRoPX/current.log. Host load during investigation was 15.35/20.35/32.69.

This fixture creates no Child and never requests resources, so #622's new ingest/snapshot paths do not execute. Its activity/runtime/fixture source is unchanged against pre-counter head 242bd40216171712db036069ca5aae4f0c381366. The worker sequentially processes twenty 30MB ledgers, with per-row 2s probe deadlines; the test independently uses a fixed 30s completion guard. Contention is plausible but the exact timeout cause is unproven.
