---
id: 000231
status: open
created: 2026-09-11
updated: 2026-09-11
estimate_hours:
github_issue:
---

# A digital clock at the right end of couch's status bar

## Problem

Couch's status bar uses its left side for thread chips and notices, and the
right end is usually empty. The operator asked for a digital clock there, which
puts that space to use: 24-hour format, updating every second.

What the bar does today, from the code:
- `RenderStatusRow(width, StatusModel)` (`couchtty/reserve.go`) draws left to
  right: the actor chips, then the notice, each clipped at the row width.
  **Nothing is aligned to the right edge**; the row just ends where its content
  ends.
- **The chips are click targets.** Each one records its column range as a
  `ChipSpan`, and a mouse click maps a column back to an actor
  (`RenderedStatusRow.ColumnToActor`).
- **Couch repaints on events, not on a clock.** The only periodic tick,
  `MenuEventTick`, drives the progress spinner and runs only while a progress
  notice is showing. A clock that changes every second needs a repaint every
  second.
- The bar shares its style with pair's tab strip (#225).
