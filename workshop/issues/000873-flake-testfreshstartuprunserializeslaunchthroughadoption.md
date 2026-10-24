---
id: 000873
status: open
deps: []
github_issue:
created: 2026-10-24
updated: 2026-10-24
estimate_hours:
card_mirror: '6bf1631406ff017137088825fffdda5dcb21d487' # card fields mirrored from issue-cards; edit via sdlc
---

# flake: TestFreshStartupRunSerializesLaunchThroughAdoption

## Problem

Known flaky tests (#394), registered with same-head fail→pass evidence:

- github.com/xianxu/pair/cmd/internal/couchcmd.TestFreshStartupRunSerializesLaunchThroughAdoption: Failed in sdlc merge suites on 2026-10-24: PR 590 head 8747012d (log .git/sdlc/test-stats/failed/da7653e44c27a012b3893c4378e7053c.log) and the main baseline 2b9334fd (log .git/sdlc/test-stats/failed/668931af27a5603e851eae1f4d974e35.log). Both: fresh_startup_run_test.go:199: startup failed: lock thread store: resource temporarily unavailable; no other test failed. sdlc labelled it a persistent main failure. Registered on TL ops:0 instruction so pair merges can proceed; TL gives the root cause to pair:7 (author of the #863 fresh-startup test).

## Spec

## Done when

- The root cause is fixed (never a retry loop or a longer timeout); closing this issue unregisters the tests.

## Plan

- [ ]

## Log

### 2026-10-24
