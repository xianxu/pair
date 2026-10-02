---
id: 000369
status: open
deps: []
github_issue:
created: 2026-10-01
updated: 2026-10-01
estimate_hours:
card_mirror: 'b01318fc41fef90aeeaedd1fedc0d82462d300db' # card fields mirrored from issue-cards; edit via sdlc
---

# couch switcher: click selects row, double-click jumps

## Problem

In the couch switcher (the panel menu), a single left click on a selectable
row jumps straight to that thread. Clicking can't be used to just point at a
row, so it doesn't fit how the keyboard cursor works (move the cursor, then
Return).

Today's path: `couchtty/terminal_input.go` (panel branch) →
`RenderedMenu.PointToRow` → `MenuEventMouseSwitch` → `couchtty/menu.go`
reducer, which calls `selectMenuRow` and then `dispatchMenuRow` in one step.

## Spec

- **Single click** on a selectable row **selects** it: the menu cursor moves
  to that row and nothing is dispatched. After that, keyboard **Return** acts
  on the row exactly as it would after arrow-key navigation (same
  `enterOperationFor` dispatch).
- **Double click** on a selectable row **jumps** to it, whether or not it was
  already selected: the same dispatch today's click does, still marked
  `InFlight.Manual` (so it is pinning and ctrl+backspace undoes it).
- Terminals send no double-click event, so couch detects it itself: two
  left presses on the **same row key** within a short window (about 400ms,
  one named constant). The time comes from the console's injected
  clock/now seam, not from `time.Now` inside the reducer.
- Unchanged: pending or unselectable rows ignore both single and double
  click (pair#206). Status-chip clicks on the bottom row still jump in one
  click; this issue only covers the switcher rows.
- A click on a different row, or one after the window has passed, starts
  a new click sequence, so it is a select.

## Done when

- Unit tests in the reducer: a single click selects the row and returns no
  effects; a double click within the window on the same row dispatches the
  switch with `Manual` set; a second click on a *different* row, or after
  the window, only selects; Return after a click-select dispatches the
  selected row; clicks on pending rows do nothing.
- Operator smoke test in a live couch (after `make build`): click selects
  the row and highlights it, Return jumps, double-click jumps.

## Plan

- [ ] Split `MenuEventMouseSwitch` into a select event and a jump event, or
      give it a click count; keep the reducer pure (the caller supplies the
      click count or timestamp)
- [ ] Track the last click (row key + time) in Console; classify single vs
      double click in `terminal_input.go`
- [ ] Tests from Done when; `make test`
- [ ] Update the click description in `atlas/couch.md`

## Log

### 2026-10-01
