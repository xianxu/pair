---
id: 000808
status: open
deps: []
github_issue:
created: 2026-10-22
updated: 2026-10-22
estimate_hours:
collaboration-mode: collaboration
card_mirror: '5b86352adac27a2555c5bce0a062e3715d4326fd' # card fields mirrored from issue-cards; edit via sdlc
---

# Cast tools: one vertical bar with pointer / sketch / note modes and one colour picker; Go live fixed bottom right (operator)

## Problem

## Spec

Operator feedback on pair#753's recheck (10-22 ~11:01), point 2, verbatim: "the 'go live' and 'pause' is not well
aligned with the sketch control. I think the go live button should always be in bottom right corner of screen. and
the sketch control can go to the left of it, as current horizontal bar. or it can be merged into the virtual bar.
the latter is a larger change but makes sense: we don't need two ways to pick color, we don't need two ways to show
toggles. the vertical bar can have basically three toggles in 'annotation link': 1 for pointer; 2 for sketch, and 3
for making note. the sketch when turned on, shows more control for this sub mode."

Today the cast chrome has two toolbars: the right-hand vertical palette (colours, pointer, note bubble) and the
bottom horizontal bar (Sketch, colour select, Undo, Redo, Done), plus Go live / Pause above it. That's two colour
pickers and two kinds of toggle.

Design (the operator's preferred "larger" option):
- **One vertical tool bar** with three mode toggles: **pointer** (ephemeral, presenting only, never submitted),
  **sketch**, **note** (bounding-box annotation). Exactly one mode is active.
- **One colour picker** shared by all modes.
- **Sketch's sub-controls** (Undo, Redo, Done/Note) appear attached to the bar only while sketch mode is on.
- **Go live / Pause** is fixed at the **bottom-right corner**, independent of the tool bar.
- Keeps pair#753's feedback model (annotation + sketch are the feedback set; pointer never submitted) and pair#805's
  "Pending feedback" panel.

## Done when

- The console and viewer show one vertical bar with the three modes and one colour picker; the bottom horizontal bar
  is gone; Go live / Pause sits bottom right (browser tests plus a native console check).
- Mode switching preserves in-progress ink and notes correctly (tests); staged for an operator look.

## Plan

- [ ]

## Log

### 2026-10-22
