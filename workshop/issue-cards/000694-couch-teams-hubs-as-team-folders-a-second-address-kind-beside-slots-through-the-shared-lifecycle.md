---
id: 000694
status: open
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
github_issue:
started: 2026-10-18T22:48:50-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
---

# Couch teams: hubs as team folders, a second address kind beside slots, through the shared lifecycle

## Problem

Couch treats every supervised agent as a slot (a code worktree, `repo:N`). The TL hub is just "the slot Couch started from", and a second hub means a second ops checkout whose state hides in a worktree path. The operator (2026-10-18, project introduce-team): split the concepts. Slots are workers in code worktrees, governed by sdlc. Hubs are teams, each a folder `teams/<team>/` in an operations repo, governed by tl-lead, and named after the team.
