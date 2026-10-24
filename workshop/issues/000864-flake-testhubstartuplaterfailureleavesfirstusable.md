---
id: 000864
status: open
deps: []
github_issue:
created: 2026-10-24
updated: 2026-10-24
estimate_hours:
card_mirror: 'a0f07de7d3086620f5ff80241ccb4e057ea3c18c' # card fields mirrored from issue-cards; edit via sdlc
---

# flake: TestHubStartupLaterFailureLeavesFirstUsable

## Problem

Known flaky tests (#394), registered with same-head fail→pass evidence:

- github.com/xianxu/pair/cmd/internal/couchcmd.TestHubStartupLaterFailureLeavesFirstUsable: failed in attempt 4fed2c456da3773658b2bd4328454d6a, passed on the same head in attempt 5bafd7c76410f108c1e6ee72e3e78952

## Spec

## Done when

- The root cause is fixed (never a retry loop or a longer timeout); closing this issue unregisters the tests.

## Plan

- [ ]

## Log

### 2026-10-24
