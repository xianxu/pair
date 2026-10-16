---
id: 000611
status: working
deps: []
github_issue:
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
card_mirror: 'f9e956c445e25fbd37fe378e295c66ccf6f6bbce' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-16T06:20:09-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
---

# Bring couchcore under a measured test execution budget

## Problem

Couchcore consumes nearly Go's default ten-minute per-package timeout: #591's integrated run abe0a5ed883f06ffe58305cb45a29c09 timed out at 600.245 s, while its automatic main comparison 2064bde22de3d40f5d15091fe41ef443 passed with couchcore at 580.585 s. #589 is affected too. #610 temporarily gives the broad suite an explicit 20m timeout to unblock landings; it does not solve the accumulated execution cost. This is part of ops/test-economy, whose project is maintained by the hub.

## Spec

Profile couchcore's slowest tests using recorded per-test execution data under both isolated and suite-equivalent load. Distinguish real IO/setup cost, serialization and scheduling waits. Choose a measured package split or deterministic sharding strategy that preserves test inventory, fixture ownership, assertions and cancellation. Set and defend a practical execution budget with headroom, and remove #610's temporary 20m allowance when the measured suite is reliably within that budget. No retries, skips or longer sleeps as a performance fix.

## Done when

- [ ] Recorded profiles identify the slowest tests and quantify execution separately from suite queue wait, under idle and representative load.
- [ ] A justified package split or sharding design preserves every test and its isolation contract, with deterministic coverage and bounded concurrency.
- [ ] Before/after full-suite measurements demonstrate the agreed execution budget with headroom; fixture deadlines and assertions remain intact.
- [ ] The temporary 20m allowance from #610 is removed or replaced by the measured, documented steady-state budget.

## Plan

- [x] Measure an isolated couchcore run with Go JSON events; distinguish execution, paused parallel tests, compilation/setup, and machine co-tenancy.
- [ ] Compare suite-equivalent timings, audit fixture/package boundaries, and send a measured full-flow design to the hub before any split.
- [ ] After design approval, implement and qualify the agreed test organization, inventory preservation and execution budget; remove the temporary allowance; full-flow review and landing.

## Log

### 2026-10-16 — deferred performance work

Filed unassigned at the hub's request (decision 173f498b-9796-4fd6-a777-07e28ce4107c). #610 is the minimal unblock and lands first, followed by unchanged #591 and #589 retries. The timeout failure alone does not identify a particular slow test; this follow-up owns profiling before choosing the package/shard boundary.

2026-10-16: Dispatched by ops:0 (verified receipt 34a5f4e7-0d39-4ada-9556-d23a2e357561). Claimed and entered planning from fresh main after #612 landed. No test-structure changes authorized before measured profile and hub plan review. A read-only independent audit will map test/helper boundaries while the main slot records an isolated JSON timing run. ARCH-CONSTRAINTS: preserve the 10-minute default with measured headroom and bounded concurrency; profile before choosing mechanism.

2026-10-16: Static audit: 270 same-package test files, 1,369 source top-level tests plus 8 fuzz functions, 317 free helpers; newTestEnv has 208 references across 68 files and many tests use package-private state. TestMain dispatches self-exec aliases/helpers before normal tests and defaults parallelism to 3×GOMAXPROCS unless explicitly set. Existing upstream sharder is specific to ariadne cmd/sdlc, so evaluate reusing ordinary Go discovery/JSON/run semantics rather than duplicating its compile/conversion stack. Raw isolated profile and host metadata: /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair-611-profile.mIMPri/. The package run has no competing Go suite at its initial process snapshot, but OS load averages were 16.41/21.90/23.80 on 12 logical CPUs; label this package-isolated, not an idle-machine guarantee. First parallel continuation is 143.744s after package start, proving a substantial sequential execution phase before parallel-queue release. No split chosen or code changed.

2026-10-16: Isolated JSON profile PASS: 326.006s package, 328.25s command wall; 1,376 top-level cases (1,359 pass, 17 existing skips). Sequential phase 143.743848s; remaining parallel phase 182.262s. Critical environment correction: inherited GOMAXPROCS=3 and GOFLAGS=-p=3 make TestMain's actual parallel default nine, not 36. Source CPU-count inference alone would have overprovisioned the shards. Slowest test 46.26s; no individual near the package timeout. Durable profile: workshop/plans/000611-couchcore-test-execution-budget-profile.md; raw JSON SHA-256 e45f670a3506a5687ad440a12e4d278c1921a9f2fbe11bbf1b12d3bb1704827a.

Full-flow design draft: workshop/plans/000611-couchcore-test-execution-budget-plan.md. Recommend three ordinary Go-command process shards, sorted round-robin from Go-discovered names, aggregate parallel budget at most nine and honoring smaller caller CPU limits. Predict worst lane 262.78s; proposed qualification <300s isolated and <420s representative load, then remove #610's allowance. Preserve package-private fixtures, TestMain and self-exec helpers. Counter-check every discovered Test/Fuzz/Example exactly once, including existing skip dispositions; every other package executes once. Refuse explicit native/live opt-ins in the default parallel runner and retain their existing serial routes; this boundary is explicit for hub review. No implementation or split yet. Real Make suite baseline with only the final Go command instrumented as JSON is running, holding the existing machine lock. Finish it and reconcile measured loaded distribution before implementation.

## Revisions

2026-10-16 — Full-flow design proposal: use process selection of the intact package rather than a directory split or a fork of the specialized ariadne sharder. Concurrency is based on measured caller GOMAXPROCS=3, not host logical CPU count. Proposed thresholds and live-route handling require hub design review before change-code.

2026-10-16 — R2 direction (receipt f85a0b66-f9d0-47d3-b5f0-51c3a2f54288): hub rejects sharding-first. Make the slowest tests cheaper, remeasure, and propose lanes only if isolated time still exceeds about200s. The previous runner proposal is unapproved history, not an implementation contract. Revised full-flow plan and per-test cause table are appended to the canonical plan/profile. No source implementation changes made.

Diagnostic causes: three domain properties each generate731,566,080 raw cells for2,406,528 admitted points; CPU profile attributes98.35% cumulative samples to iterator work. HostJudgments then discards most dependency coordinates before re-expanding them. Git traces count1,096 calls in12-scenario real reconciliation and931 in recovery convergence; fake SDLC identity uses four real queries per observation, allowing fresh batched evidence without caching mutable state. Bounded continuation test performs100 publications for ten requested rows. Lifecycle shape test allocates stores before rejecting invalid shapes and writes metadata twice. Default producer conformance walks the operator fleet/remotes rather than a bounded test fleet. The loaded-tail SetAside hold-policy test creates nine fully provisioned Git worlds although it chiefly needs real directories/locks and one minimal healthy checkout. Preserve all domain/case assertions and real conformance where it is the claim. No savings claim until repeat measurements.

2026-10-16: Instrumented full-Make before baseline PASS,900.89s wall; couchcore539.310s (serial321.993447s, parallel217.317s), same1,376 top-level dispositions. Go package scheduling/build lag15.799428s separately; no machine-lock wait reported. Targeted CPU/Git-trace diagnostics overlapped its Go phase, explicitly recorded as loaded co-tenancy. Raw full log SHA-2569aa661a8edbdca473d6d326b8552c8a20667fc86c7706359ce2782489a254271. Profile artifact contains the completed baseline. R2 cause-based plan delivered to hub (40482e40-e68a-4f06-8b98-59f16f44943b submitted); waiting for full-flow design response before change-code.
