---
id: 000850
status: open
deps: []
github_issue:
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
collaboration-mode: collaboration
card_mirror: '85359be3310d62c6bb2b6e7ecd9d7e8ffbae240c' # card fields mirrored from issue-cards; edit via sdlc
---

# Cast notes: sticky-note style, minimize to a draggable button, draggable panel, visible add-note on sketches

## Problem

## Spec

**Cast notes: make the floating notes usable on the page (echo operator, 10-23).** In the Chrome cast's floating
notes (broadcast-web), the operator asked for:
1. **Sticky-note style:** the same look as the authoring experience (light yellow, like a sticky note).
2. **Minimize:** collapse the notes into a draggable button; the expanded panel draggable too. Today the notes cover
   the page and block clicking a note to edit it.
3. **Fixed height with a vertical scrollbar** (keep).
4. **A visible way to open or add notes on a sketch** (the operator couldn't find it).
Evidence: ops-echo `session.side/2026-10-23-03-echo-23-board-feedback.md` and its screenshots.

## Done when

- Notes render in the sticky-note style; minimize collapses them to a draggable button and restores them; the
  expanded panel can be dragged; its height is fixed with a scrollbar (tests for each behaviour).
- A sketch shows a visible control to open or add its notes (test), plus a brief operator look.

## Plan

- [ ]

## Log

### 2026-10-23
