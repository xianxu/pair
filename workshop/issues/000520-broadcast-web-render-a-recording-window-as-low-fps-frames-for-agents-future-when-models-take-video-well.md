---
id: 000520
status: open
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: '6be211110d4fdc8454c13dec1be9d46548ca0ef5' # card fields mirrored from issue-cards; edit via sdlc
---

# broadcast-web: render a recording window as low-fps frames for agents (future, when models take video well)

## Problem

Agents can query a broadcast-web recording's past DOM (`couch --broadcast-web-query DIR dom|events`), but not its past pixels: a purely visual change (a CSS animation, a flash) shows as a class change, not as what it looked like. The replay (#471) renders to a web page, not to images or video.

## Spec

Future (the operator, 2026-10-12): when models handle video or image sequences well, add a query that renders a time window of a recording as low-fps frames (e.g. `couch --broadcast-web-query DIR frames --from T1 --to T2 --fps 2`), reusing the fidelity checker's headless replay renderer (#478), writing PNGs (or a short video) with their recording times. Bounded frames and size; never a foreground window.

## Done when

- An agent asked about 'something flashed 10 seconds ago' can look at frames from that window; frames match the fidelity renderer's output at the same times.

## Plan

- [ ]

## Log

### 2026-10-12
