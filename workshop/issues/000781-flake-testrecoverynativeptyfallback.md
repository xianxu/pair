---
id: 000781
status: open
deps: []
github_issue:
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
collaboration-mode: collaboration
card_mirror: '415b4457eee9d10b58cf5ec7869974a34ce6bf28' # card fields mirrored from issue-cards; edit via sdlc
---

# flake: TestRecoveryNativePTYFallback

## Problem

Known flaky tests (#394), registered with same-head fail→pass evidence:

- github.com/xianxu/pair/cmd/internal/termcmd.TestRecoveryNativePTYFallback: Same unchanged head 0acb74917: recorded suite attempt a8cba83c6566c428b6b70a65c9f2e05e failed only TestRecoveryNativePTYFallback/termination at recovery_test.go:324 with empty PTY output; parent attribution is coarsened by the suite parser. Saved log /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/a8cba83c6566c428b6b70a65c9f2e05e.log, sha256 prefix a5427c3a84c1. On the same head without code changes, go test ./cmd/internal/termcmd -run ^TestRecoveryNativePTYFallback$ -count=1 -v passed both shell and termination, package 0.106s; complete go test ./cmd/internal/termcmd -count=1 passed 3.038s. Fixture directly uses creack/pty and checks a short-lived child final token after Wait; Darwin final-slave-close loss is a causal lead, not proven. Separate from production ptychild issue #770. No full-suite passing retry claimed.

## Spec

## Done when

- The root cause is fixed (never a retry loop or a longer timeout); closing this issue unregisters the tests.

## Plan

- [ ]

## Log

### 2026-10-21
