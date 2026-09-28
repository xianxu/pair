---
id: '000128'
status: done
started: 2026-07-30T16:48:13-07:00
created: 2026-07-28
updated: 2026-07-30
estimate_hours: 2.26
actual_hours: 1.01
---

# share escape-sequence framing between termcmd and wrapcmd

## Problem

Two packages in one binary now carry their own escape-sequence framing:

- `wrapcmd`: `otherEscRe` (wrap.go:189) encodes CSI plus OSC terminated by
  BEL-or-ST, and `stripCodexOutputMarkers` (wrap.go:766) is a byte-level marker
  stripper with tail carry (`p.stdoutPending`, wrap.go:310).
- `termcmd`: `queries.go` (#127) frames CSI and OSC to strip capability queries
  out of a tab redraw.

The *policy* tables must stay separate — they are in one case opposed, since
`wrapcmd` strips `\x1b[>7u` so codex stops pushing Kitty flags while `termcmd`
requires `\x1b[>1u` to survive a replay. What should not be duplicated is the
**framing**: "where does this CSI/OSC end".

#127 deliberately scoped the extraction out: the repo is a flat
`cmd/internal/<pkg>` layout with no shared home, so extracting would have created
a package as a side effect of a two-defect bugfix. `otherEscRe` is also consumed
three ways — `ReplaceAll` in `stripTerminalControls` (wrap.go:812) and the
capture-early path (wrap.go:1151), and `FindIndex` **at an offset** in the
colored-run walker (wrap.go:1018) — so a byte-scanner does not drop in; sharing
means rewriting three call sites that feed scrollback capture and agent-output
detection.
