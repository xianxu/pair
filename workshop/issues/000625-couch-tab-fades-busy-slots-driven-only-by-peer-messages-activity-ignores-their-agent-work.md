---
id: 000625
status: open
deps: []
github_issue:
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
card_mirror: '15d9628d15e2eb44329cd82d4d83c34759fcc2ef' # card fields mirrored from issue-cards; edit via sdlc
---

# Couch tab fades busy slots driven only by peer messages (activity ignores their agent work)

## Problem

Couch's tab bar fades a slot's chip when the slot looks idle for a day or more (`couchtty/reserve.go:370-390`,
`idle_shade.go:16-39`), from `MenuState.Activity[thread]` (`console_presentation.go:37-39`, merged at `menu.go:461`):
"each thread's last operator input or agent work". On 2026-10-16 the operator saw `ariadne:2` faded while it was
actively working (running ariadne#412's recorded suite after a day of peer-message-driven turns). Slots driven only by
the TL's peer messages apparently never register "agent work", so they fade even when busy: the same blind spot pair#618
found for codex binding confirmation (only Pair-log turns written by the draft pane count).

## Spec

- Find what feeds `Activity` today and confirm the hypothesis (peer-message turns, background shells and agent output
  don't count).
- Activity must reflect agent work however the turn started: a turn in progress, a completed turn (peer message, draft
  pane, launch prompt), or a running background command the agent is waiting on. Reuse the signal Couch already has for
  "turn open" (the busy check behind relaunch's refusal) rather than adding a new probe.
- Fading then means what the operator reads it as: nobody (operator or agent) has done anything there for a day.

## Done when

- Tests: a slot whose only turns come from peer messages stays bright; a slot idle for over a day fades.
- Live: a working, TL-driven slot is not faded.

## Plan

- [ ]

## Log

### 2026-10-16
