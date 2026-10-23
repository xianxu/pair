---
id: 000824
status: open
deps: []
github_issue:
created: 2026-10-22
updated: 2026-10-22
estimate_hours:
collaboration-mode: collaboration
card_mirror: '7578a1f5146984c3758383cbe2ea5e7464bb5c99' # card fields mirrored from issue-cards; edit via sdlc
---

# Tab bar: the whole slot chip is clickable, including its checkout-state glyph

## Problem

## Spec

Operator report (10-22, screenshot of a chip `:1⮭`-style branch glyph): on the Couch tab bar, only the slot number
(`:1`) of a slot chip is clickable; the checkout-state glyph after it (the branch symbol, and the other SlotGlyph
markers) is not. The whole chip should be one click target.

Likely the same class as pair#796's `+N` fix: the glyph is measured as one column but drawn as two (or the hit span is
built from the label only), so the hitbox stops before the glyph. Fix the chip hit span from the rendered physical
cells (the grapheme-correct width from #796), covering label plus glyphs, in both tab styles; and sweep other chips
with trailing glyphs for the same gap.

## Done when

- Clicking any physical cell of a slot chip, including each state glyph, selects that slot (physical-cell click test),
  in collapse and multiline styles.
- A sweep finds no other chip whose hit span is narrower than its drawn cells (test or listed audit).

## Plan

- [ ]

## Log

### 2026-10-22
