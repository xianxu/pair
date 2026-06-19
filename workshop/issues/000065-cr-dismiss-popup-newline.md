---
id: 000065
status: working
deps: []
github_issue:
created: 2026-06-18
updated: 2026-06-18
estimate_hours: 0.5
---

# Return key dismisses completion popup + inserts newline when nothing selected

## Problem

In the draft pane, when a completion popup is showing and the user has NOT
Tab-selected an item, pressing Return does nothing useful — the keystroke just
closes the menu and the expected newline never lands. The draft's `<CR>` is
fundamentally "insert a newline" (send is `Alt+CR`), so a swallowed Return is a
real papercut while typing.

Root cause: `nvim/init.lua`'s insert-mode `<CR>` expr map returned a bare `<CR>`
for the popup-visible-but-nothing-selected case:

    return (pum_visible() and pum_has_selection()) and '<C-y>' or '<CR>'

A bare `<CR>` fed while the pum is up only dismisses the menu — the newline is
eaten. This is the well-known nvim gotcha under `completeopt=...,noselect`.

## Spec

`<CR>` in the draft must ALWAYS produce its intended effect:

- popup not visible            → `<CR>` (plain newline) — unchanged.
- popup visible + a selection  → `<C-y>` (accept highlighted item) — unchanged.
- popup visible, no selection  → dismiss the menu AND insert a newline. Treat it
  as "I didn't pick anything; keep what I typed and break the line."

The fix is the canonical idiom: feed `<C-e>` (cancel completion, preserving the
typed text exactly) followed by `<CR>` (now processed as a normal newline).

## Done when

- Return inserts a newline when the popup is up with nothing selected, keeping
  the typed text on screen.
- Tab-select-then-Return still accepts the highlighted item (no regression).
- Return with no popup is an ordinary newline (no regression).
- Regression test asserts the expr output for all three states.

## Plan

- [ ] Extract a pure `cr_keys(visible, has_selection)` decision in `nvim/init.lua`,
      add the `<C-e><CR>` branch, route the `<CR>` map through it, expose it as
      `_G.PairCRKeys` for testing.
- [ ] Add `tests/cr-newline-test.sh` (autopair-style headless driver) asserting
      the three states; wire it into `Makefile.local` (`test-cr` + `test`).
- [ ] Run `make test-cr`; live-dogfood the newline-on-Return in a real session.

## Log

### 2026-06-18

- Confirmed current map at `nvim/init.lua:3301`; `completeopt` is
  `menu,menuone,noinsert,noselect` (`init.lua:443`), so nothing is ever
  auto-selected — the no-selection Return path is the common case, not an edge.
- Behavioral headless test of popup rendering is flaky (pum needs a UI;
  feedkeys timing). Following the repo's autopair-test precedent: assert the
  returned expr-string, trust nvim's `<C-e><CR>` semantics, verify newline live.
