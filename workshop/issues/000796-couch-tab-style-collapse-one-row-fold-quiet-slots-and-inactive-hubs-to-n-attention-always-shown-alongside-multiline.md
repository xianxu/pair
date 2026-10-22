---
id: 000796
status: open
deps: []
github_issue:
created: 2026-10-22
updated: 2026-10-22
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'bb343ee3923710f39f348bf6db16d0c98fdbebca' # card fields mirrored from issue-cards; edit via sdlc
---

# Couch: --tab-style collapse (one row; fold quiet slots and inactive hubs to +N, attention always shown) alongside multiline

## Problem

## Spec

Operator (10-22, after accepting pair#779's two-line wrap): "another way to deal with the width limit is to keep it at
one line but have some intelligent collapsing. for example, the inactive hub's worker slots can be collapsed. slots
without either notification (amber) or blocking (red) can be collapsed to just a number, e.g.
`pair  :1 (red)  :4 (amber)  +4` (indicating 4 more slots hidden). can we make such a mode and selectable from command
line? maybe --tab-style collapse | multiline."

Design:
- `couch --tab-style multiline|collapse` at launch (default `multiline`, today's pair#779 behaviour); a runtime toggle
  key is a possible follow-up, not in scope.
- **collapse:** one tab row. Within each repo group, show the active slot, every slot needing attention (amber
  notification, red blocked) and the group's :0, and fold the rest into `+N`. Groups of an **inactive hub** fold the
  same way. Clicking `+N` expands that group in place until focus leaves it (or opens the switcher filtered to it;
  choose in the plan). Attention always wins over ordinary slots, as in pair#779's overflow; the active slot is never
  folded.
- Reuse pair#779's measured packing, hit-map and chrome-row publication: collapse is a different input to the same
  packer, not a second renderer. The status-text cap (pair#789) applies in both modes.

## Done when

- With `--tab-style collapse`, a fleet of N slots per group renders in one row: active plus attention slots shown,
  the rest as `+N` (render test), and clicking `+N` expands that group (test).
- `multiline` is unchanged (existing tests).
- Staged for an operator look.

## Plan

- [ ]

## Log

### 2026-10-22
