---
id: 000779
status: open
deps: []
github_issue:
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
collaboration-mode: collaboration
card_mirror: '66dc0ca27dbf542c4ae724ae8ef4aeaae0aeae9b' # card fields mirrored from issue-cards; edit via sdlc
---

# Couch tab bar: wrap onto a second line when it overflows, instead of hiding slots behind +N

## Problem

## Spec

Operator report (10-21 13:08, ops session file item 12, screenshot `ops/assets/session/2026-10-21.13-08-21.150.jpg`):
Couch's tab bar ran out of space. With ops, ariadne, blog, pair, parley, you-decide, ops-echo and echo groups and their
slots, the row overflows and the tail collapses into "+3", so slots are hidden.

Wrap the tab bar onto a **second line** when it doesn't fit (like text wrapping), instead of truncating to "+N":
- Wrap at group boundaries where possible (keep a repo's slots together); break inside a group only if one group alone
  is wider than the line.
- At most two lines; if two lines still don't fit, keep today's "+N" overflow on the second line's end.
- The Chrome/cast status group stays where it is (left of line one).
- The rest of the layout (panes, footer) shrinks by the extra line only when the second line is in use.
- Resizing the terminal re-wraps without flicker; the active tab stays visible.

## Done when

- Render tests: fits on one line (unchanged), needs two lines (wraps at a group boundary), one oversized group, and
  overflow beyond two lines (+N), at several widths; active tab always visible.
- Pane geometry accounts for the second line (no overlap), with a test.

## Plan

- [ ]

## Log

### 2026-10-21
