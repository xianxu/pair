---
id: 000386
status: open
deps: []
github_issue:
created: 2026-10-02
updated: 2026-10-02
estimate_hours:
card_mirror: 'ee425ac4b0d3407923ea859d90575ba90ebf72e8' # card fields mirrored from issue-cards; edit via sdlc
---

# Switcher focus view: no default selection when current slot is hidden

## Problem

When ctrl-space opens the Couch switcher in the **focus view** (pair#372) and the
slot being left (or the one paging) is not among the focus-view rows, the switcher
still selects something. `onHotkey` (`cmd/internal/couchtty/console.go`) routes
through `reconcileRootSelection` (`cmd/internal/couchtty/menu.go`), which falls
back to "the first visible row" when the preferred address isn't visible. That
puts the cursor on an arbitrary slot that the operator never chose, so an Enter
pressed from habit jumps somewhere unintended.

## Spec

- Ctrl-space opens the switcher in the focus view, and the preferred address (the
  pager, else the current slot) is not a visible focus-view row → **nothing is
  selected**: no row is highlighted, and Enter does not switch to anything.
- From that empty selection:
  - the first **Down** selects the first selectable row;
  - the first **Up** selects the last selectable row.
- After that, Up and Down behave as they do today, clamped at the ends.
- Unchanged: the normal view, a preferred slot that is visible in the focus view,
  and the existing fallback in other reconcile paths (refresh, filter edits).
  Decide in the design whether an empty selection should survive a background
  inventory refresh. The likely answer is yes, until the operator moves.
- `moveRootSelection` currently treats "no match" as index 0, so the first Up from
  an empty selection would land on row 0. That has to become an explicit
  empty-selection state.

## Done when

- Unit tests in `couchtty` cover the following cases. Ctrl-space in the focus view
  with the current slot hidden leaves no row selected. The first Down from that
  state selects the first row, and the first Up selects the last. Enter with no
  selection is a no-op. With the current slot visible, it is still preselected.
  The normal view's behavior is unchanged.
- An operator smoke test on a live Couch passes on pair:0 after `make build`.

## Plan

- [ ]

## Log

### 2026-10-02
- Filed. Code anchors: `onHotkey` (console.go), `reconcileRootSelection`,
  `moveRootSelection`, `selectableRootRows` (menu.go).
