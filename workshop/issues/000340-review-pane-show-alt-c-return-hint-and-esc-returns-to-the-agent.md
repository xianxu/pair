---
id: 000340
status: working
deps: []
github_issue:
created: 2026-09-28
updated: 2026-09-28
estimate_hours:
card_mirror: '0e09bc509411578aa6f50588394de785a997e1f0' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-09-28T15:14:24-07:00
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
- **Document the review buffer's own keys.** The operator also forgot the
  accept/reject keys. They exist (`nvim/review.lua`, buffer-local, normal mode)
  but appear in neither README's key table nor the Alt+h help page:
  - `Alt+a` / `Alt+r`: accept / reject the 🤖 suggestion at the cursor
    (also `<leader>a` / `<leader>r`)
  - `Alt+Shift+A` / `Alt+Shift+R`: accept / reject every 🤖 suggestion in the
    paragraph, up to the cursor
  - `Alt+q`: insert a human comment marker (visual mode: quote the selection)
  - `Alt+Enter`: finish the human turn
  - `]m` / `[m`: next / previous marker
  The full list goes in the Alt+h help (`pair-help`, `cmd/internal/keyscmd`)
  and README's key table, derived from the keymaps' `desc` fields where
  practical, so the list can't drift from the mappings. The bottom bar is
  narrow, so it carries only the most-needed hints: the way out, plus
  `Alt+a/r accept/reject`. Width budget and wording are to settle at design time.
- **Alt+n / Alt+Shift+N step through markers** (operator request): in the review
  buffer, Alt+n does `]m` (next 🤖 marker) and Alt+Shift+N does `[m` (previous),
  both wrapping, both buffer-local, normal mode.
  - Alt+n is free here. Pair's Alt+n reload is draft-only since #333, and the
    review pane installs only non-draft globals
    (`workbench_route.install_global_maps(false)`).
  - Alt+Shift+N is the global "restart the agent conversation" shortcut, and
    the review pane installs it. The buffer-local mapping overrides it in review
    only (the agent restart stays reachable from the draft). The Alt+h help must
    say so: the row for Alt+Shift+N names the review exception.
  - Check under Couch that the review pane actually receives Alt+n
    (Couch's routing may replace Pair's, #284), and with Pair alone.
  - `]m` stops at every 🤖 marker, the operator's own `[H]` comments included.
    Skipping to agent proposals only (`pending` in `review/markers.lua`) is a
    possible refinement, not asked for.
- Update help/README/atlas prose for the review mode's keys (lessons: UI text
  is a public contract).

## Done when

- The review pane's bottom bar shows the return key (and the accept/reject
  hint) in the idle and awaiting states.
- Alt+h help and README list the review buffer's keys (accept/reject,
  paragraph accept/reject, comment marker, finish turn, marker jumps). A test
  fails if a review keymap is added without a help entry.
- Normal-mode Esc in the review pane hides it and leaves focus on the agent
  pane. Insert-mode Esc still just leaves insert mode. A test covers both,
  and fails if the mapping is reverted.
- Alt+n / Alt+Shift+N in the review buffer move to the next / previous marker,
  under both Pair alone and Couch. Alt+Shift+N still restarts the agent from
  the draft pane.
- Live smoke by the operator: open review with Alt+c, read the hint, Esc back to the agent.

## Plan

- [ ]

## Log

### 2026-09-28

- Filed from operator feedback (screenshot of the review bar with no exit hint).
  They found Alt+c by guessing.
- Added: Alt+n / Alt+Shift+N → `]m` / `[m` in the review buffer. Operator's
  choice; the Alt+Shift+N override of the global agent restart is noted in Spec.
  Operator confirmed: while the review pane has focus, Alt+Shift+N means `[m`
  (previous marker). Everywhere else it keeps the agent restart.
- Added: document Alt+a / Alt+r (and the rest of the review keys). The
  operator had forgotten those too.
