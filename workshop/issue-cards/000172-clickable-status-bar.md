---
id: '000172'
status: done
started: 2026-09-05T12:13:46-07:00
created: 2026-09-02
updated: 2026-09-06
estimate_hours: 2.69
actual_hours: 5.62
---

# Mouse support: click the status bar and the switcher

## Problem

The reserved row already renders one chip per actor (`couchtty/reserve.go` —
`StatusActor` "one chip on the row", `RenderStatusRow`), and those chips are
exactly the things the operator wants to reach. Reaching one today means
`ctrl-space`, then finding it in the switcher — a keyboard round trip to select
something already visible and already pointed at.
