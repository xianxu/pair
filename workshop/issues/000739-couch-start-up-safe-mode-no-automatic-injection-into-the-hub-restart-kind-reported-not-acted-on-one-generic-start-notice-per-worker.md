---
id: 000739
status: open
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
collaboration-mode: collaboration
card_mirror: '340f47d44a295cdc1a55ce71cd1d4dfc3009581b' # card fields mirrored from issue-cards; edit via sdlc
---

# couch: start-up safe mode: no automatic injection into the hub, restart kind reported not acted on, one generic start notice per worker

## Problem

After a restart, Couch acts on its own: it injects system notices into every slot, the hub included ("Couch
restarted and the hub is available…"), and recovery has been attempted automatically at times. The operator (10-20,
drain redesign, ops side doc `2026-10-20-17-drain-redesign.md`): no automatic recovery; a restart may be for
something else, or the setup may be wedged; the operator tells the TL "start recovery" in one line.

## Spec

- **Hub safe mode:** on start, the hub's TL gets no automatic system message injection and nothing is auto-started
  inside the hub.
- **Restart kind, reported only:** Couch tells a normal Couch restart (an attachable Zellij session for the fleet
  exists) from a full shutdown (none), and reports it to the TL once, in one line ("full shutdown detected: N slots
  parked; recovery waits for the operator"), without acting on it.
- **Worker start notice:** when a worker slot starts (resume or fresh), Couch sends one short generic notice: "you
  restarted; reread your issue Log and continue where you left off; report to your hub". The TL may follow with its
  own message; Couch never sends more than the one generic notice per start.

## Done when

- Tests pin: no injection into the hub on start; restart-kind classification from fixture Zellij state; exactly one
  generic notice per worker start.
- Live: after the next full restart, the TL sees one report line and nothing else until the operator speaks.

## Plan

- [ ]

## Log

### 2026-10-20
