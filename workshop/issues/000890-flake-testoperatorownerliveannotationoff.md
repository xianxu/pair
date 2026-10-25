---
id: 000890
status: open
deps: []
github_issue:
created: 2026-10-25
updated: 2026-10-25
estimate_hours:
card_mirror: '5d212431405e21ff0b136b0347f13b2ac9218f90' # card fields mirrored from issue-cards; edit via sdlc
---

# flake: TestOperatorOwnerLiveAnnotationOff

## Problem

Known flaky tests (#394), registered with same-head fail→pass evidence:

- github.com/xianxu/pair/cmd/internal/webcast.TestOperatorOwnerLiveAnnotationOff: pair#885 M1 live-Chrome proof, 2026-10-25, pair:10 at e769340d4 (WEBCAST_LIVE_CHROME=1, load average 23-35): 2 failures in 20 runs, both operator_owner_live_test.go:84 "waiting for Note selected after expanded Pen: context deadline exceeded" after 60s (the test ctx), i.e. the click on .sketch-label button never left Note aria-pressed=true. Unmodified main (4ae4b15e1) passed 10/10 in the same window. The #885 change to this test only moved its wait from a 10s hand-rolled deadline to testwait.WaitEvery at the same 25ms cadence; the condition, clicks and ctx are unchanged, so the event was lost rather than slow. Cause unclassified (candidate: the label submit click races the toolbar re-layout after resize(400)). Not tuned here.

## Spec

## Done when

- The root cause is fixed (never a retry loop or a longer timeout); closing this issue unregisters the tests.

## Plan

- [ ]

## Log

### 2026-10-25
