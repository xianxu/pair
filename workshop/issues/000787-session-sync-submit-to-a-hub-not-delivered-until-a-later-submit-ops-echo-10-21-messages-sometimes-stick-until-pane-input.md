---
id: 000787
status: open
deps: []
github_issue:
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
collaboration-mode: collaboration
card_mirror: '97f8761eb7813469a6617e282e9b02d122efb271' # card fields mirrored from issue-cards; edit via sdlc
---

# Session-sync submit to a hub not delivered until a later submit (ops-echo 10-21); messages sometimes stick until pane input

## Problem

## Spec

From ops-echo:0's TL (10-21): the operator submitted an edit to `session.side/04-agent-dev-suggestions.md` (ops-echo)
and the hub never received a submit delivery for it; the hub saw the change only when a later submit (registered
15:23:04) arrived. The operator also reports, more generally, that messages to a hub sometimes stick until a human
types in the hub pane.

The earlier hub-stall class (pair#735, no-op resize cursor) landed 10-20 and is in the running build, so this is
either a new stall path or a lost/coalesced session-sync submit event.

Investigate with the session-sync event log and the hub's delivery records for that window: was the first submit
registered (owed) at all, delivered but not recognized, or held behind composer recognition? Fix the cause; a
submit must always produce exactly one delivery (or a visible owed entry) without needing pane input.

## Done when

- The 10-21 case is explained from records.
- A regression covers the identified path (lost/coalesced submit, or delivery held until input).

## Plan

- [ ]

## Log

### 2026-10-21
