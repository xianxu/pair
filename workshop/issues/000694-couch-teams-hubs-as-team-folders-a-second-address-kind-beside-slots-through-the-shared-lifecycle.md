---
id: 000694
status: working
deps: []
github_issue:
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
collaboration-mode: collaboration
card_mirror: '0f6dca2a2a2e3c3c6d540fe6d79eb5396822e1de' # card fields mirrored from issue-cards; edit via sdlc
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

## Spec

A hub is a second kind of Couch address, `team` (e.g. `devinfra`, `echo`), backed by a team folder that contains a committed `.hub` (its claims: repos and slots). `couch teams/devinfra ../ops2/teams/finance` starts one hub per team folder; a startup path with a `.hub` is a team, and a plain repo path keeps today's meaning. Several hubs may share one operations checkout, each with its team folder as its working directory. Addressing: `couch --send-to <team>`; `hub` from a slot resolves to the hub that claims it. The switcher shows hubs by team name, apart from slots. Lifecycle: hubs go through the same single lifecycle mechanism as slots (pair#684's Converge: Continue, Resume from disk, Start fresh), without sdlc claims. Session-sync ownership and the attention finder's scope accept a team. Persisted records follow ARCH-EVOLVE.

## Done when

- `couch teams/devinfra teams/echo` from ops starts two named hubs; each echo slot reports to `echo`, others to `devinfra`; restart resumes both hubs (live check).
- Plain `couch` and plain repo paths behave as today.

## Plan

Durable plan: `workshop/plans/000694-couch-teams-hubs-as-team-folders-a-second-address-kind-beside-slots-through-the-shared-lifecycle-plan.md`.

- [ ] M1 — The team model (pure): TeamIdentity, hubs keyed by name, `.hub` directive lines preserved, one namespace for teams and families
- [ ] M2 — Teams run: startup with team folders, team registration and addressing, switcher sections and the bar's team display (capitals, team/repo merge), session-sync owner, actors JSON (parley attention scope filed)
- [ ] M3 — Team lifecycle on #695 (split from #684): lease, transcript binding and failure records for team agents; restart resumes both teams (live check)

## Log

### 2026-10-18

- Claimed and planned on the hub's dispatch (delivery 8d0bb794), from the introduce-team umbrella design §1–3. Measured: no `.hub` exists on this machine; `sdlc workspace --json` from `ops/teams/devinfra` reports `ops:0`, so without `COUCH_TEAM` every team agent in a checkout would register as the same slot (guarded by `TestTeamWrapperNeverRegistersAsCheckoutSlot`). Today's multi-hub startup (`start-hub`) and ownership rules are reused, keyed by name instead of root (ARCH-DRY). Open to the hub: #684 landing at its M2 close so M3 can start; the `.hub` directive-line reading of design §2.
- Operator display input (hub delivery d7c631c0) folded in as plan Revision 1: teams in capitals with no `:N`, a pure bar-merge rule for an adjacent same-named team and repo, and teams allowed to share a repo's name (bare name addresses the team). #684's M1–M2 split into #695 (delivery 51ef8069); M3 now depends on #695.
