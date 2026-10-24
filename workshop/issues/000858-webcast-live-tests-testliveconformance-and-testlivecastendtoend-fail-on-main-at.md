---
id: 000858
status: open
deps: []
github_issue:
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
card_mirror: '34b5076267505701885f7491dfa27fe3fbf0aa6d' # card fields mirrored from issue-cards; edit via sdlc
---

# Webcast live tests TestLiveConformance and TestLiveCastEndToEnd fail on main at the in-page pause click

## Problem

## Spec

**Two webcast live tests fail on clean main: TestLiveConformance and TestLiveCastEndToEnd (found by pair:6 while
working pair#849, 10-23).** Both fail at the same step, clicking the in-page pause control, identically on clean
`origin/main`, so they're not caused by any branch. Find whether the pause control moved or changed (pair#850's panel
and controls, or another recent cast change), or the tests' selector is stale; fix the cause, not the test's
expectation, unless the control legitimately moved.

## Done when

- Both tests pass on main, with the cause stated in the Log.

## Plan

- [ ]

## Log

### 2026-10-23
