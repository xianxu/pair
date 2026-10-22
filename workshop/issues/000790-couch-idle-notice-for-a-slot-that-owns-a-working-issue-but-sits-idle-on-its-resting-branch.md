---
id: 000790
status: open
deps: []
github_issue:
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
collaboration-mode: collaboration
card_mirror: '403e96dcfd3c4c6128d5bdd37442c5d51f152f91' # card fields mirrored from issue-cards; edit via sdlc
---

# Couch: idle notice for a slot that owns a working issue but sits idle on its resting branch

## Problem

## Spec

10-21: pair:6 owned pair#759 (status working) but after landing #776 at 16:54 it returned to its resting branch and
sat idle until 20:00 with no notice. Couch's "waiting 10 min idle off its resting branch" fires only off the resting
branch, so an owner idle on its resting branch is invisible to the hub.

Fix: the idle notice also fires for a slot that **owns a working issue** (sdlc's owner record, already readable)
and has been idle 10 minutes, whatever branch it's on: "slot X idle 10 min while owning #N (working)". No new
persisted state; the same mute file applies.

## Done when

- A slot idle on its resting branch while owning a working issue gets the notice (test with a fake owner source).
- A slot on its resting branch owning nothing still gets no notice (test).

## Plan

- [ ]

## Log

- 10-21 22:0x (ops:0): **correction to the Spec's premise.** pair:6 was also listed in Couch's
  `wait-notice-off` mute file (with most worker slots: ariadne:1/4/5, pair:1/2/3/5/6/7, parley.nvim:1/2/3), so the
  idle notice was suppressed regardless of branch. The TL cleared the worker mutes (kept :0 slots, ops:1, echo:2).
  The resting-branch gap may still be real; verify whether the notice fires for an owner idle on its resting branch
  with no mute before building. Also worth considering: mutes that expire (a TTL) instead of a permanent list.

### 2026-10-21
