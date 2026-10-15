---
id: 000574
status: open
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
card_mirror: 'e3a8f920610576ed9b75c9c1afd9ba054d1f35ef' # card fields mirrored from issue-cards; edit via sdlc
---

# Couch tells the hub about slot lifecycle events (start, park, relaunch, exit, recovery) as system messages

## Problem

The hub (TL) isn't told about slot lifecycle events. When the operator parks a slot (Alt+n), relaunches it, brings a
new slot up (parley.nvim:3 on 2026-10-15), or a slot's agent exits, the TL learns it only by polling `couch --actors`,
by peeking, or from a worker's own report. Tonight's evidence: stalled slots went unnoticed for hours, and messages to a
relaunched slot were lost. The reverse direction already exists: on connect, Couch sends a worker a "Couch restarted
and the hub is available…" notice, attributed to ops:0.

## Spec

- Couch sends the hub a **system lifecycle message** for each slot event: started (new or resumed), parked (Alt+n),
  relaunched, agent exited, recovered after a Couch restart (Continue / Resume / Start fresh), and needs-relaunch.
  One line each: slot, event, agent/model, branch and issue if known, and whether it was operator-initiated.
- Clearly marked as a system message (not from a peer slot), so the hub's rules treat it as information, never as a
  request; it's batched or coalesced so a Couch restart's many reattaches produce one summary, not twenty messages.
- Opt-out per hub (some operators may not want it), default on when a TL hub is configured.
- The worker-side "hub is available" notice should say it's from Couch, not "from ops:0".

## Done when

- Tests: park, relaunch, new slot, agent exit and post-restart recovery each produce the expected hub message; a
  restart's reattaches coalesce into one summary.
- `couch --skill` documents the messages; tl-lead says how the hub treats them (update Slots, no reply).

## Plan

- [ ]

## Log

### 2026-10-15
