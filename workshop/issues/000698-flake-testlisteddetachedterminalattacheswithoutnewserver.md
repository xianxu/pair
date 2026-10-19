---
id: 000698
status: open
deps: []
github_issue:
created: 2026-10-19
updated: 2026-10-19
estimate_hours:
collaboration-mode: collaboration
card_mirror: '312b933ed014aa180e60889e6f859a4f960584e7' # card fields mirrored from issue-cards; edit via sdlc
---

# flake: TestListedDetachedTerminalAttachesWithoutNewServer

## Problem

Known flaky tests (#394), registered with same-head fail→pass evidence:

- github.com/xianxu/pair/cmd/internal/couchcore.TestListedDetachedTerminalAttachesWithoutNewServer: failed in attempt adbcdbbf22e41d689a246f916f9dd2d2, passed on the same head in attempt b8cfedebf5403d1c880f0aa2a5e89ccd

## Spec

## Done when

- The root cause is fixed (never a retry loop or a longer timeout); closing this issue unregisters the tests.

## Plan

- [ ]

## Log

### 2026-10-19
