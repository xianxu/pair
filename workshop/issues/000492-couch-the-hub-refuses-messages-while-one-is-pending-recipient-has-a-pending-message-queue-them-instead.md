---
id: 000492
status: open
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: '2237161a37643af9a432a70c53f567fc12273aa1' # card fields mirrored from issue-cards; edit via sdlc
---

# couch: the hub refuses messages while one is pending ('recipient has a pending message'); queue them instead

## Problem

10-12 ~10:47–11:04: pair:3 tried for about 17 minutes to send its #488 plan to `hub` (ops:0) and was refused every time: `couch: refused: recipient has a pending message`. With one hub and six or more slots reporting to it (now the constitution's rule, ariadne#336), a one-pending-message limit per recipient makes the hub a bottleneck: whenever the TL is busy, every other slot's report or question is refused, and each slot has to keep retrying or give up. pair:3 gave up and left its plan on screen.

## Spec

Find where the one-pending limit lives (per recipient, per sender pair, or the hub's admission) and why it exists (ordering? avoiding typing over a draft?). Then let a recipient, at least the hub, queue several pending messages and deliver them in order as its turns end, within the existing per-sender rate limits and capacity (the hub's bucket is already 64). The sender gets `queued` with a position, not `refused`. Keep any genuine protection (never typing into a picker or a draft), but as holding, not refusing.

## Done when

- Several slots can send to the hub while it is mid-turn; each gets `queued`, and all are delivered in order once it settles.
- Tests for ordering and for holding behind a picker/draft; the 10-12 case reproduced and fixed.

## Plan

- [ ]

## Log

### 2026-10-12
