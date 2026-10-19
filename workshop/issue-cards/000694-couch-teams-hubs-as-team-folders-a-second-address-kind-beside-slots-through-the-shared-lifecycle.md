---
id: 000694
status: open
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
github_issue:
---

# Couch teams: hubs as team folders, a second address kind beside slots, through the shared lifecycle

## Problem

Couch treats every supervised agent as a slot (a code worktree, `repo:N`). The TL hub is just "the slot Couch started from", and a second hub means a second ops checkout whose state hides in a worktree path. The operator (2026-10-18, project introduce-team): split the concepts. Slots are workers in code worktrees, governed by sdlc. Hubs are teams, each a folder `teams/<team>/` in an operations repo, governed by tl-lead, and named after the team.
