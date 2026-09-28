---
id: '000096'
status: done
started: 2026-07-01T13:38:04-07:00
created: 2026-07-01
updated: 2026-07-01
estimate_hours: 3.25
actual_hours: 1.25
---

# route pair-wrap and pair-scribe through dispatcher

## Problem

`pair-wrap` (the PTY proxy that wraps every agent turn) and `pair-scribe` (a PTY
logging wrapper) are the two remaining Pair-owned helper binaries not routed
through the Go dispatcher. Both are **interactive PTY proxies** — they take over
real stdio, set raw terminal mode, handle signals (SIGWINCH), and run for the
life of a session — so they do NOT fit the buffered `Dispatch(args) Result`
path that #76 used for `context`/`scrollback-render`. They were carved out of
#92 (which routes the finite internal-call helpers) because they are session
*entrypoints*, not internal calls Pair makes, and because pair-wrap wraps every
turn: a regression there breaks all sessions, so it deserves its own review
boundary and focused verification.

Note: `pair-scribe` also appears **orphaned** — built as a binary but not in the
runtime bundle manifest and with no Pair-owned caller found in the tree — so this
issue must first decide whether to route it for surface consistency or retire it.
