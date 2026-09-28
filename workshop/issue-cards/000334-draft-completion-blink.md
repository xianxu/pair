---
id: 000334
status: open
created: 2026-09-27
updated: 2026-09-27
estimate_hours:
github_issue:
---

# draft nvim: audit as-you-type completion; evaluate blink.cmp

## Problem

The draft nvim (`nvim/init.lua`) does as-you-type completion by hand, with no
plugins: ~350 lines across `path_complete` (~L1667), `word_complete` (~L1840),
`spell_suggest_popup` (z=, ~L1919), the `TextChangedI`/`TextChangedP`
dispatcher (~L3890), and helpers (`plain_items`/`indexed_items`,
`spell_popup_active`, `spell_pick_digit`, `<M-1>..<M-9>` quick-pick). It feeds
`vim.fn.complete()` directly and filters with `matchfuzzy` (paths) or
prefix-anchored scoring (words).

parley.nvim is adopting blink.cmp (app cmdline completion, then moving the
plugin's own completion from nvim-cmp to blink, including spelling). This is
a good time to check whether pair's hand-rolled completers should move to
blink sources, or stay as they are.
