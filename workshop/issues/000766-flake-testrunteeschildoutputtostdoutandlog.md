---
id: 000766
status: open
deps: []
github_issue:
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'f796659159b6bee66ff4bc7b37f876ab5e320d6e' # card fields mirrored from issue-cards; edit via sdlc
---

# flake: TestRunTeesChildOutputToStdoutAndLog

## Problem

Known flaky tests (#394), registered with same-head fail→pass evidence:

- github.com/xianxu/pair/cmd/internal/scribecmd.TestRunTeesChildOutputToStdoutAndLog: Same exact head 263cbba51998e2f853e2c1012f9c36c39858ad20: full recorded suite attempt 05d38e5d5280975a07e7e6ad51a2b0b4 failed only this test in 10m30s (stdout empty); all remaining packages completed. Failed log /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/05d38e5d5280975a07e7e6ad51a2b0b4.log. Subsequent exact targeted go test -run ^TestRunTeesChildOutputToStdoutAndLog$ -count=1 -v passed in 0.055s on unchanged head. A separate count=100 reproduced two failures in 2.591s, confirming intermittent missing PTY child output. No scribe source is changed by #761. Root cause remains to be established; this is registration of the observed same-head fail/pass, not a whole-suite passing claim.

## Spec

## Done when

- The root cause is fixed (never a retry loop or a longer timeout); closing this issue unregisters the tests.

## Plan

- [ ]

## Log

### 2026-10-21
