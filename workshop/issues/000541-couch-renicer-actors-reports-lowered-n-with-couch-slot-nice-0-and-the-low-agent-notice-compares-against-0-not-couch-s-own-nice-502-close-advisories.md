---
id: 000541
status: working
deps: []
github_issue:
created: 2026-10-13
updated: 2026-10-13
estimate_hours:
card_mirror: 'c52b54e54c7ac2b5ef92e5d3581a6a83683a3398' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-13T17:15:36-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "e38c4c61", done: "f79896fc"}
---

# couch renicer: --actors reports lowered=N with COUCH_SLOT_NICE=0, and the low-agent notice compares against 0 not Couch's own nice (#502 close advisories)

## Problem

Two advisories from #502's close review (PR #313), filed at the TL's request:
1. With `COUCH_SLOT_NICE=0` (renicer off) `couch --actors` prints `lowered=N`: `PlanRenice` counts every
   descendant as already at the level, and the column shows Lowered+AtLevel.
2. The "agent below normal" notice fires when the agent's nice is above 0. A Couch started from a niced shell
   launches every agent at its own nice, so every slot gets advice (relaunch) that can't help.

## Spec

- At level 0 nothing is lowered and nothing is counted as at the level: `--actors` shows `lowered=0`.
- The notice fires only when a confirmed agent runs below Couch's own priority: its nice is above Couch's,
  read from the same snapshot (Couch's pid is in it; missing → 0).

## Done when

- `PlanRenice` at level 0 reports `AtLevel` 0; a test pins it.
- A renicer test: Couch at nice 5 and the agent at 5 → no notice; the agent at 10 → one notice.

## Plan

- [x] level 0 counts nothing (`couchcore/renice.go`), test.
- [x] notice against Couch's own nice (`couchcmd/renicer.go`), tests.

## Log

### 2026-10-13
- Fixed both. Each new assertion fails on the old code (level 0 counted AtLevel; the notice compared with 0).
  README and atlas wording follow.
