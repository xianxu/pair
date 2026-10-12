# Claude Code 2.1.296 mid-turn screens (pair#438)

Re-rendered, not raw streams. The source is the recorded `parley.nvim:1` pane on
2026-10-10, 23:45–23:46: Claude Code 2.1.296 in a long turn, with the ops TL's
delivery `de593285` pasted into it. The recording's scrollback was replayed
through the wrapper's terminal model (94×39) and cut at chunk boundaries that
fall between synchronized updates. Each screen's cells, faint/bold attributes,
truecolor foregrounds and cursor were then written back as one synchronized
frame. The recording has no full repaint to cut from, so a raw slice would not
be self-contained.

- `turn-empty.raw`: the spinner running ("Skedaddling… 5m 24s") over an empty
  composer.
- `turn-paste.raw`: the same turn with the peer envelope pasted into the
  composer (reads occupied).
- `turn-queued.raw`: after the submit, Claude Code queued the message behind
  the turn. The message sits above the spinner with "ctrl+enter to send now",
  and the composer shows the faint hint "Press up to edit queued messages",
  which reads empty.

## Picker latch (pair#464)

Re-rendered the same way, from the `pair:3` pane on 2026-10-11, 23:45–00:12
(`scrollback-1-pair-7-claude.raw`). There, three peer messages expired "waiting
for picker to close" while no picker was on screen.

- `permission-autodeny.raw`: a Bash permission prompt ("Dangerous rm
  operation…") with Claude Code's countdown, "will automatically deny this
  request in 1:53". No composer is on screen. Claude Code sent the
  `Claude needs your permission` OSC while this was up.
- `permission-autodenied.raw`: the first frame after the countdown ran out.
  The prompt is gone with no key pressed, and the composer is back (empty).
- `shell-footer.raw`: a turn with a background shell, whose footer reads
  "1 shell · … · ↓ to manage". The composer is empty.
- `idle-suggestion.raw`: the idle slot the messages couldn't reach. Its
  composer shows Claude Code's faint prompt suggestion ("The operator reviewed
  the format; proceed to close and merge"), which reads empty.
