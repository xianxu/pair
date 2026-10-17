---
id: 000626
status: open
deps: []
github_issue:
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
card_mirror: 'c1c0288e676b7dd8fdbe55bed96c6cf2c33adba9' # card fields mirrored from issue-cards; edit via sdlc
---

# Workbench activity invariant intermittently times out waiting for cold worker

## Problem

During #622 M1 affected-package verification at head aaf17569ea4005aa59502de3a3e47ecacb01dada, `go test ./cmd/internal/couchcmd ./cmd/internal/couchtty ./cmd/internal/ptychild ./cmd/internal/couchcore ./cmd/internal/artifactpath` failed `TestWorkbenchPerformanceIdleUITick` at workbench_activity_invariant_test.go:77: `activity worker did not finish`. Test duration 50.43s; couchtty package 101.425s. Couchcmd passed 132.400s and ptychild passed 1.078s; the command continued through the remaining packages.

The unchanged targeted repeat `go test ./cmd/internal/couchtty -run '^TestWorkbenchPerformanceIdleUITick$' -count=1 -v` passed the read/decode accounting with test duration 30.16s. Retained repeat log: /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair622-activity-investigation.bQRoPX/current.log. Host load during investigation was 15.35/20.35/32.69.

This fixture creates no Child and never requests resources, so #622's new ingest/snapshot paths do not execute. Its activity/runtime/fixture source is unchanged against pre-counter head 242bd40216171712db036069ca5aae4f0c381366. The worker sequentially processes twenty 30MB ledgers, with per-row 2s probe deadlines; the test independently uses a fixed 30s completion guard. Contention is plausible but the exact timeout cause is unproven.

## Spec

Diagnose worker progress and deadline attribution in this fixture under normal package/suite load. Preserve its cold-read and unchanged-tick invariants; do not solve the report by merely increasing the timeout or adding retries. Keep failure evidence and distinguish lost work from expensive cold reads and scheduling delay.

## Done when

- Root cause is demonstrated with worker/probe progress evidence.
- A regression guards that cause while preserving the twenty-owner ledger accounting and unchanged UI-tick zero-read assertions.
- Package and recorded full-suite verification pass without blind retries or a larger arbitrary deadline.

## Plan

- [ ]

## Log

2026-10-16: Filed unassigned from #622 after one package-run failure and one unchanged targeted pass. No related production or fixture change is included in #622; its recorded full suite remains queued.
