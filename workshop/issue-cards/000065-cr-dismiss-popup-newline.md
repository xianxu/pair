---
id: '000065'
status: done
created: 2026-06-18
updated: 2026-06-18
estimate_hours: 0.5
actual_hours: N/A
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
