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

- [ ] Measure an isolated couchcore run with Go JSON events; distinguish execution, paused parallel tests, compilation/setup, and machine co-tenancy.
- [ ] Compare suite-equivalent timings, audit fixture/package boundaries, and send a measured full-flow design to the hub before any split.
- [ ] After design approval, implement and qualify the agreed test organization, inventory preservation and execution budget; remove the temporary allowance; full-flow review and landing.

## Log

### 2026-10-16 — deferred performance work

Filed unassigned at the hub's request (decision 173f498b-9796-4fd6-a777-07e28ce4107c). #610 is the minimal unblock and lands first, followed by unchanged #591 and #589 retries. The timeout failure alone does not identify a particular slow test; this follow-up owns profiling before choosing the package/shard boundary.

2026-10-16: Dispatched by ops:0 (verified receipt 34a5f4e7-0d39-4ada-9556-d23a2e357561). Claimed and entered planning from fresh main after #612 landed. No test-structure changes authorized before measured profile and hub plan review. A read-only independent audit will map test/helper boundaries while the main slot records an isolated JSON timing run. ARCH-CONSTRAINTS: preserve the 10-minute default with measured headroom and bounded concurrency; profile before choosing mechanism.

2026-10-16: Static audit: 270 same-package test files, 1,369 source top-level tests plus 8 fuzz functions, 317 free helpers; newTestEnv has 208 references across 68 files and many tests use package-private state. TestMain dispatches self-exec aliases/helpers before normal tests and defaults parallelism to 3×GOMAXPROCS unless explicitly set. Existing upstream sharder is specific to ariadne cmd/sdlc, so evaluate reusing ordinary Go discovery/JSON/run semantics rather than duplicating its compile/conversion stack. Raw isolated profile and host metadata: /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair-611-profile.mIMPri/. The package run has no competing Go suite at its initial process snapshot, but OS load averages were 16.41/21.90/23.80 on 12 logical CPUs; label this package-isolated, not an idle-machine guarantee. First parallel continuation is 143.744s after package start, proving a substantial sequential execution phase before parallel-queue release. No split chosen or code changed.
