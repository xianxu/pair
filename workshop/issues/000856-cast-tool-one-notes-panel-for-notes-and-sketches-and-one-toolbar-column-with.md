---
id: 000856
status: open
deps: []
github_issue:
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
card_mirror: 'b415e1f86bf0ea88eec5f29f9c5833810a944bde' # card fields mirrored from issue-cards; edit via sdlc
---

# Cast tool: one Notes panel for notes and sketches, and one toolbar column with tools revealed on toggle

## Problem

## Spec

**Cast tool: one Notes panel for notes and sketches, and one toolbar column (operator, 10-23, from their echod
annotations).** Agreed design, from ops `session.side/2026-10-23-24-cast-tool-learnings.md` (with screenshots
described there):
- **One Notes panel for everything:** each note and each sketch is an entry (a sketch entry shows its thumbnail, its
  note, add/edit note, and "show on page"); the panel is the only place notes and sketch notes are read or opened.
  Replaces the per-sketch buttons and pair#850's separate "Sketch notes (N)" button.
- **One toolbar, one column:** live/pause first; Sketch as a toggle that, when on, reveals the palette, Undo, Redo and
  Done below it; Undo and Redo hidden when they can't act. Nothing (notes, popups, status) is drawn over the toolbar.
- **Status messages** (such as "Pending feedback is out of date") show in the Notes panel's header, not as a floating
  box.
The exact positions get settled against a mock flow (the operator's note: "once we have the mock flow set up, we can
be a lot more precise how things are laid out"), so the plan starts with a mock of the layout for the operator to
react to before code.

## Done when

- Notes and sketches are entries in one panel; there are no per-sketch on-page buttons (tests).
- The toolbar is one column with the reveal-on-toggle behaviour, and no other cast UI overlaps it at common viewport
  sizes (tests), plus the operator's live look.

## Plan

- [ ]

## Log

### 2026-10-23
