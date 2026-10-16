---
id: 000611
status: open
deps: []
github_issue:
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
card_mirror: '0a5ec0d01a9c5ac838d44cb524e7ad967b2b01bc' # card fields mirrored from issue-cards; edit via sdlc
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

- [ ]

## Log

### 2026-10-16 — deferred performance work

Filed unassigned at the hub's request (decision 173f498b-9796-4fd6-a777-07e28ce4107c). #610 is the minimal unblock and lands first, followed by unchanged #591 and #589 retries. The timeout failure alone does not identify a particular slow test; this follow-up owns profiling before choosing the package/shard boundary.
