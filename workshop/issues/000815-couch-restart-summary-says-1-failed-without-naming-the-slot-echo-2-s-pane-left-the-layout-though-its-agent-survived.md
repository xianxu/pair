---
id: 000815
status: open
deps: []
github_issue:
created: 2026-10-22
updated: 2026-10-22
estimate_hours:
collaboration-mode: collaboration
card_mirror: '544b8a403179862a64c6c643756eb1d14ee73ac0' # card fields mirrored from issue-cards; edit via sdlc
---

# Couch restart summary says '1 failed' without naming the slot; echo:2's pane left the layout though its agent survived

## Problem

## Spec

The 10-22 ~13:50 Couch restart's lifecycle update said "restart reattachments complete; 1 failed" but its per-slot list
named only recovered slots, so the failed one had to be inferred (echo:2, by ops-echo:0 noticing it missing). After
the restart echo:2's agent was alive (it answered the restart notice at 13:52; `--actors` occupied; later `--peek`
source live), but the operator's console layout no longer showed its pane, and for a while `--peek` reported source
recording, not live.

Fix:
- The restart summary names every failed reattachment with its slot, the failing step and a one-line recovery hint
  (as the success rows do).
- Find why echo:2's reattach failed while its agent survived, and why the pane left the console layout; a live agent
  whose reattach failed should be re-attachable from the console without a relaunch.

## Done when

- A restart with one injected reattach failure reports that slot by name with its reason and recovery (test).
- The echo:2 case's root cause is recorded; a regression covers it.

## Plan

- [ ]

## Log

### 2026-10-22
