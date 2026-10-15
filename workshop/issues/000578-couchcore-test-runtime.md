---
id: 000578
status: open
deps: [ariadne#392]
github_issue:
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
card_mirror: 'c859a288d2b79e94554ddddec87e54113dbaa00a' # card fields mirrored from issue-cards; edit via sdlc
---

# Profile and reduce couchcore test runtime

## Problem

The 2026-10-15 `sdlc test-stats --since 7d` sample shows `cmd/internal/couchcore` at 481.3 seconds p90 over 21 runs, a major component of pair's 556/786-second suite p50/p90. #390's operator review deferred numeric budgets until weekly trends exist, but a cause-level performance investigation can begin from the measured bottleneck.

## Spec

Profile couchcore test runtime using comparable full-suite records and controlled targeted runs, separating test execution, fixture startup, external-process waits, cache state and machine contention. Identify the small set of slow tests or shared setup costs. Propose and implement a change only when it preserves assertions and deterministic fault detection; avoid longer timeouts, blind parallelism or deleting tests based on coverage overlap alone. Report before/after package p50/p90 and suite wall with sample counts, plus the exact tests changed. Do not install the withdrawn 600-second ceiling or 240-second target. Follow ariadne#390's approved track-first design and ariadne#392's weekly measurement.

## Done when

- [ ] A reproducible profile identifies which couchcore tests or fixtures account for the observed package time, with host/cache/load conditions and sample counts.
- [ ] Any optimization has meaningful tests for behavioral equivalence and a measured before/after package and suite wall comparison; if no safe change exists, the report records the bottleneck and next experiment.
- [ ] The result feeds the ops test-economy weekly snapshot; no package budget or timeout is changed without a later operator decision.

## Plan

- [ ] Collect and profile representative couchcore runs after the first weekly snapshot.
- [ ] Test and ship a cause-level improvement, or document why the measured bottleneck remains.

## Log

### 2026-10-15

Filed from ariadne#390's approved test-economy plan. The 481.3-second number is a measurement, not a new limit.
