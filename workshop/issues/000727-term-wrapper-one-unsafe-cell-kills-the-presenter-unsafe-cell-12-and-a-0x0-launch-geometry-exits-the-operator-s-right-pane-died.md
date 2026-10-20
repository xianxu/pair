---
id: 000727
status: open
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
collaboration-mode: collaboration
card_mirror: '8ce9379f7a5b33f07ee9b1958354172ac18cdc39' # card fields mirrored from issue-cards; edit via sdlc
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

## Spec

- An unsafe cell degrades that cell (render a replacement character, log once with the codepoints), never kills the
  presenter. The safety intent (no control sequences reach the parent terminal) is kept by the replacement.
- A 0x0 or transient geometry at launch waits briefly for a real size (or uses a minimal default) instead of exiting.
- Capture the offending cell's content in the diagnostic so the trigger is identifiable.

## Done when

- A frame with a control rune in one cell renders with a replacement and the presenter stays up (test).
- Launch with a 0x0 geometry that becomes 80x24 shortly after succeeds (test).
- The operator's `v teams/devinfra/tl-status-xian-xu.md` opens again (live).

## Plan

- [ ]

## Log

### 2026-10-20
