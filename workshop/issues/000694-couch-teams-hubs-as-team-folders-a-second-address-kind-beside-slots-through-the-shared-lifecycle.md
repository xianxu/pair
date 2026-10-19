---
id: 000694
status: open
deps: []
github_issue:
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
collaboration-mode: collaboration
card_mirror: '4576de1461a971fdb3c5716acae09ed8f3bddc20' # card fields mirrored from issue-cards; edit via sdlc
---

# Couch teams: hubs as team folders, a second address kind beside slots, through the shared lifecycle

## Problem

Couch treats every supervised agent as a slot (a code worktree, `repo:N`). The TL hub is just "the slot Couch started from", and a second hub means a second ops checkout whose state hides in a worktree path. The operator (2026-10-18, project introduce-team): split the concepts. Slots are workers in code worktrees, governed by sdlc. Hubs are teams, each a folder `teams/<team>/` in an operations repo, governed by tl-lead, and named after the team.

## Spec

A hub is a second kind of Couch address, `team` (e.g. `devinfra`, `echo`), backed by a team folder that contains a committed `.hub` (its claims: repos and slots). `couch teams/devinfra ../ops2/teams/finance` starts one hub per team folder; a startup path with a `.hub` is a team, and a plain repo path keeps today's meaning. Several hubs may share one operations checkout, each with its team folder as its working directory. Addressing: `couch --send-to <team>`; `hub` from a slot resolves to the hub that claims it. The switcher shows hubs by team name, apart from slots. Lifecycle: hubs go through the same single lifecycle mechanism as slots (pair#684's Converge: Continue, Resume from disk, Start fresh), without sdlc claims. Session-sync ownership and the attention finder's scope accept a team. Persisted records follow ARCH-EVOLVE.

## Done when

- `couch teams/devinfra teams/echo` from ops starts two named hubs; each echo slot reports to `echo`, others to `devinfra`; restart resumes both hubs (live check).
- Plain `couch` and plain repo paths behave as today.

## Plan

- [ ]

## Log

### 2026-10-18
