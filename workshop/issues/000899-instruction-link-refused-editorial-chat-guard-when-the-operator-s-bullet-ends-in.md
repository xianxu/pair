---
id: 000899
status: open
deps: []
github_issue:
created: 2026-10-25
updated: 2026-10-25
estimate_hours:
card_mirror: 'f7d115ac3820612ab992c5ed717940c9f19189d1' # card fields mirrored from issue-cards; edit via sdlc
---

# Instruction link refused (editorial-chat guard) when the operator's bullet ends in a trailing space

## Problem

## Spec

**`tl-lead status instructions link` is refused when the operator's bullet ends with a trailing space (ops-echo:0,
10-25).** On ops-echo's session.md, linking an answer to instruction item 2 failed: `couch --session-sync write:
refused: editorial-chat at bytes 831:997: preserve the complete rendered chat item and its source membership`. Item 2
is the operator's text ending in a trailing space before the newline; item 1 linked fine. (ops:0's session file had
items with trailing spaces too, e.g. "6. do you have enough pair slots now? ", which linked fine, so the trigger may
be the trailing space combined with something else in that item.) The append of ` → [answer](…)` seemingly changes
how the chat item is rendered or tokenized, so the editorial-chat guard sees the operator's item as modified.

## Done when

- Linking an answer to an instruction bullet whose text ends in trailing whitespace succeeds and preserves the
  operator's text exactly; a test with that bullet shape (and the ops-echo case reproduced).

## Plan

- [ ]

## Log

### 2026-10-25
