---
id: 000333
status: working
deps: []
github_issue:
created: 2026-09-26
updated: 2026-09-26
estimate_hours:
started: 2026-09-26T16:51:02-07:00
flow: {kind: quick, provenance: inferred, spec: "a7856128", done: "82d008d4"}
---

# Pass Option+n through to right-terminal TUIs

## Problem

When the right terminal pane is focused on a TUI program, Pair currently
intercepts `Option+n` as its own reload action. This prevents TUIs that bind
that key from receiving it.

## Spec

- In the focused right-terminal TUI path, pass `Option+n` through unchanged to
  the TUI program.
- Keep current Pair handling for the right terminal's shell/non-TUI state and
  for every other pane.
- Reuse the existing TUI detection and input-routing boundary; do not add a
  second independent classification.

## Done when

- Right-terminal TUIs receive `Option+n`.
- Right-terminal shells/non-TUI states retain Pair's existing `Option+n`
  behavior.
- Other panes retain their existing `Option+n` behavior.
- Routing regressions cover the TUI passthrough and neighboring retained
  behaviors.
- User-facing keybinding documentation reflects the focused right-terminal
  exception.

## Plan

- [ ] Locate the existing focused-pane and TUI input-routing decision.
- [ ] Route `Option+n` through only for a focused right-terminal TUI.
- [ ] Add routing regressions for TUI passthrough and non-TUI/other-pane
  behavior.
- [ ] Update the keybinding documentation.

## Log

### 2026-09-26

- Approved design: right-terminal TUI programs receive `Option+n`; shell and
  all other pane paths retain their existing Pair behavior. Reuse the existing
  TUI routing boundary and cover it with focused regressions.
