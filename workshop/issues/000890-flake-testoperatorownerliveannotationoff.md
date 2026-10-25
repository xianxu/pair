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

The owner live fixture must observe completion of a requested viewport resize before sampling native click geometry. Emulation metrics change innerHeight before Chrome dispatches resize; the old toolbar can already fit, so its bounds do not prove floating Notes consumed resize. Arm a viewport-matching resize receipt before metrics, await it after existing synchronous UI listeners, and dispose its listener/object on success or error. Same geometry is an immediate no-op. Retain native one-shot press/release, real permissions and existing timeout. ARCH-ORDER separates changed viewport from consumed resize; ARCH-DRY owns the CDP resize sequence in one test helper. No product/API/persisted-state change.

## Done when

- A controlled private-Chrome regression holds resize delivery after metrics change and rejects a helper that reports completion early; releasing the event completes the barrier. Same-geometry and error cleanup are covered.
- The owner scenario passes bounded native runs with the existing timeout, one click per action; no retry or arbitrary sleep.
- The recorded affected check and normal landing suite pass; close unregisters890.

## Plan

- [ ] Pin consumed-resize ordering with a held native event and establish RED.
- [ ] Await the armed resize receipt in the owner fixture; verify native and lifecycle controls.
- [ ] Record evidence, lifetime lesson, inventory and final verification.

## Log

### 2026-10-25

Claimed from main after884 landed. Dispatch4f39b992 asks890, then895, then888 one at a time, private headless Chrome only, targeted count<=10, no retry/timeout increase. Root investigation: owner test computes a hit-tested center in Runtime.evaluate, then dispatches separate CDP mousePressed/mouseReleased; final Save-label is inside a feedback panel after short-viewport resize and tool exit. Sketch895 similarly computes Done geometry before native input. Shared cause is unproven; inspect rendered/presented layout, event targets, and model completion before proposing a fix. Existing #858 live-control helper guards hidden/disabled geometry but is specific to the live toggle. No code changes yet. Baseline head11fece5cfb3b7da9ab93e6e592ec4f940919887a.

Root captured with diagnostics and browser-only CPU throttle4:9/10 pass,1 failure46.273s total. Native Save coordinates260.09375 hit feedback-body after button moved248->206; model still label-required, generation0 allowed, host visible. [Capture](../evidence/000890/lost-click.md). Initial trace-only10/10 PASS34.626s. No passing retry used as fix. Hub plan c0cec94d posted before code; quick plan proceeding unless objected. Separate895/888 remain unclaimed until890 lands or shared-cause proof justifies a fold.
