---
id: 000709
status: open
created: 2026-10-19
updated: 2026-10-19
estimate_hours:
github_issue:
started: 2026-10-19T20:14:40-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
---

# landing: pilot landing trains in pair (60-minute interval)

## Problem

ariadne#449 (landed, PR 324) adds landing trains: ready merges in one repo are batched and tested with one recorded suite and
merged together, instead of one suite per change. Trains are off unless a repo has `.sdlc/landing.json`. The operator approved
a pilot in pair (2026-10-19): pair's suites are the long pole in the landing queue.
