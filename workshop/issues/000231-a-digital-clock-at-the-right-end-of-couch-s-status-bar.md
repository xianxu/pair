---
id: 000231
status: open
deps: []
github_issue:
target: workbench-latency
created: 2026-09-11
updated: 2026-09-11
estimate_hours:
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

## Spec

- **`HH:MM:SS`, 24-hour, local time**, right-aligned at the end of the status
  row, flipping on each wall-clock second.
- **The clock uses only space that is otherwise empty.** Chips and the notice
  keep their columns. The clock never shifts a chip, so every `ChipSpan` and
  click target stays where it is.
- **When the row is too full, the clock gives way, not the chips.** Chips are
  functional and the clock is informational, so the clock is dropped whole
  rather than truncated or overlapping. This is a proposed default for the
  operator to confirm.
- **A 1 Hz repaint of the status row only**, and it must stay off the keystroke
  path.
  - It repaints only that one row, never the agent pane.
  - It coalesces with repaints that are already pending.
  - It stops when the console stops.
- **The pure part stays pure.** `RenderStatusRow` receives the time as data,
  and the tick source is injected, so the render is table-testable without real
  time (`ARCH-PURE`). Couch has no fake clock today (`Feed` and `showMenu`
  both call `time.Now`), so this adds the seam rather than reading the wall
  clock inside the renderer.

## Traps to check first, not after

- **Writing the row's final column (#223).** A right-aligned clock ends in the
  last column. pair#223's root cause was zellij's autowrap escaping the scroll
  region when that column is written, so find out whether this row is exposed
  before choosing the alignment. If it is, stop one column short.
- **Repaint interaction with the #209 nudge and the scroll region.** A periodic
  write to one row must not displace the screen the way the repaint nudge does
  mid-transition.
- **Idle cost.** A 1 Hz timer wakes couch every second even while nothing is
  happening. That is cheap, but it is new, and `workbench-latency` means it gets
  measured, not assumed.

## Done when

- The clock shows `HH:MM:SS` in local 24-hour time at the right end of the
  status row, updating each second.
- Table tests cover the render over a range of widths:
  - the clock appears when there is room;
  - it is dropped whole when chips and the notice fill the row;
  - chip `ChipSpan` columns are identical with and without it, so click
    targets never move.
- The tick is injected, and a test drives it without real time.
- `zellij action` latency and keystroke latency while the clock runs match the
  quiet baseline, measured with co-tenancy recorded.
- The #223 final-column question has an answer in the Log, with evidence.
- The operator confirms it looks right on the real stack.

## Plan

- [ ] Check whether writing the status row's last column triggers zellij's
      autowrap (#223), and record the answer.
- [ ] Pure: give `RenderStatusRow` a right-aligned clock slot, with width
      tests and a chip-span invariance test.
- [ ] Inject the 1 Hz tick on the console's Run loop, aligned to the second,
      coalesced, and stopped on `Stop`.
- [ ] Measure latency with the clock running; operator smoke.

## Log

### 2026-09-11

Operator request, made while #206 M2 was under way: *"display a digital clock
at the right end of couch's status bar ... progress at seconds level, 24 hour
format ... this would allow us to use some of the screen estate of that bar
towards the right end."* Filed, not started. The status-row facts above come
from reading `couchtty/reserve.go` and `console.go`.
