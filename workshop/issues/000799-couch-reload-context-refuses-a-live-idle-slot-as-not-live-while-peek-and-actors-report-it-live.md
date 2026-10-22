---
id: 000799
status: open
deps: []
github_issue:
created: 2026-10-22
updated: 2026-10-22
estimate_hours:
collaboration-mode: collaboration
card_mirror: '55764357609121898226a9d6fc3f0f0e8d56dd0b' # card fields mirrored from issue-cards; edit via sdlc
---

# couch --reload-context refuses a live idle slot as not-live while peek and actors report it live

## Problem

## Spec

10-22 ~05:20: the TL ran `couch --reload-context ariadne:4 --confirm` for a Codex slot that had checkpointed its
work and asked for a park/continue hand-off (no continuation writer in its setup). First attempt: `refused (busy)`
(the slot had just finished a turn). Seconds later, with the slot idle at its composer: `refused (not-live):
1-ariadne-53 is no longer running`, while `couch --peek ariadne:4` reported `agent codex tag 1-ariadne-53 source
live` and `couch --actors` listed it `occupied`. So reload-context's liveness check disagrees with peek and actors
about the same tag.

Also: workers told to checkpoint before context runs out have no supported hand-off when the continuation writer
isn't exposed (Codex slots); reload-context is the intended route, so it has to work for an idle, checkpointed slot.

## Done when

- reload-context's liveness check uses the same identity source as peek and actors; an idle live slot is accepted
  (test with a fixture matching this state).
- A refusal names the evidence it used (process, pid and tag) so a TL can tell which source is wrong.

## Plan

- [ ]

## Log

### 2026-10-22
