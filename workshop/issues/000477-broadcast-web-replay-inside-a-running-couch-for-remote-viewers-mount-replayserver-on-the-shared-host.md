---
id: 000477
status: open
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: '58fc829c824e7583843d2f12e501bc644c54e7a8' # card fields mirrored from issue-cards; edit via sdlc
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

- [ ]

## Log

### 2026-10-12
