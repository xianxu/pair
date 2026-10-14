---
id: 000555
status: working
deps: []
github_issue:
created: 2026-10-14
updated: 2026-10-14
estimate_hours:
card_mirror: '118bb1a73d39ffe392ad0a77ffbbcd06a2a5aa46' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-14T15:02:32-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "bf551f03", done: "c12c8e15"}
---

# Ignore generated TL skills

## Problem

ariadne#350 exports TL skills; refresh generates four unignored paths and dirties every pair slot, blocking planning.

## Spec

Commit exactly weave-generated ignores for xx-tl-lead and xx-tl-worker under .agents/skills and .claude/skills. Keep weave as their source (ARCH-DRY); no runtime change.

## Done when

- The only product diff is those four generated .gitignore additions.
- All four generated skill paths are ignored and a second weave apply leaves tracked files unchanged.
- Quick-flow review and required landing checks pass.


## Plan



## Log

### 2026-10-14

TL approved this exact generated maintenance change and quick flow. Initial refresh completed; diff is exactly four ignore additions. No durable plan needed. Full suite will run because the landing gate requires it.
