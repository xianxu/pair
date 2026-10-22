---
id: 000819
status: open
deps: []
github_issue:
created: 2026-10-22
updated: 2026-10-22
estimate_hours:
collaboration-mode: collaboration
card_mirror: '5a166ce53c028e845415707d59cb8829ce681aab' # card fields mirrored from issue-cards; edit via sdlc
---

# Collapse tab bar: a one-glyph-per-slot state strip next to +N

## Problem

## Spec

Operator request (10-22, after accepting pair#796's collapse tab bar): next to a collapsed group's `+N`, show **one
glyph per hidden slot** for its state, so the worktrees' activity reads at a glance without opening the menu.
The operator's sketch (using ⌥ for the branch glyph): `pair⌥  +9 ⌥⌥+-⌥*--⌥`, nine glyphs for nine hidden slots.

- Reuse the state glyphs the full chips already show (e.g. the branch glyph for a slot on an issue branch, the
  attention/bell and other existing markers); one cell each, in slot order; no new state vocabulary.
- Width: the strip counts toward the collapse planner's width budget (grapheme-correct, per pair#796's width fix); if
  it doesn't fit, fall back to `+N` alone (attention still forces `!`).
- Clicking anywhere on `+N` or its strip opens that group's menu (whole-chip hitbox).
- Live slots only, consistent with pair#796's counts.

## Done when

- A collapsed group renders one glyph per hidden live slot in order, using the existing chip glyphs (test).
- When the strip doesn't fit, the chip falls back to `+N` with no clipping mid-glyph (test).
- The strip is part of the clickable chip (physical-cell click test).

## Plan

- [ ]

## Log

### 2026-10-22
