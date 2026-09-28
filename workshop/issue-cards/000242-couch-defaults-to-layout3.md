---
id: '000242'
status: done
started: 2026-09-13T15:55:46-07:00
created: 2026-09-13
updated: 2026-09-13
estimate_hours: 0.389
actual_hours: 0.62
---

# couch defaults to --layout3: every thread gets pair's right-hand terminal unless --layout2 is asked for

## Problem

`couch` with no flag runs layout2 (`couchcore/couch.go:107`, `Layout:
Layout2` in `New`; the CLI overrides it only when a flag is given,
`couchcmd/run.go:266-269`; the help text says so at `:715`). The operator
runs layout3 every time, and has since #198 made it reachable: every thread
record in the store is layout3 —

    ~/.local/share/pair/couch/threadstore/records: 7 records
      7 × "layout": "layout3"     (tools, parley.nvim, pair, arc-agi-3, brain, astro, ariadne)
      0 × pre-#198 (no field)

— so the default is the one thing nobody uses. The operator has not hit a
failure here; the ask is simply that the default match how couch is used, so
`couch` means the workbench that is actually run. (A consequence worth
knowing, not the motivation: with one layout per couch process and a startup
guard that refuses to mix, a flagless `couch` next to layout3 threads refuses
to start — `atlas/couch.md:1075-1083`.)

The reasons layout2 was the default are recorded and gone: the 2026-08-22 pin
("couch owns terminal switching, so layout3's third pane is the layer couch
replaces") was an actor-cluster claim that #170's rescope to couch-lite
invalidated, and #198 reversed the pin without moving the default. The
operator's own list (2026-09-09): layout2 was kept for simplicity, doubt
about hosting a browser on the right, and the right pane's quality — all
three have since changed (#199 gave the right pane its own tab bar; browser
hosting was dropped; couch-lite is done).
