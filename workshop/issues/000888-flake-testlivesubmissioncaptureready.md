---
id: 000888
status: open
deps: []
github_issue:
created: 2026-10-25
updated: 2026-10-25
estimate_hours:
card_mirror: 'd9c07d1a021678c77f5881365974f2ba98e61e04' # card fields mirrored from issue-cards; edit via sdlc
---

# flake: TestLiveSubmissionCaptureReady

## Problem

Known flaky tests (#394), registered with same-head fail→pass evidence:

- github.com/xianxu/pair/cmd/internal/webcast.TestLiveSubmissionCaptureReady: pair#885 M1 live-Chrome proof, 2026-10-25, pair:10 at 42f17374d^ code (webcast sweep in progress; live_capture_ready_test.go unchanged by it): WEBCAST_LIVE_CHROME=1 go test -count=1 -run (live set) ./cmd/internal/webcast/ under load average ~35 failed at live_capture_ready_test.go:205 "animation timing before start=58.136000 current=199.990000; after start=58.136000 current=180.000000" and :207 "screenshot restarted animation or changed viewport" (Same:true, State:finished, 864x997 DPR 2). The same test passed in the following baseline run on unmodified HEAD and in later runs. Cause unclassified; the assertion compares Web Animations currentTime across a screenshot, which a loaded compositor can perturb. Not tuned here.

## Spec

## Done when

- The root cause is fixed (never a retry loop or a longer timeout); closing this issue unregisters the tests.

## Plan

- [ ]

## Log

### 2026-10-25
