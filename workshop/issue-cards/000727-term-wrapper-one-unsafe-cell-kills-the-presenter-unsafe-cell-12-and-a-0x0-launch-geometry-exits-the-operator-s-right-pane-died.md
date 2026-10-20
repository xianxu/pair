---
id: 000727
status: open
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
github_issue:
---

# term wrapper: one unsafe cell kills the presenter ('unsafe cell 12'), and a 0x0 launch geometry exits: the operator's right pane died

## Problem

10-20 ~09:00, the operator's ops:0 right pane (nvim on the TL session file, launched as `v teams/devinfra/tl-status-xian-xu.md`
through pair's `term` wrapper, supervisor on the staged pair:0 build f3c597f8) crashed, and relaunching it failed:

- `term: terminal: unsafe cell 12` then `terminal: presenter unavailable` (screenshot: ops
  `teams/devinfra/assets/tl-status-xian-xu/2026-10-20-term-unsafe-cell.png`);
- `term: terminal: invalid geometry 0x0 (maximum 262144 cells)` (screenshot `…/2026-10-20-term-geometry-0x0.png`).

`unsafe cell` is `validateCell` (cmd/internal/terminal/frame.go:119): a cell whose content isn't plain text (a control
rune or invalid UTF-8), exceeds MaxClusterBytes, or has an invalid link. The session file itself has no control
characters (checked), so the offending cell comes from nvim's rendered screen (sign column, virtual text, an OSC 8
link, or a grapheme cluster). One bad cell ends the whole presenter, so the operator loses their pane; and a 0x0
geometry at launch (pane measured before it had a size) refuses outright instead of waiting for a real size.
