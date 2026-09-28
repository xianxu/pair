---
id: '000114'
status: done
started: 2026-07-10T07:54:48-07:00
created: 2026-07-10
updated: 2026-07-10
estimate_hours: 0.45
actual_hours: 0.15
---

# draft completion should skip outside insert mode

## Problem

Draft-pane typeahead completion is scheduled from `TextChangedI/P`. If the user
leaves Insert mode before a debounced callback runs, the callback can still call
`vim.fn.complete()`. Neovim raises `E785: complete() can only be used in Insert
mode`, currently observed when selecting a word around spell-check/completion
flows.
