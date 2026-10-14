---
id: 000545
status: working
deps: []
github_issue:
created: 2026-10-13
updated: 2026-10-13
estimate_hours:
card_mirror: '1bbac479d83e4132a03f67d989ec7dd9cc66e2b5' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-13T23:20:11-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
---

# couchcore: eliminate spurious workspace creation lock contention

## Problem

#543's complete make -k test run failed six couchcore tests with
`workspace host creation is busy` (one observed loaded suite, six failures):

- TestManagedCreateReuseRefusesProfileDriftDuringSetup
- TestReconcileResultListsWhatItSetAside
- TestSavedWorkIsCollectedPastRetention
- TestSetAsideHoldsMatchThePlan
- TestSetAsideMovesTheWholeCheckoutAndCanBeRestored
- TestSetAsideHostRecordsItsBranch

All six passed unchanged in one isolated rerun (26.819s). Logs:
/tmp/pair543-full-suite.log and /tmp/pair543-core-rerun.log. This is an initial
sample, not a general failure-rate estimate. TL also reports two failures during
#528; the peer has been asked for names/counts. Establish repeated loaded
baseline and fixed counts, following ariadne#360/#361's interleaved method.

## Spec

Identify the actual owner/lifetime of the contended lock. Correct the cause
without increasing a timeout or weakening exclusion while a real workspace
producer remains alive. Distinguish shared test paths from incidental fork
inheritance; use deterministic regression evidence and repeated runs alongside
a complete suite. Preserve parent-death and intentionally inherited lease safety.

## Done when

- Root cause is demonstrated, with the lock path and owner/lifetime identified.
- A deterministic regression fails before the fix and passes after it; genuine
  competing producers still exclude a second creation, including parent death.
- Repeated before/after runs alongside suite load record exact failure counts
  and denominators, at least 50 repetitions of the affected cases per variant.
- Full make -k test completes, failures are explained, close returns SHIP and
  the PR lands through sdlc with validation.

## Plan

- [ ] Investigate and design the smallest ownership fix; approval before code.
- [ ] Implement with regression and loaded before/after evidence.
- [ ] Complete suite, documentation, review and landing.

## Log

### 2026-10-13

- Filed and claimed via TL receipt 0a70adcf-201a-45b0-9c71-fb6704a2b9c6.
  start-plan refreshed the slot's Ariadne and Pair baselines; rerun started the
  branch. Existing fixture roots are unique temp directories, despite shared
  repo-name basenames. Leading hypothesis is incidental fork inheritance of the
  descriptor flock: Close retains lock while any fork copy exists, and the
  existing 100ms settle is not a lifetime proof. Intentional inheritance must stay.
