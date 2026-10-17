---
id: 000639
status: open
deps: []
github_issue:
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'b6db56ff0af160a4f60ea83df7a8b6fbcd115a56' # card fields mirrored from issue-cards; edit via sdlc
---

# Right terminal: Alt+Shift+Return toggles Normal and Focus (75% centered) only; drop Maximize

## Problem

## Spec

Operator, 2026-10-16: "change right pane maximize, from 3 states to just two: left/right even split, and the right pane in center 3/4. remove the actual maximized view. I find the 3/4 width of screen is enough and still give some peek of background activity in the agent pane."

Today Alt+Shift+Return cycles the selected right terminal Normal → Focus → Maximize → Normal (`cmd/internal/layoutcmd/rightpane.go`; README's key table):
- Focus floats it, centered at 75% width and full height, with the agent pane dimmed.
- Maximize makes it native zellij fullscreen.

New behaviour: a two-state toggle, Normal ↔ Focus. Maximize goes away as a target of the cycle.
- Alt+Shift+Return from Focus returns to the tiling, focusing the invoking pane, as Maximize → Normal does today.
- A pane already fullscreen when the cycle runs, left over from a previous build or set natively, still returns to Normal. Don't strand it.
- Update the README key table, the atlas and the key help.
- Keep the split rule: a split acts only on the selected half.

## Done when

- The pure plan tests show Normal → Focus → Normal, and that no plan targets Maximize.
- An existing fullscreen right terminal returns to Normal on one press; a test covers it.
- The live chord conformance test passes on the two-state cycle.
- README, the atlas and the key help describe two states.

## Plan

- [ ]

## Log

### 2026-10-16
