---
id: 000895
status: open
deps: []
github_issue:
created: 2026-10-25
updated: 2026-10-25
estimate_hours:
card_mirror: '0df2c3f3cd0a6a2d66bfd297538766dc651b2805' # card fields mirrored from issue-cards; edit via sdlc
---

# flake: TestOperatorSketchLiveDrawing

## Problem

Known flaky tests (#394), registered with same-head fail→pass evidence:

- github.com/xianxu/pair/cmd/internal/webcast.TestOperatorSketchLiveDrawing: pair#885 M1 live-Chrome proof, 2026-10-25, pair:10 (WEBCAST_LIVE_CHROME=1, load ~20-35): 1 failure in 10 at operator_sketch_live_test.go:225 "waiting for required label: context deadline exceeded" after 60s. The failing wait helper (file line 95) and every line before 643 are unchanged from main by #885, so the failure precedes any #885 edit; an earlier -count=10 run of the same test in the full live set passed 10/10. Cause unclassified (the label-required phase never appeared after the stroke). Not tuned here.

## Spec

## Done when

- The root cause is fixed (never a retry loop or a longer timeout); closing this issue unregisters the tests.

## Plan

- [ ]

## Log

### 2026-10-25
