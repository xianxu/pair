---
id: 000514
status: open
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: '8c98766635be4eaab480dbb43fcbe9fa4f2a0ffc' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T18:20:35-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
---

# test-perf-capture asserts a 6 s wall-clock budget on a shared loaded machine; fails under load on main

## Problem

`make test-perf-capture` (`doctor/perf_test.sh`) asserts that `pair`'s scrollback capture finishes within a 6 s
wall-clock budget. On the shared machine, with other slots' full suites running, it fails on unchanged main:
10-12 18:14, load 64/72/62, main failed twice (7 s, 16 s); at load ~180 it took 11-12 s. It blocked #505's close
(which doesn't touch capture) until the TL waived it. A wall-clock bound on a shared, loaded machine measures the
machine, not the code (see lessons: "A stopwatch ceiling hides the race it was meant to catch", #479).

## Spec

## Done when

-

## Plan

- [ ]

## Log

### 2026-10-12
