---
id: '000088'
status: done
started: 2026-06-29T18:54:30-07:00
created: 2026-06-29
updated: 2026-06-29
actual_hours: N/A
---

# Revert codex Alt Enter submit regression from 87

## Problem

After #87, Codex submit is broken for every Return modifier — the user can no
longer submit at all from the Codex pane. #87 (authored by Codex) changed
`sendKeymapByAgent["codex"].altCR` from a bare `\r` to `\x1b\r` (ESC CR),
theorizing Codex needed the "modified submit chord." That diagnosis was wrong:

- Codex's input parser reads a lone `\r` as the **Enter** key. #31/#34 proved
  this — overlay-active emits a bare `\r` and Codex confirms pickers with it.
- `\x1b\r` parses as **Alt+Enter** (ESC = Alt prefix), which Codex does **not**
  bind to submit. So after #87 Alt+Enter inserts nothing / no-ops and the
  composer text just sits there.
- `altCR: \r` was the stable mapping from 2026-05-10 → 2026-06-29 (7 weeks of
  working Codex submit) and still matches `agy`'s convention.

#87 misattributed the **draft-pane** submit failure (#86's domain — nvim writes
body then sends a separate `send-keys "Alt Enter"`) to the byte value, and
"fixed" it by corrupting the universal submit byte, breaking direct Alt+Enter
submit too.
