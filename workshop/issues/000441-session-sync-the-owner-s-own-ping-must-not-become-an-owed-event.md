---
id: 000441
status: open
deps: []
github_issue:
created: 2026-10-11
updated: 2026-10-11
estimate_hours:
card_mirror: '83fbe36548e5fb140d43cf5fad229d2d62494536' # card fields mirrored from issue-cards; edit via sdlc
---

# session-sync: the owner's own ping must not become an owed event

## Problem

parley.nvim#314 pings the owner slot ("session-sync ping: <file> owes a response since …") when a session-sync file has been owed for 5 minutes. The ping is itself a delivery to the owner slot, and pair#437/#439's ledger records every delivery to the owner as an owed event. So the ping creates a new owed event, which the next ping episode answers with another ping: a self-sustaining loop whenever the agent doesn't fold (seen 10-11 09:41: the ping `ba7bbd61` from ops:0 to ops:0 appeared in `Owed`).

## Spec

A delivery whose sender is the owner slot itself and whose text is a session-sync ping (`session-sync ping:`) is not recorded as owed. Same identity rule the breaker uses (pair#433): sender identity, not text alone, so a peer can't suppress a real owed event by imitating the prefix. A submit (`submitted:`) from the owner's own editor stays owed: it's the operator's turn ending.

## Done when

- A ping from the owner slot leaves `Owed` unchanged (test).
- A `submitted:` from the owner slot and any delivery from another slot are still owed (tests).

## Plan

- [ ] Exclude owner-sent pings in the ledger's record path; tests for the three cases.

## Log

### 2026-10-11
