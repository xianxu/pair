---
id: 000639
status: working
deps: []
github_issue:
created: 2026-10-16
updated: 2026-10-17
estimate_hours:
collaboration-mode: collaboration
card_mirror: '79bf6dc640bfea3e036d795e2cd01f06c9e74d57' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-17T00:09:57-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
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

- [ ] Make planner/runtime tests require Focus → Normal and retain native-fullscreen exit coverage; observe red.
- [ ] Reuse existing restore/focus/clear operations for Focus exit; update source-derived help, README, atlas and conformance scenarios.
- [ ] Run affected packages and native chord conformance, coordinate real-setup smoke with hub, then recorded full suite, close and land.

## Log

### 2026-10-16

### 2026-10-17 — quick design

TL dispatched the operator’s explicit two-state contract and authorized quick flow. ARCH-PURE/ARCH-DRY: retain PlanRightPane as the one transition owner and reuse its existing operations. Focus exit re-embeds, restores the recorded tiling and split order, returns to the invoking pane, clears the return record and nudges dimming. Native fullscreen remains observable only for the one-press compatibility exit; no plan enters it. Existing lock, stop-on-error and degraded-restore behavior remain. No new durable artifacts, external calls or background work. Existing runtime fake and native chord/conformance tests cover single and split panes, invoking-focus restoration, failure ordering and native fullscreen. The fixed 75% centered geometry is unchanged. Plan posted to hub, proceeding unless objected.
