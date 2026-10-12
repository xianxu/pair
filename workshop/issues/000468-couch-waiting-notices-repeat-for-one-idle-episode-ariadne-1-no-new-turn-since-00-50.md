---
id: 000468
status: open
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: 'ac6a8750d9522a39d2ec417e2d51f058477e2ee1' # card fields mirrored from issue-cards; edit via sdlc
---

# couch: waiting notices repeat for one idle episode (ariadne:1, no new turn since 00:50)

## Problem

pair#451 promises one waiting notice per episode. ariadne:1 finished its last turn at 00:50 on 10-12 (parked by the TL until a Couch restart) and had no turn since, yet the hub got 'slot ariadne:1 waiting 10 min idle off its resting branch' at least four times (deliveries 19729e90…, d9e1c39b…, 94e5abd7…, 900d3997…, roughly every 30 minutes). Each is noise the TL has to fold.

## Spec

Find what re-arms the episode for a slot that stays idle: an incoming message that is delivered but doesn't start a turn, a wrapper heartbeat, a settle report, or a timer. An episode ends only on real activity (a turn, operator input, a branch change). Fix at the root; keep one notice per episode.

## Done when

- A test with a slot idle off its resting branch for an hour, with periodic wrapper reports, yields one notice.
- The repeat seen on ariadne:1 is explained in the Log.

## Plan

- [ ]

## Log

### 2026-10-12
