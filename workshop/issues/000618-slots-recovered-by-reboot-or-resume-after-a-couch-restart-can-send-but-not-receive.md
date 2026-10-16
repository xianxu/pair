---
id: 000618
status: open
deps: []
github_issue:
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
card_mirror: '09602efe8b3d0afc84580deeba81a98fccee5fa6' # card fields mirrored from issue-cards; edit via sdlc
---

# Slots recovered by reboot or resume after a Couch restart can send but not receive

## Problem

After the 2026-10-16 14:57 Couch restart, a slot recovered with `couch --reboot` (pair:3, fresh, tag 1-pair-58) or
`couch --resume` (ariadne:5, tag 1-ariadne-46) can **send** to the hub but **cannot receive**: every
`couch --send-to` to it returns "Query couch --message-status … before any further send", and the status says
"receipt unavailable: no connected recipient retains this message". The agent is live and idle at its prompt.

- ariadne:5 recovered only after `couch --relaunch ariadne:5 --confirm --same-binary` (a plain relaunch refused as
  stale-binary, "changes nothing").
- pair:3 can't be relaunched either: "resume-binding-provisional: its agent has not completed a turn yet", although
  its agent completed a turn (it sent the hub its recovery report).
- The first `--relaunch` of pair:7 and of ariadne:5 each timed out at ~16 s and left a "recover / abandon" choice;
  a second call then succeeded.

So a recovered slot looks healthy in `couch --list` and to the operator, yet silently drops the TL's dispatches.

## Spec

- Find why the recovered wrapper is not a connected recipient (registration on reboot/resume vs. on first turn;
  the binding stays "provisional" after a completed turn).
- One state, one action: a slot that can't receive shows so in `couch --list`, with the one command that fixes it.
- The first relaunch call shouldn't strand a recover/abandon choice on a ~16 s timeout.

## Done when

- Tests: a rebooted and a resumed slot each receive a `--send-to` without a relaunch; a completed first turn makes the
  binding non-provisional.
- Live: after a Couch restart, reboot one slot and resume another; both receive a send.

## Plan

- [ ]

## Log

### 2026-10-16
