---
id: 000303
status: open
deps: []
github_issue:
created: 2026-09-20
updated: 2026-09-20
estimate_hours:
---

# Completion popup mispositioned when the token wraps across screen rows

## Problem

Operator report, 2026-09-20, with a screenshot of the draft pane: the path
completion popup appears in the wrong place. The operator's reading: *"it seems
to be anchoring at the end of the line, but when the line wraps, the position of
the autocomplete is wrong."*

**What the screenshot shows** (draft pane in the brain thread, `wrap` on):

```
make a ticket in parley the chat file slug generation seems inconsistent. for example ../
parli/workshop/parley/█                                            ┌ ../parli/worksh…
                                                                   │ ../parli/worksh…
```

- The token being completed is `../parli/workshop/parley/`. The line wrapped
  **inside** it: row 1 ends with `../`, row 2 is `parli/workshop/parley/` with
  the cursor at its end.
- Measured off the image (18 px per column): the pane is ≈ 97 columns wide; row 1
  is 89 characters, so the token starts at column ≈ 86 of row 1; the cursor is at
  column 22 of row 2.
- The popup's left edge is at column ≈ 78, one row below the cursor, and it runs
  into the pane's right edge: each entry is clipped to ≈ 15 of its ≈ 30
  characters (`../parli/worksh`), with the neighbouring pane's text showing
  immediately after. It is not under the cursor (column 22) and not under the
  token's start.

**The anchor is the token's start, not the end of the line.** The completers pass
the token's 1-indexed start byte to `vim.fn.complete(startcol, items)`
(`nvim/init.lua:1667` `path_complete` → `_G.PairCompleteProbe.sink`, which is
`vim.fn.complete`; `word_complete` and `spell_complete` do the same). Neovim
positions its native popup from that column. That is fine while the token sits on
the cursor's own screen row, and is exactly what breaks when the token's start is
on an *earlier* screen row than the cursor — the shape in the screenshot.

**Why this shape is common here rather than rare.** The draft sets
`wrap`, `linebreak`, `breakindent` (`nvim/init.lua:257-259`). `linebreak` breaks
at the `breakat` characters, whose default set includes `/`, `.` and `-`. Those are
precisely the characters a path token is made of, so a long draft line is likely
to wrap in the middle of a path. The path completer is the feature most exposed to
this, and it is also the one that fires on `../` and `~/` prefixes, which is what
people type when pointing the agent at another repo.

## Spec

Not designed. What is known and what is not:

- **Same class, more than one completer.** Any completion whose replace span
  straddles a wrap boundary is affected. `word_complete` and `spell_complete` reach
  the same `complete()` call, so the fix belongs with the shared sink, not in
  `path_complete`.
- **`complete()`'s start column does two jobs.** It selects the text the pick
  replaces *and* anchors the popup. Moving the anchor to the cursor's row without
  changing the replace span is therefore not a one-argument change; whichever
  direction is chosen has to keep the replacement correct (CompleteDone and
  `<C-y>` substitute using the span passed in, `nvim/init.lua:3789`).
- **The popup must never be clipped by the pane edge.** Even before position is
  settled, a popup whose entries are cut at the pane's right edge is a defect
  on its own; the observed clipping suggests the width was not accounted for at
  the placement column.
- **Not established:** whether the misplacement is Neovim's own placement for a
  span that starts on a previous screen row (which would make it a Neovim
  behavior to work around), or something in how pair drives it. The first thing
  the fix needs is an oracle that can say where the popup went.

## Done when

- With a path token that wraps mid-token, the popup appears at a position related
  to where the operator is typing (under the cursor's row, inside the pane), and
  every entry is fully visible up to the popup's width cap. The oracle reads the
  popup's actual screen position, not the `complete()` argument.
- The same holds for `word_complete` and `spell_complete` with a token that wraps.
- A regression test drives a real Neovim through the completion chain with a draft
  narrow enough to wrap inside the token, and asserts the popup's row/column
  relative to the cursor. A test that only checks the `startcol` handed to
  `complete()` cannot catch this: that value is correct.
- The unwrapped case (token wholly on the cursor's row) is unchanged.

## Plan

- [ ]

## Log

### 2026-09-20

Filed from an operator report with a screenshot. Read the completers; did not
change anything.

**A reproduction attempt did not get a result, recorded so it is not repeated
blind.** The plan was to read the popup's position from Neovim itself through
`vim.ui_attach(ns, {ext_popupmenu = true})`, whose `popupmenu_show` event carries
the row/col Neovim chose, for three shapes: token wholly on the first row, token
straddling the wrap, token wholly on the second row. Three approaches to getting
into Insert mode under a headless run were tried and none produced a reading:

- `nvim --headless -l script.lua` never reaches Insert mode, so
  `vim.fn.complete()` raises `E785: complete() can only be used in Insert mode`,
  whether Insert is requested with `startinsert`, `startinsert!` or
  `nvim_input('A')` (an `-l` script runs without the main loop consuming input).
- `nvim --headless -c 'luafile …'` with the work moved into an `InsertEnter`
  autocmd never fired it, and the process waited forever until killed.

What is likely to work is what pair already has for terminal behavior: a real pty
(pair's terminal tests allocate one) or an `--embed` msgpack-RPC client that
attaches a UI, so a genuine event loop is running when `complete()` is called.
That is also the test shape the Done-when needs, so it is worth building once for
both.
