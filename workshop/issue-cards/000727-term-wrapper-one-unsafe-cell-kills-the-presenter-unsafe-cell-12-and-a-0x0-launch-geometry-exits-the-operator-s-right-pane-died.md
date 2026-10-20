---
id: 000727
status: codecomplete
created: 2026-10-20
updated: 2026-10-20
estimate_hours: 1.98
github_issue:
started: 2026-10-20T08:47:04-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
actual_hours: 1.69
tracker:
    version: 1
    completion:
        token: close-52cfcc3b173d
        repository: github.com/xianxu/pair
        reviewed_head: 97c32eaad2245af63532d100ac67e53283b268eb
        evidence_commit: a7690953db28fd0e97ac4551eee12aec1b788fc0
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
