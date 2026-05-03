---
id: 000012
status: working
deps: []
created: 2026-05-03
updated: 2026-05-03
---

# copy on select behavior provide a visual cue

current copy on select, the selection stays on, copy is done to clipboard and text inserted into nvim. all good. 

an improvement would be to give user a visual cue some operation happened, a visual feedback. I'm thinking about after user finish selection (mouse up), the selected text blinks for once, about 500ms, then gone. 

is this doable?

## Done when

- After mouse-up selection in the agent pane, the inserted quote in nvim flashes briefly (~500ms) with a clearly visible highlight.
- The first line of the inserted quote is scrolled to the top of the nvim window (so multi-line quotes are visible from their start).
- Cursor lands on the single empty line directly after the quote, in insert mode (no extra blank line).

## Spec

Decision: rather than try to flash the actual selected text in the agent pane (zellij owns selection rendering and exposes no API for region highlight), put the visual cue at the destination — the nvim draft pane — where the user's eyes are heading anyway. The strong visual on insertion doubles as the "something happened" cue and as orientation for where the quote landed.

Three behaviors bundled:

1. **Flash** the inserted block via an extmark/namespace highlight (`IncSearch` group), cleared after 500ms with `vim.defer_fn`.
2. **Scroll** so the first line of the inserted block is the top visible line (`zt`).
3. **Cursor** on the single empty line immediately below the block, in insert mode.

To do all three cleanly we need to know the line range of the inserted block — easier in nvim than in shell. So move the injection from "type characters into insert mode via `zellij action write-chars`" to "trigger an nvim Lua function that reads the quote from a temp file and inserts it via the buffer API."

Frame-flash in zellij and visual-bell-via-outer-TTY were considered and rejected: zellij has no per-pane frame styling action, and BEL flashes the entire host window (too coarse).

## Plan

- [x] Add `PairPasteQuote()` global function to `nvim/init.lua`:
  - read quoted body from `$PAIR_DATA_DIR/quote-<tag>`
  - insert as buffer lines (replace if buffer is a single empty line; else append after current row)
  - cursor on a fresh empty line directly after the block
  - `zt` to scroll first inserted line to top
  - flash inserted range via `nvim_buf_set_extmark` with `IncSearch` hl, cleared after 500ms
  - `startinsert`
- [x] Rewire `bin/clipboard-to-pane.sh`:
  - keep clipboard read + par reflow + `> ` prefix
  - write quoted body to `$PAIR_DATA_DIR/quote-<tag>`
  - focus nvim, then `Ctrl-\ Ctrl-N` + `:lua PairPasteQuote()<CR>` instead of typing the body
- [x] Update `atlas/architecture.md` (clipboard-to-pane.sh and init.lua sections).

## Log

### 2026-05-03

Started work. Approach: nvim-side injection via `PairPasteQuote()` Lua function triggered from shell, eliminates the keystroke-replay path and gives us line-range knowledge for flash + scroll + cursor positioning in one place.

