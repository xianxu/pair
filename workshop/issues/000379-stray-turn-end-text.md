---
id: 000379
status: open
deps: []
github_issue:
created: 2026-10-01
updated: 2026-10-01
estimate_hours:
card_mirror: '290451c59fe9b2f0e6db1363fea7279f6f0c8dfd' # card fields mirrored from issue-cards; edit via sdlc
---

# Turn-end text flashes at focused pane cursor

Low priority. The operator is curious about the cause; Couch's screen handling is otherwise solid.

## Problem

Under Couch, Claude's turn-end line ("✻ Sautéed for 10m …") sometimes flashes
**outside the agent pane**, at the screen cursor of the focused pane (the
draft nvim or the right terminal). It stays for about a second, then a repaint
covers it. It is too brief to screenshot, and it's the only screen artifact the
operator has seen. Plain `pair` hasn't been checked; the operator now uses Couch.

Evidence so far (2026-10-01):

- Claude draws that line with **only relative** cursor motion, right after
  closing one synchronized frame, sending progress-bar escape codes and
  opening the next frame (recorded in `scrollback-couch-…-claude.raw`):
  `ESC[?2026l ESC]9;4;0;BEL ESC]9;4;3;BEL ESC[?2026h ESC[?25l ESC[2D ESC[3B \r ESC[9A ✻ Churned for 1m 45s`.
  Inside the zellij pane that is fine, because zellij tracks each pane's cursor.
- Text landing at the *focused pane's* cursor means that, between zellij's client
  output and Couch, a "move cursor to (row, col)" instruction was lost (or never
  emitted). Most likely it was the first one in a redraw, since zellij leaves
  the cursor at the focused pane between redraws.
- It is **intermittent**, but Claude sends the same bytes at every turn end, so
  the cause must depend on timing (read boundaries, or interleaved redraws),
  not on the byte content.
- `Endpoint.Feed` (`cmd/internal/terminal/endpoint.go`) hands each chunk
  directly to the `vt` emulator, and that parser keeps its state between
  chunks. A sequence split across two reads in Couch should therefore parse
  correctly; a bug inside the `third_party/vt` fork isn't ruled out.

## Spec

Hypotheses, in rough order:

1. **Interleaved redraws.** zellij redraws two panes in the same instant (the
   operator typing in draft, or the right terminal printing, while Claude
   finishes) and loses or reorders a cursor move. This would explain why it's
   rare; it is the leading theory if the operator recalls concurrent activity.
2. **zellij 0.45.1 race** when Claude closes one frame and immediately opens
   another.
3. **`third_party/vt` mis-parses** one specific chunk split.
4. **Couch's 150ms timeout for synchronized frames** publishing a half-drawn
   frame. This is weak: text would still be correctly positioned.

No existing log captures the deciding stream. The `.raw` scrollback records
Claude → zellij. `zellij.log` records no render bytes. `COUCH_TRACE`,
`COUCH_INPUT_TRACE` and `COUCH_MOUSE_TRACE` record timing, keystrokes and mouse
events, not output.

Instrument: an opt-in output recorder at `Endpoint.Feed`, enabled by an
environment variable that names a file, following the `trace.go` pattern. It
logs each chunk zellij sends to Couch with a timestamp and its boundaries. The
stream contains everything shown on screen, so it stays opt-in and the file is
deleted after the investigation, as with `COUCH_INPUT_TRACE`.

Analysis: find the turn-end paint in the log and check whether a cursor-position
instruction precedes it, and where the chunk splits fall. That separates "zellij
never sent the move" from "Couch lost it". Reproduce on purpose by typing in
draft as a long turn finishes.

## Done when

- The opt-in recorder exists, is off by default, and is tested (records chunk
  boundaries, writes nothing when off).
- At least one captured occurrence has been analysed, and the layer at fault
  (zellij, `vt` fork, Couch) is named in the Log, with an upstream report or a
  fix filed accordingly.

## Plan

- [ ]

## Log

### 2026-10-01

- Filed from a diagnosis conversation; hypotheses above. The operator could not
  say whether the earlier sightings coincided with typing.
