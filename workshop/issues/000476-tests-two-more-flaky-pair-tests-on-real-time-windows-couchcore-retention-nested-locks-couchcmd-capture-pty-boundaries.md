---
id: 000476
status: working
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: 'b5106c2590966ec149ca96045aecd87e3a10f647' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T03:57:00-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "c15c62bb", done: "c9efc146"}
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
