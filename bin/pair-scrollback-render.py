#!/usr/bin/env python3
"""pair-scrollback-render — replay a pair-wrap raw capture through pyte.

Inputs (produced by pair-wrap with --scrollback-log):
  <raw>           bytes the agent emitted via its master PTY
  <events.jsonl>  one JSON line per resize: {"type":"resize","offset":N,"cols":C,"rows":R}

Output:
  <out.ansi>      one rendered scrollback line per logical pyte row,
                  styles re-emitted as SGR escapes so the file's bytes
                  reproduce the agent's original colors when displayed
                  through any ANSI-aware viewer (terminal, AnsiEsc.vim,
                  baleia.nvim, etc.)

Approach: pyte.HistoryScreen does the same VT100 interpretation that
zellij does live, including width-based wrap, alternate-screen flips,
and scroll-region handling. Feeding the raw bytes at the right widths
(per the events sidecar) makes pyte's row count match what zellij's
top-right indicator showed, so a remembered "line 880" lands on the
right line in the output.

Usage:
  pair-scrollback-render <raw> <events.jsonl> <out.ansi>

Filed as #000017 M3.
"""

import collections
import json
import os
import re
import sys
from typing import Iterable, Optional

import pyte


# Strip xterm / kitty private CSI sequences that pyte 0.8.2 mis-handles:
#
#   \x1b[<u, \x1b[>1u   — kitty keyboard protocol
#   \x1b[>4;2m          — XTERM modifyOtherKeys (claude emits this at
#                         startup). pyte's SGR handler drops the `>`
#                         prefix and reads the params as standard SGR,
#                         so `>4;2m` becomes `4;2m` → underscore + faint
#                         sticky on every subsequent cell. This was the
#                         "everything is underlined" bug in the rendered
#                         scrollback (#000018 follow-up).
#
# Leaving DEC private (`\x1b[?...`) alone — pyte handles those modes
# correctly. CSI grammar: ESC `[` private-prefix? params intermediates final.
PRIVATE_CSI_RE = re.compile(rb'\x1b\[[<>][0-9;:]*[ -/]*[@-~]')


# Cap on how many scrolled-out rows we retain. Generous because the
# point of this whole feature is "I remember a line from way back".
# At ~100B/row average, 100k rows is ~10MB — fine.
HISTORY_ROWS = 100_000


class CaptureScreen(pyte.Screen):
    """Plain Screen + a manual deque of rows that have scrolled off the top.

    Why not pyte.HistoryScreen: HistoryScreen overrides __getattribute__ to
    wrap every event handler with before_event / after_event hooks. The
    override fires on every method dispatch and every attribute read —
    ~19M calls for a 3 MB raw input in our use, ~95% of wall time. We
    don't need its page-navigation features (history.bottom, prev_page /
    next_page); we only need the scrolled-out rows. Capturing them
    directly in index() — the single point where pyte actually drops a
    row off the top of the buffer — gives identical output at 3-4x the
    speed.

    Resize-induced row loss is not handled here: the renderer feeds
    segments at their original widths, and pyte's resize() on a Screen
    keeps the visible buffer, just clamping width. Rows lost across
    resize boundaries were never in the live screen's history either, so
    this matches what the user saw.
    """

    def __init__(self, columns: int, lines: int, history_rows: int = HISTORY_ROWS) -> None:
        super().__init__(columns, lines)
        self.history_top = collections.deque(maxlen=history_rows)

    def index(self) -> None:
        top, bottom = self.margins or (0, self.lines - 1)
        if self.cursor.y == bottom:
            # parent's index() is about to drop self.buffer[top] — grab it
            # before the shift. Falls back to None for never-written rows,
            # which serialize_row turns into an empty line.
            self.history_top.append(self.buffer.get(top))
        super().index()

# Default fallback if events.jsonl is empty or the first event is missing
# dimensions. Should rarely fire (pair-wrap always logs an initial
# resize at offset 0); kept as a safety net so the renderer doesn't
# crash on a malformed sidecar.
DEFAULT_COLS = 80
DEFAULT_ROWS = 24


# Map pyte's lowercase color names to the SGR base codes (30 + offset
# for the standard 8, 90 + offset for the "bright" 8). pyte converts
# 38;5;N indexed colors to RGB hex strings before they reach Char.fg,
# so we only need to handle named + RGB hex below.
NAMED_COLORS = {
    "black": 0, "red": 1, "green": 2, "yellow": 3,
    "blue": 4, "magenta": 5, "cyan": 6, "white": 7,
    "brightblack": 8, "brightred": 9, "brightgreen": 10, "brightyellow": 11,
    "brightblue": 12, "brightmagenta": 13, "brightcyan": 14, "brightwhite": 15,
}


def _color_sgr_codes(value: str, fg: bool) -> list[str]:
    """Return the SGR fragments to set fg-or-bg to `value`.

    fg=True returns 30/90/38-family codes; fg=False returns 40/100/48.
    "default" → 39 / 49. Unknown values fall back to default.
    """
    base_low = 30 if fg else 40
    base_high = 90 if fg else 100
    default = "39" if fg else "49"
    if value == "default":
        return [default]
    n = NAMED_COLORS.get(value)
    if n is not None:
        return [str((base_low if n < 8 else base_high) + (n if n < 8 else n - 8))]
    # Truecolor: pyte stores as a 6-char lowercase hex string ("64c8ff").
    if isinstance(value, str) and len(value) == 6:
        try:
            r = int(value[0:2], 16)
            g = int(value[2:4], 16)
            b = int(value[4:6], 16)
        except ValueError:
            return [default]
        prefix = "38" if fg else "48"
        return [f"{prefix};2;{r};{g};{b}"]
    return [default]


def char_sgr(ch: pyte.screens.Char) -> str:
    """Build the SGR escape that sets attrs to match `ch` exactly.

    Always emits a full reset-and-reset state (bold, italic, etc.)
    rather than diffing against the previous cell. Slightly larger
    output than ideal, but trivially correct and the output is plain
    text — viewer file size isn't a hard constraint here.
    """
    parts = ["0"]  # reset to known state, then layer on
    if ch.bold:
        parts.append("1")
    if ch.italics:
        parts.append("3")
    if ch.underscore:
        parts.append("4")
    if ch.blink:
        parts.append("5")
    if ch.reverse:
        parts.append("7")
    if ch.strikethrough:
        parts.append("9")
    parts.extend(_color_sgr_codes(ch.fg, fg=True))
    parts.extend(_color_sgr_codes(ch.bg, fg=False))
    return "\x1b[" + ";".join(parts) + "m"


def _char_attrs(ch: pyte.screens.Char) -> tuple:
    """Hashable attribute tuple, used to skip SGR re-emission within a row."""
    return (ch.fg, ch.bg, ch.bold, ch.italics, ch.underscore,
            ch.blink, ch.reverse, ch.strikethrough)


def serialize_row(row, cols: int) -> str:
    """Render one pyte row as a single line of ANSI-decorated text.

    Trims trailing blank cells (default-styled spaces) so a viewer
    doesn't have to scroll past pad. Empty rows return an empty string.
    Each non-empty row ends with `\\x1b[0m` to keep styles from bleeding
    into the next line if the viewer concatenates without resetting.

    `row` may be None for a CaptureScreen history entry that scrolled
    out before it was ever written — treat as an empty line.
    """
    if row is None:
        return ""
    last_nonblank = -1
    for x in range(cols):
        ch = row[x]
        # A "blank" cell is a space at default fg/bg and no decorations.
        if ch.data and ch.data != " ":
            last_nonblank = x
        elif ch.bg != "default":
            # Non-default background space (e.g. inverse-video padding) is
            # visible content — keep it.
            last_nonblank = x
    if last_nonblank < 0:
        return ""
    out = []
    last_attrs: Optional[tuple] = None
    for x in range(last_nonblank + 1):
        ch = row[x]
        attrs = _char_attrs(ch)
        if attrs != last_attrs:
            out.append(char_sgr(ch))
            last_attrs = attrs
        out.append(ch.data or " ")
    out.append("\x1b[0m")
    return "".join(out)


def parse_events(path: str) -> list[dict]:
    """Read events.jsonl into a list of dicts. Empty file → empty list."""
    events = []
    try:
        with open(path) as f:
            for line in f:
                line = line.strip()
                if not line:
                    continue
                try:
                    events.append(json.loads(line))
                except json.JSONDecodeError:
                    # Malformed line — skip but keep going. Better to
                    # render with imperfect width-tracking than crash.
                    continue
    except FileNotFoundError:
        pass
    return events


def initial_size(events: list[dict]) -> tuple[int, int]:
    """Pull (cols, rows) from the first resize event, or fall back to defaults."""
    for e in events:
        if e.get("type") == "resize" and "cols" in e and "rows" in e:
            return int(e["cols"]), int(e["rows"])
    return DEFAULT_COLS, DEFAULT_ROWS


def feed_segments(stream: pyte.Stream, screen: CaptureScreen,
                  raw: bytes, events: list[dict]) -> None:
    """Feed `raw` to `stream` in segments delimited by resize events.

    Each event's `offset` is the byte index in `raw` at which the new
    (cols, rows) takes effect. Segments before the first resize are
    fed at the screen's initial size (set by the caller from the first
    event). pyte expects str input, so we decode each segment as UTF-8
    with errors='replace' — segment boundaries falling mid-codepoint
    are rare and a single replacement glyph is cheaper than running an
    incremental decoder.

    Private CSI sequences are stripped from each chunk *before* decode
    so pyte never sees the offending bytes. Stripping at the chunk
    level (rather than once on the whole `raw`) keeps the byte
    offsets in `events` aligned with what we feed: a resize event at
    offset N applies to the chunk that contained byte N originally.
    A private CSI straddling a chunk boundary is rare enough that the
    decode-replace fallback handles it harmlessly.
    """
    resize_events = [e for e in events if e.get("type") == "resize"]
    # The first resize sets initial size; we apply it before this call,
    # so skip it here and only act on subsequent resizes as boundaries.
    boundaries = [(int(e["offset"]), int(e["cols"]), int(e["rows"]))
                  for e in resize_events[1:]]
    cursor = 0
    for offset, cols, rows in boundaries:
        if offset > cursor:
            chunk = PRIVATE_CSI_RE.sub(b"", raw[cursor:offset])
            stream.feed(chunk.decode("utf-8", errors="replace"))
            cursor = offset
        screen.resize(rows, cols)
    if cursor < len(raw):
        chunk = PRIVATE_CSI_RE.sub(b"", raw[cursor:])
        stream.feed(chunk.decode("utf-8", errors="replace"))


def iter_all_rows(screen: CaptureScreen) -> Iterable:
    """Yield every logical row in display order: scrolled-out → visible buffer.

    Visible buffer iteration uses `range(screen.lines)` rather than
    `buffer.keys()`. pyte's buffer is sparse — unwritten rows aren't in
    `.keys()` — so walking keys silently skips trailing blank rows when
    the agent cleared and paused mid-redraw. That would shift every
    subsequent line number, breaking the feature's core promise that
    `:880` lands where zellij showed line 880. The StaticDefaultDict
    yields a default-style Char for missing rows, and serialize_row
    returns "" for an all-blank row, so the count stays correct.

    `history_top` entries may be None for rows that scrolled out before
    they were ever written to (pyte's buffer.get(top) returns None for
    missing keys). serialize_row handles None as "empty line".
    """
    for row in screen.history_top:
        yield row
    for y in range(screen.lines):
        yield screen.buffer[y]


def render(raw_path: str, events_path: str, out_path: str) -> None:
    events = parse_events(events_path)
    cols, rows = initial_size(events)
    screen = CaptureScreen(cols, rows, history_rows=HISTORY_ROWS)
    stream = pyte.Stream(screen)
    with open(raw_path, "rb") as f:
        raw = f.read()
    feed_segments(stream, screen, raw, events)

    final_cols = screen.columns
    viewport_top = len(screen.history_top) + 1  # 1-indexed line where visible buffer starts
    out_lines = []
    for row in iter_all_rows(screen):
        out_lines.append(serialize_row(row, final_cols))
    # Trim trailing blank lines so a half-empty visible buffer doesn't
    # leave a tail of empties at the end of the file.
    while out_lines and out_lines[-1] == "":
        out_lines.pop()

    # Write the viewport sidecar *before* the .ansi rename so a reader
    # that opens the new .ansi never sees a stale sidecar from the
    # previous render. Sidecar is best-effort; on failure scrollback.lua
    # falls back to its prior bottom-alignment behaviour.
    viewport_path = (out_path[:-len(".ansi")] if out_path.endswith(".ansi") else out_path) + ".viewport"
    try:
        with open(viewport_path, "w", encoding="utf-8") as f:
            f.write(f"{viewport_top}\n")
    except OSError:
        pass

    # Atomic write: a second concurrent render (e.g. user double-tapping
    # Alt+/) would otherwise race on truncate-then-write of the same
    # path, leaving nvim with whatever survived the interleave. Write
    # to a sibling tempfile, then os.replace — guarantees readers see
    # either the old file or the complete new one.
    tmp_path = out_path + ".tmp"
    with open(tmp_path, "w", encoding="utf-8") as f:
        f.write("\n".join(out_lines))
        if out_lines:
            f.write("\n")
    os.replace(tmp_path, out_path)


def main(argv: list[str]) -> int:
    if len(argv) != 4:
        print("usage: pair-scrollback-render <raw> <events.jsonl> <out.ansi>",
              file=sys.stderr)
        return 2
    render(argv[1], argv[2], argv[3])
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
