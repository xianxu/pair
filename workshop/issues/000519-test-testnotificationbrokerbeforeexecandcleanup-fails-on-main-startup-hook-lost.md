---
id: 000519
status: open
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: 'bc8d192968ba7d50d4923fd1f021df27d48faa3a' # card fields mirrored from issue-cards; edit via sdlc
---

# test: TestNotificationBrokerBeforeExecAndCleanup fails on main ('startup hook lost')

## Problem

`TestNotificationBrokerBeforeExecAndCleanup` fails with 'startup hook lost' on main (pair:6, 2026-10-12, while closing #483): it reproduces alone, and with every #483-changed Go source replaced by origin/main through a Go overlay, so it predates #483.

## Spec

Find when it started failing (bisect on main), confirm the cause, and fix the root, not the test's timing. Never under machine load.

## Done when

- The test passes 20 runs in a row on main; the fix fails it when reverted; the cause is in the Log.

## Plan

- [ ]

## Log

### 2026-10-12
