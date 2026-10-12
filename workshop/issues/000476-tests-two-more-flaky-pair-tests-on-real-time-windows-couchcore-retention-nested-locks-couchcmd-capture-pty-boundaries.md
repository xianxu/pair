---
id: 000476
status: open
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: 'd09385318677eeb0e41a95532e13546efbffafed' # card fields mirrored from issue-cards; edit via sdlc
---

# tests: two more flaky pair tests on real-time windows (couchcore retention nested locks; couchcmd capture PTY boundaries)

## Problem

While fixing pair#473 (PR #265), pair:5 saw two other tests fail once each under load, untouched by that change: `couchcore` `TestRetentionNestedStoreLocksYieldAndReleaseRoot/forget` (a 150 ms real-time window, retention_test.go:604) and `couchcmd` `TestCapturePTYBothBoundaries/regular` (a drain timeout, capture_test.go:238). Same family as #473: assertions on real-time windows that a loaded machine misses.

## Spec

As in #473: reproduce by widening the window or loading the machine, fix the root cause (an injectable clock, waiting on the event, deterministic ordering), never a longer timeout or a retry; fix product code if the race is there. Then sweep the suite for the same pattern (fixed real-time windows asserted under load) and list any others in the Log, fixing the cheap ones and filing the rest.

## Done when

- Both tests pass 200 runs under -race and load, and in repeated full-suite runs; causes in the Log.
- The sweep's findings are recorded (fixed or filed).

## Plan

- [ ]

## Log

### 2026-10-12
