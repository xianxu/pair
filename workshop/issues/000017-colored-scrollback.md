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

- [x] In pair-wrap, hook `SIGWINCH` (it already proxies the signal to
      the agent's PTY); on each, log a `{type:"resize",offset,cols,rows}`
      line to `<scrollback>.events.jsonl`.
- [x] Log an initial resize event at startup (for pyte's first config).
- [ ] Verify (manual): trigger Alt+Up/Alt+Down rung swap, see resize
      entries appear with byte-offset boundaries that match `wc -c
      <scrollback>.raw` checkpoints.

### M3: pyte replayer

- [x] New `bin/pair-scrollback-render` (Python). CLI:
      `pair-scrollback-render <raw> <events.jsonl> <out.ansi>`.
- [x] Use `pyte.HistoryScreen` so scrollback rows are preserved.
- [x] For each contiguous segment between resize events, set screen
      size and feed bytes; on resize boundary, resize the screen and
      continue.
- [x] Serialize: iterate scrollback (history.top) → visible buffer.
      (history.bottom skipped — it's for alt-screen page-forward, not
      part of the user's scrollback mental model.) Per row, emit SGR
      only when style changes, reset at row end.
- [x] Truecolor support — pyte normalizes 256-color and 24-bit RGB
      into 6-char hex strings on Char.fg/bg; renderer detects hex vs.
      named and emits 38;2;R;G;B accordingly.
- [ ] Verify (manual, against a real session): render a captured
      session, count lines, eyeball a few line numbers against what
      the zellij indicator showed.

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

### 2026-05-09

**M1 done — raw stream capture.** pair-wrap now accepts
`--scrollback-log <path>`. When set, opens with `O_TRUNC | O_CREAT | O_WRONLY`,
tees every chunk read from the agent's master PTY into the file, and
swallows write errors (proxy never blocks). Flag parsing is a tiny
prelude before the existing `agent_basename = os.path.basename(argv[0])`;
existing flagless invocations remain unchanged. `--` ends flag parsing
in case the wrapped command itself takes a `--something` argv[0].

Not yet wired in the layout — `zellij/layouts/main.kdl` still launches
pair-wrap without the flag. M5 will turn it on once the renderer +
viewer exist; landing it now would create per-session .raw files that
no consumer reads.

**M2 done — events sidecar.** `--scrollback-log /path/foo.raw` now
also opens `/path/foo.events.jsonl` (truncated). `set_winsize` was the
existing single-source entry point for both the initial size and every
SIGWINCH; threaded a `log_scrollback_event` call through it. The first
event lands right after pty.fork at offset 0. Counter `SCROLLBACK_BYTES`
is bumped after each successful write to the .raw fd, so the next resize
event's `offset` cleanly demarcates "from this byte onward, use these
new (cols, rows)". JSON is one line per event:

```
{"type":"resize","offset":0,"cols":120,"rows":30}
{"type":"resize","offset":18324,"cols":120,"rows":15}
```

**Manual smoke test** (sandbox blocks pty allocation, so this must be
done in a real terminal):

```
mkdir -p /tmp/pwtest
python3 bin/pair-wrap --scrollback-log /tmp/pwtest/foo.raw \
    /usr/bin/env bash -c 'echo hello scrollback; sleep 0.1'
xxd /tmp/pwtest/foo.raw | head -3
cat /tmp/pwtest/foo.events.jsonl
# expect: at least one resize event at offset 0
```

**M3 done — pyte renderer.** `bin/pair-scrollback-render <raw>
<events.jsonl> <out.ansi>` replays the captured stream through
`pyte.HistoryScreen(cols, rows, history=100_000)`, applying resize
events at their byte offsets. Output is one line per pyte row
(history.top in order, then visible buffer), each line carries SGR
codes that reproduce pyte's Char.fg/bg/decorations. Tested on two
synthetic streams in `$TMPDIR/sb-render-test/`:

1. **Color preservation** — 12 lines mixing default / red / truecolor
   bold. Output: 12 lines, each with the right SGR (e.g.
   `\x1b[0;1;38;2;100;200;255;49m`).
2. **Scrollback + mid-stream resize** — 50 lines into a 5-row screen
   with a resize from 40 cols → 20 cols at byte 250. Output: all 50
   lines preserved, line numbers monotonically `001..050`.

pyte is a new runtime dependency (`pip install pyte`). Will document
in M6 alongside the atlas update.
