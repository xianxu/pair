---
id: 000018
status: open
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

- [ ] `nvim/scrollback.lua` — `Alt+q` normal-mode keymap: prompt
      for comment via `vim.fn.input`, insert `🤖[<comment>]` at end
      of current line. Lift `modifiable=false` for the insert.
- [ ] `nvim/scrollback.lua` — `Alt+q` visual-mode keymap: get the
      selection (single line), prompt, replace with `🤖<sel>[<comment>]`.
- [ ] Verify (headless): drop one of each kind of marker, dump
      buffer, check both forms appear in expected positions.

### M2: extraction on viewer exit

- [ ] Marker parser: walk a line by index, find every `🤖`, peek
      at the next byte to discriminate scoped vs bare, record
      `{kind, X?, Y, byte_range}` per match.
- [ ] Strip-all-markers helper for the bare-form context: removes
      every marker from a line back-to-front (preserves earlier
      ranges) then trims whitespace.
- [ ] `VimLeavePre` autocmd: walk buffer, find markers, format,
      write to sidecar. Skip writing if no markers (no-op exit
      should be silent).
- [ ] Verify (headless): synthetic buffer with mixed markers,
      assert sidecar contents match expected `> X\nY\n\n` blocks.

### M3: pickup in draft

- [ ] `nvim/init.lua` — `FocusGained` autocmd: check sidecar exists
      and non-empty, read contents, append to draft buffer (separated
      by a blank line if the draft already has content), delete
      sidecar atomically (rename-to-tempfile-then-rm or just
      `os.remove`), `vim.notify` count.
- [ ] Verify (manual): drop markers, exit viewer, focus back on
      draft, see them appended.

### M4: docs + cleanup

- [ ] Update `nvim/scrollback.lua`'s statusline hint to mention Alt+q.
- [ ] Update `bin/pair -h` Alt+/ entry to mention the marker workflow.
- [ ] Atlas: extend the "Colored scrollback dump" section with the
      marker-and-pickup loop.
- [ ] Add to lessons.md if any subtle bugs caught during testing.

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
