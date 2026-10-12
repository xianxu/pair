---
id: 000477
status: working
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: '277f4c9c15a67b10219e623b56443c8280929893' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T05:09:29-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
---

# broadcast-web: replay inside a running Couch, for remote viewers (mount ReplayServer on the shared Host)

## Problem

pair#471 (PR #266) replays recordings via `couch --broadcast-web-replay`, on loopback only. A colleague on another machine can't open a replay link, and an issue or a DOM submission can't point a reviewer at a moment of a recording (couch-broadcast-web, ops `teams/devinfra/projects/couch-broadcast-web/project.md`).

## Spec

- Mount `webcast.ReplayServer` on the shared broadcast Host inside a running Couch (the endpoint and tunnel #454 shares), behind the cast's token model, so replay links (`…/r/<id>/#t=MS`) work for remote viewers.
- How a replay link is produced: from Couch (e.g. a menu action on a recording, or `couch --broadcast-web-replay ID` printing the public link while Couch runs), and for agents (the query CLI can print the link to a moment).
- #471's advisory: a recording whose last line doesn't parse lists its length as unknown, not 0.
- Replay draws notes and strokes with #470's modules once #470 lands (coordinate through the TL).

## Done when

- A replay link opened from another device plays a real recording through the tunnel, seeks, and shows notes.
- The length advisory fixed; tests; the operator's smoke test, scheduled by the TL.

## Plan

Durable plan: `workshop/plans/000477-broadcast-web-replay-inside-a-running-couch-for-remote-viewers-mount-replayserver-on-the-shared-host-plan.md`.

- [ ] M1 — replay shares (one token per recording) on the shared Host; broker op `replay-share`; `couch --broadcast-web-replay ID` prints the public link while Couch runs (loopback fallback, `--local`); `--broadcast-web-query DIR link`; length-unknown advisory; atlas + skill
- [ ] M2 — after #470 lands: merge main; replay draws notes and strokes with #470's modules; live check

## Log

### 2026-10-12
