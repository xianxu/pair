---
id: 000321
status: open
deps: [pair#319]
github_issue:
created: 2026-09-24
updated: 2026-09-24
estimate_hours:
---

# Diverged slot glyph uses amber, not red

## Problem

#319 draws the diverged slot glyph `±` in red and the dirty mark `*` in amber.
In live use, the operator wants one attention colour: `±` should use the same
amber as `*`.

## Spec

- `couchtty.slotGlyphSGR` (the single per-character styling decision shared by
  the tab bar and the switcher) returns `attentionSGR` for
  `couchcore.SlotGlyphDiverged` as well as `SlotGlyphDirty`.
- Remove `slotAlertSGR`; nothing else uses it.
- No change to which glyphs appear (`couchcore.SlotGlyph` is untouched).
- Update the README glyph table (`±` "(red)" → "(amber)") and the #317/#319
  section of `atlas/couch.md`.

## Done when

- `TestSlotGlyphColoursInBothViews` asserts `±` and `*` are amber in both views,
  `+`, `-` and the branch glyph are uncoloured, and nothing is coloured without
  256-colour support; it fails if `±` loses its colour.
- Goldens are byte-unchanged (they strip ANSI).
- Live check: a diverged slot shows an amber `±`.

## Plan

- [ ]

## Log

### 2026-09-24

- Filed from the #319 smoke test. Blocked on #319 landing on origin/main; then
  pull slot 1 and implement here.

