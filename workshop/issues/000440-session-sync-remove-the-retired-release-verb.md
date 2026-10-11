---
id: 000440
status: open
deps: []
github_issue:
created: 2026-10-11
updated: 2026-10-11
estimate_hours:
card_mirror: '64fb6a8b6d0d5d0b3f4430e6e0f085be34d8febf' # card fields mirrored from issue-cards; edit via sdlc
---

# session-sync: remove the retired release verb

## Problem

pair#439 retired `couch --session-sync release` (the agent holds no lock) but kept it as a no-op for one release so older skills calling it don't break. Its end is part of #439's design (ARCH-FUNERAL).

## Spec

Once the ops xx-tl skill and every session-sync caller use `read` + `write --if-hash`, drop the `release` verb from the parser, usage, and tests; an old call then fails as an unknown verb.

## Done when

- `couch --session-sync release FILE` is a usage error; no doc or skill mentions it.

## Plan

- [ ] Grep ops and parley.nvim for `session-sync release`; remove the verb and its tests.

## Log

### 2026-10-11
