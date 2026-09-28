---
id: 000340
status: open
deps: []
github_issue:
created: 2026-09-28
updated: 2026-09-28
estimate_hours:
card_mirror: '98a0047bea5812c7902ca12adff304f75d76e66c' # card fields mirrored from issue-cards; edit via sdlc
---

# Review pane: show Alt+c return hint, and Esc returns to the agent

## Problem

With the Alt+c review pane open, nothing on screen says how to get back to the
agent pane. The operator had to guess Alt+c again. The review pane's bottom bar
(`statusline_text()` in `nvim/review.lua`) shows only
`🪄 <mode> • <file>  L<n>/<N>`: no way out.

## Spec

- **Hint in the bottom bar.** Add `Alt+c → agent` (wording to settle) to the
  review statusline, in both its idle and awaiting (spinner) forms. It must
  name the key that actually works. Alt+c is routed to `PairReviewToggle`
  (`nvim/init.lua`), which hides the floating review pane.
- **Esc returns to the agent pane.** In the review buffer's NORMAL mode, Esc
  does what Alt+c does from there: hide the review pane, and focus lands on
  the agent pane. Insert/visual-mode Esc keeps its vim meaning (leave the
  mode). Esc in normal mode is otherwise a no-op in stock vim, so nothing is
  lost. Floats opened from the review (diagnostic float, definition float)
  keep closing on Esc first, before the pane itself hides.
- Check where focus lands after hiding the floating pane. If it doesn't land
  on the agent pane, focus it explicitly by pane ID (ID-based, never relative;
  see lessons).
- Update help/README/atlas prose for the review mode's keys (lessons: UI text
  is a public contract).

## Done when

- The review pane's bottom bar shows the return key in the idle and awaiting states.
- Normal-mode Esc in the review pane hides it and leaves focus on the agent
  pane. Insert-mode Esc still just leaves insert mode. A test covers both,
  and fails if the mapping is reverted.
- Live smoke by the operator: open review with Alt+c, read the hint, Esc back to the agent.

## Plan

- [ ]

## Log

### 2026-09-28

- Filed from operator feedback (screenshot of the review bar with no exit hint).
  They found Alt+c by guessing.
