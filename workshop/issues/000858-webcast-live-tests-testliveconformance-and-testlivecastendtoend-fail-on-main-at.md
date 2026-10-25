---
id: 000858
status: working
deps: []
github_issue:
created: 2026-10-23
updated: 2026-10-25
estimate_hours:
card_mirror: '1f19959151ae7e950d0dfefc5f6b934e7c8f23ff' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-25T00:11:45-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "56328961", done: "dc4bbbee"}
---

# Webcast live tests TestLiveConformance and TestLiveCastEndToEnd fail on main at the in-page pause click

## Problem

## Spec

The native fixture clicks the actual in-page Live/Pause button using real CDP mouse input. It must resolve the control inside Couch's closed shadow root rather than assuming the outer UI host's center is the button. Production controls, authority and pause semantics stay unchanged. Headless Chrome uses a private profile and never raises a foreground window. Retain existing readiness and state deadlines.


**Two webcast live tests fail on clean main: TestLiveConformance and TestLiveCastEndToEnd (found by pair:6 while
working pair#849, 10-23).** Both fail at the same step, clicking the in-page pause control, identically on clean
`origin/main`, so they're not caused by any branch. Find whether the pause control moved or changed (pair#850's panel
and controls, or another recent cast change), or the tests' selector is stale; fix the cause, not the test's
expectation, unless the control legitimately moved.

## Done when

- Root cause is recorded against clean main; the reported native pause failure is reproduced before the fix.
- The fixture locates the actual Live/Pause button and checks its usable geometry before real mouse input; a missing/hidden control refuses rather than clicking a guessed host point.
- TestLiveConformance passes pause/resume and TestLiveCastEndToEnd observes paused viewer state. Audit and qualify every shared clickControl consumer, one targeted native run at a time, with no focus-taking Chrome.
- Required recorded pre-close and mandatory close pass; no production behavior or timeout changes.


- Both tests pass on main, with the cause stated in the Log.

## Plan

- [ ] Reproduce on main and trace the host/button geometry ownership.
- [ ] Replace the stale fixture hit target with the actual closed-shadow control; keep real input and downstream state assertions.
- [ ] Run the named native regressions and affected fixture consumers; inventory, recorded pre-close, close and merge.


- [ ]

## Log

### 2026-10-25 — Cause-first diagnosis

Started from clean main63d35ac8295d6b4ff04001c683b038eca1d0e3d7. WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/webcast -run '^TestLiveConformance$' -count=1 -v -timeout=90s fails17.15s at the page pause assertion: Phase remains live. Launch uses Headless:true and a private t.TempDir profile. The shared helper queries only outer data-couchcast-ui geometry; #856's mountCastTools adopts liveButton into a fixed-position column, invalidating that hit-target assumption. The closed shadow's actual cast-button remains the production click/binding owner. Fix that fixture ownership boundary (ARCH-DRY, ARCH-PURPOSE, ARCH-MOCK), not the pause behavior. No new state, wire, timer or persistent resource. Runtime cost is fixture-only bounded debugger queries on an explicit action.

#857 remains on its own pushed branch, codecomplete after documentation-only FIX-THEN-SHIP correction; native revised-source suite passed, required integrated check terminal2123/lease-904989313 is retained on its independent snapshot. It has no merge runner and may safely finish while this canonical checkout handles #858. Pair:0 stays unchanged; #857's operator re-look is tomorrow.


### 2026-10-23
