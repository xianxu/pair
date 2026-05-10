---
id: 000017
status: working
deps: []
created: 2026-05-09
updated: 2026-05-09
---

# Colored, line-numbered scrollback dump

## Problem

zellij now renders a frame on the agent pane (#08bf61b) so the
top-right scroll-position indicator (e.g. `500/540`) is visible. The
user can see "I'm at line 880 of scrollback" — but has no way to *jump
back* to that line.

zellij has `EditScrollback`, which dumps the pane's scrollback to a
tempfile and opens it in `$SCROLLBACK_EDITOR`. Two problems:

1. **It strips styles.** zellij stores scrollback as a styled cell
   grid internally, but the dump writes plain text — claude's TUI
   palette (cyan commands, dim block-quotes, color-coded diffs) all
   flatten to default fg. Reading colorless TUI output is painful.
2. **It opens in a new tiled pane.** Breaks pair's two-pane invariant
   and disables swap layouts (`exact_panes=2`) for the duration.

We want: from the agent pane (or anywhere), trigger a keybind that
opens the agent's full scrollback — **with colors preserved, line
numbers matching the indicator, in a floating pane** — and lets the
user `:880` to jump.

## Spec

A new keybind (e.g. `Alt+/`) that opens the agent's scrollback in a
floating nvim pane, with:

- **ANSI colors preserved** — claude's TUI palette renders as it did
  live.
- **Line numbers match zellij's indicator** — `:880` jumps to what
  was at line 880 in the rendered scrollback.
- **No layout disruption** — floating pane, dismissed with `:q`,
  pair stays 2-paned.

Out of scope (for this issue):
- Editing the scrollback (read-only is fine).
- Cross-agent: claude only initially. codex/gemini follow if the
  same machinery applies.
- Search-and-jump UX beyond what nvim already provides.

## Approach

### Capture: raw PTY stream in pair-wrap

`bin/pair-wrap` is the existing PTY proxy that already sees every byte
the agent emits. Extend it to **tee** the raw stream (escapes intact)
to `$DATA_DIR/scrollback-<tag>-<agent>.raw`. Truncate on launch — each
session is its own narrative; pair resume starts a fresh capture.

Side-channel events file:
`$DATA_DIR/scrollback-<tag>-<agent>.events.jsonl`

Each line is a JSON event with:
- `offset`: byte offset into `.raw` where the event takes effect
- `type`: `"resize"` (initial size at startup, plus any SIGWINCH)
- `cols`, `rows`: pane dimensions at that point

This lets a replayer feed each segment of `.raw` through a terminal
emulator at the correct width, so wrap behavior matches zellij's.

### Replay: pyte renderer

`bin/pair-scrollback-render` (Python, uses [pyte][1]):
1. Read `.events.jsonl` to know the resize timeline.
2. For each segment of `.raw`, feed it to a pyte `HistoryScreen` (which
   keeps scrollback rows as cells leave the visible region) sized to
   that segment's `(cols, rows)`. Update size at resize boundaries.
3. After processing, walk pyte's `history.top` + visible buffer + `history.bottom`
   and serialize each row to ANSI: emit SGR escapes for fg/bg/attrs,
   then the row's characters, then SGR reset at row end.
4. Write the result to `$DATA_DIR/scrollback-<tag>-<agent>.ansi`.

Output is one logical line per pyte row — line numbers match what the
zellij indicator showed (both apply width-based wrapping the same way).

[1]: https://github.com/selectel/pyte

### Open: floating pane + ANSI-aware nvim

A new keybind in `zellij/config.kdl` runs a wrapper script in a floating
pane:

```
bind "Alt /" {
    Run "pair-scrollback-open" {
        floating true
        close_on_exit true
        name "scrollback"
        width "100%"
        height "100%"
        x "0"
        y "0"
    };
}
```

`bin/pair-scrollback-open`:
1. Run `pair-scrollback-render` to produce the `.ansi` file.
2. `exec nvim -u $PAIR_HOME/nvim/scrollback.lua $DATA_DIR/scrollback-<tag>-<agent>.ansi`

`nvim/scrollback.lua` is a minimal init that:
- Parses ANSI SGR escapes inline and converts to extmark-based highlights
  (custom — keep it small, ~100 lines, support 16/256/truecolor + bold/italic).
  Alternative: vendor [baleia.nvim][2] or AnsiEsc.vim. Decision deferred to
  M3 implementation, but custom is preferred (fewer moving parts).
- Read-only buffer (`set readonly nomodifiable`).
- `q` to quit (since this is a viewer, not the draft).

[2]: https://github.com/m00qek/baleia.nvim

## Plan

### M1: Raw stream capture in pair-wrap

- [ ] Add a `--scrollback-log <path>` flag to `bin/pair-wrap`.
- [ ] When flag is present, on every read from agent → outer (the
      bytes the user sees), `os.write(scrollback_fd, chunk)` in
      addition to the existing forwarding.
- [ ] Truncate the file on open (each session is fresh).
- [ ] Update `bin/pair`: pass `--scrollback-log $DATA_DIR/scrollback-${PAIR_TAG}-${AGENT}.raw`
      when invoking pair-wrap.
- [ ] Verify: `tail -f` the file while running claude, see bytes growing
      with the agent's output.

### M2: SIGWINCH events sidecar

- [ ] In pair-wrap, hook `SIGWINCH` (it already proxies the signal to
      the agent's PTY); on each, log a `{type:"resize",offset,cols,rows}`
      line to `<scrollback>.events.jsonl`.
- [ ] Log an initial resize event at startup (for pyte's first config).
- [ ] Verify: trigger Alt+Up/Alt+Down rung swap, see resize entries
      appear with byte-offset boundaries.

### M3: pyte replayer

- [ ] New `bin/pair-scrollback-render` (Python). CLI:
      `pair-scrollback-render <raw> <events.jsonl> <out.ansi>`.
- [ ] Use `pyte.HistoryScreen` so scrollback rows are preserved.
- [ ] For each contiguous segment between resize events, set screen
      size and feed bytes; on resize boundary, resize the screen and
      continue.
- [ ] Serialize: iterate scrollback (history.top) → visible buffer →
      history.bottom. Per row, walk cells and emit minimal SGR diffs
      (only when style changes), then reset at row end.
- [ ] Truecolor support — pyte exposes RGB if the agent emitted
      truecolor escapes (claude does).
- [ ] Verify: render a session, count lines, eyeball a few line numbers
      against what the zellij indicator showed before scrollback dump.

### M4: nvim integration

- [ ] `nvim/scrollback.lua` — minimal init that loads the .ansi file,
      strips SGR escapes, applies extmarks for colors/attrs.
- [ ] ANSI parsing: support SGR codes `0` (reset), `1`/`22` (bold/no-bold),
      `3`/`23` (italic), `4`/`24` (underline), `30-37`/`90-97` (fg
      indexed), `40-47`/`100-107` (bg indexed), `38;2;r;g;b` and
      `38;5;n` (truecolor + 256), and `48;2;...`/`48;5;...` for bg.
- [ ] Read-only, `q` to quit.
- [ ] Verify: open a rendered .ansi, see colors match the live agent.

### M5: keybind + open script

- [ ] `bin/pair-scrollback-open` — orchestration: run renderer, exec
      nvim with scrollback.lua on the result.
- [ ] `zellij/config.kdl` — bind `Alt+/` (or another free chord) to
      Run `pair-scrollback-open` floating, close-on-exit.
- [ ] Verify: from agent pane, `Alt+/` opens floating nvim with
      scrollback, `:880` jumps to the right line, colors render,
      `:q` dismisses, pair returns to 2-pane layout.

### M6: atlas + cleanup

- [ ] Document the capture + render + open architecture in
      `atlas/architecture.md`.
- [ ] Note pyte dependency and how it's installed (probably
      `pip install pyte` in pair's bootstrap or doc'd as a prereq).

## Risks / open questions

- **Width-tracking accuracy.** zellij's wrap logic vs. pyte's: both
  follow VT100 spec for width-based wrap, but edge cases (zero-width
  combining chars, wide East-Asian glyphs, broken double-width) might
  differ by a row here and there. Test with real claude sessions.
- **TUI tricks.** claude uses alternate-screen for menus / popups,
  cursor-positioning for in-place spinners, scroll regions for streaming
  output. pyte handles these but has occasional quirks; expect a small
  amount of debugging during M3.
- **Performance.** Multi-hour sessions may be tens of MB. pyte at
  ~MB/s means a few seconds to render. Acceptable for a manual trigger
  but not for live tailing.
- **pyte as a dependency.** pip install required. Document in README.
- **Choice of plugin vs custom in M4.** Custom keeps the dependency
  surface tight (we already write a lot of nvim Lua). Will reassess
  during M4 if SGR parsing turns out gnarlier than expected.

## Log
