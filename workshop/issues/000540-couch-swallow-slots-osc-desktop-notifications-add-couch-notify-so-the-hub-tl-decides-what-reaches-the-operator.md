---
id: 000540
status: open
deps: []
github_issue:
created: 2026-10-13
updated: 2026-10-13
estimate_hours:
card_mirror: 'db175baba8482b25814feac6ee877aaa3eb62e73' # card fields mirrored from issue-cards; edit via sdlc
---

# couch: swallow slots' OSC desktop notifications; add couch --notify so the hub (TL) decides what reaches the operator

## Problem

Agents in slots emit terminal desktop notifications (OSC 9 / OSC 777 / OSC 99 sequences, e.g. 'Claude needs your input'). With many slots under a hub they arrive at the wrong granularity (per agent turn, not per operator decision) and never worked well. The operator's attention is now managed by the TL in the hub (the tl-lead skill), which decides what's worth interrupting for.

## Spec

The operator (2026-10-13):
1. Couch swallows OSC notification sequences from every slot's terminal (the hub's included): they never reach the operator's terminal. Other OSC traffic (titles, hyperlinks, clipboard as Pair already handles it) is unaffected. Log a count per slot for diagnostics, nothing more.
2. A new verb `couch --notify [--title T] MESSAGE` (light) emits one OSC desktop notification to the operator's terminal through Couch's own output. It is meant for the hub (the TL); other callers are refused or downgraded to a hub message (decide in the plan, keep it simple).
3. Docs: `couch --help` and the couch skill say slots' notifications are swallowed and only the hub notifies; ariadne#350's tl-lead prose uses `couch --notify` for decisions the operator must see (smoke tests ready, a question, an incident).

## Done when

- A slot emitting OSC 9/777/99 produces no notification on the operator's terminal; `couch --notify` from the hub produces exactly one; tests cover both and that non-notification OSC passes through.

## Plan

- [ ]

## Log

### 2026-10-13
