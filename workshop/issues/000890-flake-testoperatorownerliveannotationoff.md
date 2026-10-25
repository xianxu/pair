---
id: 000890
status: working
deps: []
github_issue:
created: 2026-10-25
updated: 2026-10-25
estimate_hours:
card_mirror: '63162ac6e5e3b68335da8e176035604b1d838e1d' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-25T10:31:42-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:9
    worktree: /Users/xianxu/workspace/worktree/pair-slot9/pair
    repository: github.com/xianxu/pair
flake_tests: [github.com/xianxu/pair/cmd/internal/webcast.TestOperatorOwnerLiveAnnotationOff]
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

Claimed from main after884 landed. Dispatch4f39b992 asks890, then895, then888 one at a time, private headless Chrome only, targeted count<=10, no retry/timeout increase. Root investigation: owner test computes a hit-tested center in Runtime.evaluate, then dispatches separate CDP mousePressed/mouseReleased; final Save-label is inside a feedback panel after short-viewport resize and tool exit. Sketch895 similarly computes Done geometry before native input. Shared cause is unproven; inspect rendered/presented layout, event targets, and model completion before proposing a fix. Existing #858 live-control helper guards hidden/disabled geometry but is specific to the live toggle. No code changes yet. Baseline head11fece5cfb3b7da9ab93e6e592ec4f940919887a.
