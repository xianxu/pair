---
id: 000471
status: open
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: '7c3c4dfdf99ef6b80710c4bec2bc4e68e4ac3559' # card fields mirrored from issue-cards; edit via sdlc
---

# broadcast-web: replay a recording in the viewer with a timeline scrubber

## Problem

Wave 2 of couch-broadcast-web (ops `teams/devinfra/projects/couch-broadcast-web/project.md`). A recording on disk can be queried by agents (pair#462) but not watched: the viewer (pair#460) only plays a live stream.

## Spec

- Open a recording directory in the viewer (served by Couch: a route listing recordings and serving one) and play it with the same model and renderer as live.
- A timeline scrubber: segments and pauses shown as gaps; seek by reducing from the latest `seg` before t (the reference reducer's rule); play, pause, speed.
- Notes appear at their moment on their element; strokes replay; who drove (`by`, `ctl`) is visible on the timeline.
- Agent-facing: the URL of a recording at time t (e.g. `#t=1530`) so an issue or a DOM submission can link to the exact moment.

## Done when

- Every fixture replays with seeking to any pin giving the reducer's canonical state (reuse #460's checks).
- Notes and strokes show at their moments; a `#t=` link opens at that moment.
- The operator smoke-tested replay of a real recording (after #458), scheduled by the TL.

## Plan

- [ ]

## Log

### 2026-10-12
