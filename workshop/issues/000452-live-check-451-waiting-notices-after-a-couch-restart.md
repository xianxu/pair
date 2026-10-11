---
id: 000452
status: open
deps: []
github_issue:
created: 2026-10-11
updated: 2026-10-11
estimate_hours:
card_mirror: '643b85c375fe188d38b4ab1f26cd51934c1709b4' # card fields mirrored from issue-cards; edit via sdlc
---

# Live-check #451 waiting notices after a Couch restart

## Problem

pair#451 shipped waiting notices to the hub, tested against fakes over real sockets. Its third Done-when, that the TL actually receives a notice from the live Couch, needs the operator to restart Couch on a binary that contains #451, which the implementing slot cannot do. This issue holds that check.

## Spec

After #451 lands: pull and `make build` in pair:0 (`~/workspace/pair`), restart Couch from ops:0's checkout so ops:0 is the hub. Optionally start Couch with `COUCH_WAIT_NOTICE_AFTER=2m` to shorten the wait. Leave one worker slot waiting: a permission dialog, a turn ending with a question, or idle off its resting branch.

## Done when

- ops:0 receives exactly one `slot repo:N waiting …` notice for the waiting slot, and none for a slot that ended normally on its resting branch.
- The slot's next activity followed by another wait produces one more notice.

## Plan

- [ ] Operator restarts Couch on the #451 build; observe ops:0; record the notice text here.

## Log

### 2026-10-11
