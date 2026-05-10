---
id: 000018
status: working
deps: [000017]
created: 2026-05-09
updated: 2026-05-09
---

# 🤖[] comment markers in scrollback viewer → draft

## Problem

#000017 added the scrollback viewer. While reading scrollback, the
user often wants to capture follow-up notes — "check this", "fix
that line", "this looks wrong". Today they'd have to switch back to
the draft, type a prompt, lose the visual context.

The parley.nvim convention (`/xx-fix` skill) already defines a marker
syntax that claude understands:

- `🤖<X>[Y]` — scoped human comment about quoted text X
- `🤖[Y]` — bare human comment

Use it. Bind `Alt+q` in the scrollback viewer to drop a marker. On
`:q`, extract markers, format as `> <quote>\n<comment>\n\n`, append
to the draft. User comes back to the draft, sees the formatted
context, reviews, sends with `Alt+Return`.

## Spec

**Creation (in `nvim/scrollback.lua`):**

- `Alt+q` in **normal mode** — prompt for comment via `vim.fn.input`,
  insert `🤖[<comment>]` at the end of the current line. Buffer is
  read-only; lift `modifiable=false` for the duration of the insert.
- `Alt+q` in **visual mode** — capture the selection text X (single
  line only — multi-line scope is rare in TUI scrollback and would
  bloat the marker), prompt for comment Y, replace the selection
  with `🤖<X>[Y]` (parley's scoped form is self-contained: the quote
  is embedded in the marker, so future extraction doesn't depend on
  surrounding context).

Empty comment cancels the insert (no marker dropped).

**Extraction (on viewer exit):**

`VimLeavePre` autocmd walks the buffer, finds every marker on every
line. For each:

- Scoped (`🤖<X>[Y]`) → `> X\nY\n\n`
- Bare (`🤖[Y]`) on a line whose other content is L → `> <L stripped of all markers>\nY\n\n`

Markers are matched by literal byte sequence (🤖 = `\xf0\x9f\xa4\x96`)
because Lua patterns aren't UTF-8-aware; manual scan keeps semantics
explicit.

**Hand-off (write to a sidecar file, not the draft directly):**

Extracted block is written to `$PAIR_DATA_DIR/scrollback-pending-<tag>.md`
(truncated each pickup). Writing to `draft-<tag>.md` directly would
race against the draft nvim's autoread: if the user had typed
something into the draft after the last autosave, the on-disk file
is stale and nvim won't pick up our append (refuses to reload over
modified-buffer state). The sidecar avoids that.

**Pickup (in `nvim/init.lua`):**

`FocusGained` autocmd checks for `scrollback-pending-<tag>.md`. If
non-empty, append its contents to the end of the draft buffer (with
a leading blank line if the draft has trailing content), delete the
file, schedule a redraw. Brief `vim.notify` confirming the pickup
("pulled N scrollback marker(s)").

## Plan

### M1: marker creation in scrollback viewer

- [x] `nvim/scrollback.lua` — `Alt+q` normal-mode keymap: prompts
      for comment, inserts `🤖[<comment>]` at end of cursor line.
      Lifts `modifiable` AND `readonly` (both must flip; readonly
      alone triggers W10 warning).
- [x] `nvim/scrollback.lua` — `Alt+q` visual-mode keymap: reads
      live `'v'` and `'.'` positions (no need to wait for the
      `'< / '>` marks to settle), single-line check, prompts,
      replaces selection with `🤖<sel>[<comment>]`.
- [x] Verify (headless): keymaps registered (`mode=n` and `mode=x`
      both have `<M-q>`); stubbed input + simulated cursor → buffer
      contains the marker on the right line.

### M2: extraction on viewer exit

- [x] Marker parser (`find_markers_in_line`): walks line by byte
      index, anchors on `🤖` literal-byte sequence
      (`\240\159\164\150`), discriminates scoped vs bare by peeking
      at the next byte, records `{kind, X?, Y, range = {lo, hi}}`.
- [x] `strip_markers` helper: removes markers back-to-front so
      earlier ranges remain valid, trims whitespace. Used to build
      the quote line for bare markers.
- [x] `VimLeavePre` autocmd: locates the decorated buffer, calls
      `format_extraction`, writes to a tempfile + renames to
      `scrollback-pending-<tag>.md` (atomic). Silent no-op when no
      markers.
- [x] Verify (headless): mixed-marker buffer (bare on one line,
      scoped + bare on another, standalone bare on empty-content
      line) → output blocks match spec including the `(no context)`
      fallback for the standalone case. Pure-function tests
      exposed via `_G.PairScrollbackTest`.

### M3: pickup in draft

- [x] `nvim/init.lua` — `FocusGained` autocmd: reads sidecar,
      counts markers (one per `\n> ` prefix), applies via
      `nvim_buf_set_lines` + `:write` when on `*` slot (avoids the
      autoread + checktime sub-second-mtime ambiguity), or
      appends-to-file off-slot (`-N` / `+N`) so next nav-to-`*`
      reads it. Sidecar removed; `vim.notify` confirms count.
- [x] Verify (headless): on-`*` path produces exactly the expected
      buffer lines, draft file matches buffer (autosave consistent),
      sidecar gone. Off-slot path code-reviewed (no test — `nav.pos`
      is locally scoped in init.lua).

### M4: docs + cleanup

- [x] `nvim/scrollback.lua` statusline hint now reads
      "q to quit · Alt+q to drop 🤖[] · :N to jump".
- [x] `bin/pair -h` Alt+/ entry mentions the marker workflow.
- [x] Atlas: "Colored scrollback dump" section extended with the
      marker-and-pickup loop.

## Risks / open questions

- **Multi-line selections.** Spec restricts to single-line. Could
  extend later (mark each line as a separate marker?), but that's
  feature creep.
- **Comment with `]`.** A comment containing `]` would close the
  marker prematurely on extraction. Either escape on insert (e.g.
  replace `]` with `&#93;`), or just trust the user. Going with
  trust for v1; document the limitation.
- **Two viewers open at once.** Unlikely (only one Alt+/ at a time
  via the floating pane), but if it happens both would write to
  the same sidecar. Atomic rename in M2 mitigates.
- **Pickup timing.** FocusGained fires when the user clicks back into
  the draft pane (after the floating viewer dismisses). If they go
  via the agent pane first, the draft FocusGained doesn't fire until
  they Alt+down. Acceptable.

## Log

### 2026-05-09

**M1+M2 done in nvim/scrollback.lua.** Two pure functions exposed
for testing (`find_markers_in_line`, `format_extraction`) plus the
keymap callbacks and the `VimLeavePre` autocmd. Visual-mode keymap
reads live `'v'` / `'.'` instead of waiting for `'< / '>` marks.
Tested with three synthetic cases:

- Bare marker (`agent text 🤖[fix this]`) → `> agent text\nfix this`.
- Scoped marker (`before 🤖<some>[review] after`) → `> some\nreview`.
- Mixed (scoped + bare on same line, plus a standalone bare on
  empty content) → all extracted, standalone falls back to
  `> (no context)` placeholder.

End-to-end via headless nvim with stubbed `vim.fn.input`:
`Alt+q` insertion lands the marker on the right row; `:quitall`
fires `VimLeavePre`, sidecar at `$DATA_DIR/scrollback-pending-test.md`
contains the formatted block.

Bug caught during M1: the read-only buffer triggered `W10: Changing
a readonly file` even with `modifiable = true`, because `readonly`
is a separate flag. Lifting both for the insert and re-locking both
afterward suppresses it.

**M3 done in nvim/init.lua.** First attempt used append-to-file +
`:silent! checktime` and relied on autoread to reload — turned out
to be flaky because the file write happens within sub-second
mtime resolution of the buffer load, and `checktime` doesn't always
detect the change. Switched to direct `nvim_buf_set_lines` + `:write`
when on the `*` slot. Off-slot, append-to-file remains correct
because the slot-load reads from disk on the next nav.

Verified headless: `existing draft` + sidecar with 2 markers →
buffer ends up as 7 lines including separators, draft file matches
buffer, sidecar removed, `vim.notify` "🤖 picked up 2 scrollback
comment(s)".

**M4 done.** Statusline hint, `pair -h` entry, atlas section.
