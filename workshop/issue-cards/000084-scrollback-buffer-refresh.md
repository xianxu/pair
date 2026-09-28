---
id: '000084'
status: done
started: 2026-06-29T11:13:55-07:00
created: 2026-06-29
updated: 2026-06-29
estimate_hours: 1.0
actual_hours: 0.35
---

# scrollback nvim buffer refresh

## Problem

The Alt+/ scrollback viewer is currently a static snapshot:
`pair-scrollback-open` renders the agent pane's raw capture to `.ansi`, starts a
read-only nvim, and that buffer never sees later agent output. In a long-running
session, the user has to close and reopen the viewer to inspect new bottom
content.

There should be a way for scrollback nvim to load new content from within the
existing viewer. In particular, `G` should keep its normal meaning — go to the
end — but should first refresh the backing scrollback so "end" means the latest
rendered output.
