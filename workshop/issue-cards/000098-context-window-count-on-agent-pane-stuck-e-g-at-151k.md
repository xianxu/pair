---
id: '000098'
status: wontfix
started: 2026-07-05T21:45:50-07:00
created: 2026-07-01
updated: 2026-07-05
---

# context-window count on agent pane stuck (e.g. at 151k)

## Problem

The context-window size shown in the agent-pane frame title — the `(<count>)` in
`<agent> (<count>) [<cwd>]` (#71) — appears **stuck**. Reported: it used to track
down and, when it dropped to low 100k, compaction happened; now it seems frozen
at ~151k and no longer moves.
