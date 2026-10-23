---
id: 000830
status: open
deps: []
github_issue:
created: 2026-10-22
updated: 2026-10-22
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'd20aef08044f3d8668379d16021aff0c600bc6a0' # card fields mirrored from issue-cards; edit via sdlc
---

# Global Alt+x marks parked slots 'asleep'; one 'wake all' brings back exactly those

## Problem

## Spec

Operator idea (10-22, night): when the operator does the **global Alt+x** (park everything and quit), Couch should
mark the slots it parks that way as **asleep** ("put to sleep" during a restart), a flag on the parked slot distinct
from a slot the operator parked individually. That also answers the operator's earlier observation that "parked"
mixes two kinds: slots not in use, and slots that were merely running when parked.

- Global Alt+x (and `couch --drain`-driven full shutdowns) records each parked slot as asleep, with the time; an
  individual park, or a slot already parked, stays plain parked.
- On the next start, asleep slots are what a **"wake all"** action (one explicit operator or TL gesture, consistent
  with pair#739's no-automatic-resume rule) brings back; plain parked slots stay parked. tl-lead's fleet resume
  (ariadne#564) uses the asleep set instead of guessing from issue ownership.
- Waking or resuming a slot clears its flag; the switcher shows asleep distinctly from parked.
- Related: pair#827 (Alt+x must finish in one press), pair#739 (cold start stays manual).

## Done when

- After a global Alt+x, previously live slots show asleep and previously parked ones show parked (test).
- One "wake all" resumes exactly the asleep set (test); individually parked slots are untouched.
- The flag survives a full restart and clears on wake.

## Plan

- [ ]

## Log

### 2026-10-22
