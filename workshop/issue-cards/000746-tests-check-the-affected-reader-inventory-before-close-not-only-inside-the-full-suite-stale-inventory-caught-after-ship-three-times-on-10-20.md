---
id: 000746
status: working
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
github_issue:
started: 2026-10-20T14:55:45-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
---

# tests: check the affected-reader inventory before close, not only inside the full suite (stale inventory caught after SHIP three times on 10-20)

## Problem

pair#725's affected-reader inventory (`tests/affected-go-readers.json`) is checked only inside the recorded full
suite, so slots discover a stale inventory after their close review is SHIP, at the most expensive point. On 10-20
it cost three items a suite slot each plus a re-close: pair#727 (sank train 190 for 1 h 45 m), pair#726 (one
fixture file) and pair#706 (eight test files).
