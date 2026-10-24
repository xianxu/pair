---
id: 000868
status: open
deps: []
github_issue:
created: 2026-10-24
updated: 2026-10-24
estimate_hours:
card_mirror: '52121ba435bbda9c4b7b30e78432d4ca5b39cac7' # card fields mirrored from issue-cards; edit via sdlc
---

# flake: TestCancellationRetirementAllowsIDReuseWithoutInheritedRevocation

## Problem

Known flaky tests (#394), registered with same-head fail→pass evidence:

- github.com/xianxu/pair/cmd/internal/testcouch/agentfixture.TestCancellationRetirementAllowsIDReuseWithoutInheritedRevocation: failed in attempt de7d151179704cbaed6e949992bc8d8b, passed on the same head in attempt 3fabfaabea51c1ee78825a876be040a6

## Spec

## Done when

- The root cause is fixed (never a retry loop or a longer timeout); closing this issue unregisters the tests.

## Plan

- [ ]

## Log

### 2026-10-24
