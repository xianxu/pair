---
id: 000012
status: done
deps: []
created: 2026-05-03
updated: 2026-05-03
---

# copy on select behavior provide a visual cue

current copy on select, the selection stays on, copy is done to clipboard and text inserted into nvim. all good. 

an improvement would be to give user a visual cue some operation happened, a visual feedback. I'm thinking about after user finish selection (mouse up), the selected text blinks for once, about 500ms, then gone. 

is this doable?

## Done when

- After mouse-up selection in the agent pane, the inserted text in nvim flashes briefly (~500ms) with a clearly visible highlight.
- Insert behavior is conditional on cursor column:
  - **col == 0** (cursor at start of a line): treat selection as a fresh quote block — par reflow + `> ` prefix every line, scroll first inserted line to top, cursor lands on the single empty line directly after the block in insert mode.
  - **col > 0** (cursor mid-text): paste verbatim at the cursor — no reflow, no prefix, no viewport scroll, cursor right after the inserted span in insert mode.

## Spec

Decision: rather than try to flash the actual selected text in the agent pane (zellij owns selection rendering and exposes no API for region highlight), put the visual cue at the destination — the nvim draft pane — where the user's eyes are heading anyway. The strong visual on insertion doubles as the "something happened" cue and as orientation for where the text landed.

The insert behavior is split based on cursor column when the copy_command fires:

**Quote-mode (`col == 0`)** — the user is at the start of a line, so the selection is a fresh block.
- Reflow with `par 1000`.
- Prefix every line with `> `.
- If the cursor's line is empty, replace it; otherwise insert above it (the existing line slides down).
- After insert: scroll first inserted line to top via `zt`; cursor on the single empty line directly below the block; insert mode.
- Flash the quoted lines (full-line highlight via `nvim_buf_add_highlight` per line) for 500ms.

**Inline-mode (`col > 0`)** — the user is mid-text, just stitching a word/phrase in.
- No reflow, no `> ` prefix.
- Insert verbatim at the cursor via `nvim_buf_set_text` (handles multi-line splits cleanly).
- Cursor lands at the end of the inserted span; insert mode.
- No viewport scroll.
- Flash the inserted span (extmark with `end_row`/`end_col` so multi-line works) for 500ms.

To do this cleanly we need the line/col range of the inserted text — easier in nvim than in shell. So the shell hands off the *raw* clipboard via `$PAIR_DATA_DIR/quote-<tag>` and triggers `:lua PairPasteQuote()<CR>`. par reflow and quote prefixing now live in `PairPasteQuote`.

Frame-flash in zellij and visual-bell-via-outer-TTY were considered and rejected: zellij has no per-pane frame styling action, and BEL flashes the entire host window (too coarse). `set-pane-color` exists but only affects the pane's terminal-default bg/fg; many TUIs paint their own bg on every redraw, masking it.

## Plan

- [x] Add `PairPasteQuote()` global function to `nvim/init.lua` (initial version: always quote-mode).
- [x] Rewire `bin/clipboard-to-pane.sh` to hand off via temp file + `:lua PairPasteQuote()<CR>`.
- [x] Move par reflow + `> ` prefix from shell into nvim.
- [x] Split `PairPasteQuote()` into `paste_as_quote` (col==0) and `paste_inline` (col>0) branches.
- [x] Update `atlas/architecture.md` (clipboard-to-pane.sh and init.lua sections).

## Log

### 2026-05-03

Started work. Approach: nvim-side injection via `PairPasteQuote()` Lua function triggered from shell, eliminates the keystroke-replay path and gives us line-range knowledge for flash + scroll + cursor positioning in one place.

Implemented:
- `nvim/init.lua` — added `PairPasteQuote()` global with the three behaviors.
- `bin/clipboard-to-pane.sh` — replaced the `i`/write-chars body injection with a hand-off via `$PAIR_DATA_DIR/quote-<tag>` + `:lua PairPasteQuote()<CR>`.
- `atlas/architecture.md` — updated clipboard-to-pane and nvim init sections, and added `quote-<tag>` to data layout.

Verified in headless nvim (TMPDIR-aware test):
- Empty buffer case: inserted lines start at row 1, no leading blank, cursor on the empty line directly below.
- Non-empty buffer case (cursor on row 2): block inserted at rows 3+, `topline=3` confirms `zt` placed the first inserted line at the top of the window, cursor on row 5 (single empty line after block).
- Mode reports `n` in headless because `startinsert` is deferred to the next event-loop tick — interactive nvim will land in insert.

Manual verification still pending: actual flash visibility under the running tty (the headless test sets the namespace + extmarks but you can't see them without a real terminal). Will eyeball next time I select text in the agent pane.

Status moved to working at start; will move to done after manual eyeball check.

#### Iteration: source-side flash via set-pane-color

Added a complementary visual cue at the *source* end (the agent pane) by flashing its bg via `zellij action set-pane-color --bg <color>` in `bin/copy-on-select.sh`, with a backgrounded delayed `--reset`. Configurable via `$PAIR_FLASH_BG` (default `#5a4a00`) and `$PAIR_FLASH_MS` (default 500). Best-effort — `set-pane-color` only affects the pane's default bg, so cells claude is actively painting won't change; visibility depends on how much of claude's content uses transparent/default bg.

The nvim-side flash on the inserted text remains the primary cue; this is additive.

Eyeballed in a real session: visible on claude. Tuned duration to 250ms then 100ms — at 100ms it reads as a quick "fired" pulse without lingering, which felt right.

#### Iteration: cursor-conditional paste mode

Refactored `PairPasteQuote` to dispatch on cursor column:
- `col == 0` → existing quote-mode path (par reflow + `> ` prefix + zt + flash + cursor on empty line below).
- `col > 0` → new `paste_inline` path: `nvim_buf_set_text` at cursor, no reflow, no prefix, no scroll; cursor lands at end of inserted span in insert mode; flash via single multi-line extmark.

Moved par + `> ` prefix from shell into nvim — shell now hands off the raw clipboard. Cleaner separation: shell is a transport, nvim owns formatting decisions.

Verified four cases in headless nvim (TMPDIR-aware, with explicit cursor setup):
1. Empty buffer + cursor (1,0) → `> joined-by-par`, cursor on row 2, `topline=1`. ✓
2. Cursor at (1,5) of "hello world" + paste "PASTED" → `helloPASTED world`, cursor at (1,11). ✓
3. Cursor at (2,4) of "beta gamma" + paste "foo\nbar\nbaz" → splits cleanly: `betafoo`, `bar`, `baz gamma`; cursor at (4,3). ✓
4. SOL of non-blank line (2,0) → quote inserted before it, existing content slides down to row 4; `topline=2`. ✓

Defensive clamp: clamp cursor row to ≥1 because uninitialized headless nvim can return row=0; harmless on real instances.

