---
id: 000303
status: open
created: 2026-09-20
updated: 2026-09-20
estimate_hours:
github_issue:
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
