---
id: '000092'
status: done
started: 2026-07-01T09:40:38-07:00
created: 2026-07-01
updated: 2026-07-01
estimate_hours: 6.52
actual_hours: 2.61
---

# route internal calls through Go dispatcher

## Problem

The runtime tree still ships several standalone Go helper binaries
(`pair-wrap`, `pair-slug`, `pair-changelog`, `pair-continuation`, `pair-context`,
`pair-scribe`, `pair-scrollback-render`, `pair-session-watch`) that generated
internal call-sites (nvim Lua, shell hooks, `pair-wrap` turn-end) invoke by
name. #76 proved the pattern — `pair-go context` and `pair-go scrollback-render`
dispatch through a shared internal runner while the legacy binary names stay
live — but it stopped there (`pair slug` was explicitly left as a later
candidate). As long as every helper is its own binary that callers hardcode,
the runtime bundle can't shrink toward a single executable.
