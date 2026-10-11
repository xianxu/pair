---
id: 000449
status: codecomplete
deps: []
github_issue:
created: 2026-10-11
updated: 2026-10-11
estimate_hours:
card_mirror: '5b8d74c1b8647a27860ba10136d2b5596c2ad693' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-11T12:41:18-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "aa93a095", done: "584677b8"}
actual_hours: 0.07
---

# Broadcast viewer: pointer color swatches as a vertical overlay on the screen's right edge

## Problem

pair#446 put the pointer color swatches in a band below the terminal screen. The operator (10-11, after a live smoke test that worked) wants them overlaid on the terminal screen itself, at the right edge, vertically centered, in a vertical column.

## Spec

- The six swatches render as a vertical column overlaid on the terminal stage (not the page), at its right edge, centered vertically on the stage's height; they track the stage as it resizes.
- Keep it unobtrusive: small swatches, a translucent backing, and it never takes pointer events outside the swatches themselves (a point or drag over the rest of the screen still reaches the pointer).
- The selected swatch stays marked; the pick is still remembered per browser.
- Phone width: still usable (the column can shrink); no horizontal page scroll.
- Leave room for pair#447's mode icon (pointer vs comment) in the same column.

## Done when

- On a live broadcast the column sits at the stage's right edge, vertically centered, at desktop and phone widths (operator smoke test; headless check at 390px).
- Pointing over the rest of the screen works as before (viewer test).

## Plan

- [x] Move the swatch band into a stage overlay; CSS for position, size and translucency; viewer tests; headless width checks.

## Log

### 2026-10-11
- 2026-10-11: closed — Post-close fix for advisory 2: #rail gets .stale with #stage (setStale), same CSS rule; node --test tests/broadcast-viewer 39/39 incl. new stale assertion. (fixed-then-ship); review verdict: SHIP
- 2026-10-11: closed — node --test tests/broadcast-viewer 38/38 incl. new rail.test.mjs (railAt dock math; rail outside #stage; pointer-events none on rail, auto on swatches); go test ./cmd/internal/broadcast ok; headless Chrome w/ fake event stream at 1280x720, 390x844, 844x390: column on screen's right edge, vertically centered, no overflow. make -k test: only test-review's 8 nvim failures, identical on origin/main; go test ./... failures (couchcmd continuation, launcher checkpoint/native) identical on origin/main. Operator live smoke test pending a Couch restart.; review verdict: SHIP

- Design: the swatches move into `#rail`, a fixed column outside `#stage` that `fit()` docks to the drawn `.xterm-screen` rect (`railAt`, pure, tested). Outside the stage, a swatch tap never reaches the stage's pointerdown (no stroke); `pointer-events: none` on the rail with `auto` on the buttons lets a point over the translucent backing fall through to the screen. #447's mode icon becomes another rail child. The hint keeps the bottom band. This deliberately reverses #446's lesson (controls covering cells), which the operator asked for: small, translucent, edge-docked.
- Verified: `node --test tests/broadcast-viewer/*.test.mjs` 38/38 (new rail.test.mjs). Headless Chrome with a fake event stream in 1280×720, 390×844 and 844×390 iframes: the column sits on the screen's right edge, vertically centered, with 14px swatches at phone width and no overflow.
- Post-close (advisory 2, real): the rail sat outside `#stage`, so a stale screen dimmed but its swatches stayed bright. `setStale` now marks `#rail` too, under the same rule. Advisory 1 (the pass-through clause is pinned structurally) stands: node can't hit-test, and headless Chrome checked the layout.
