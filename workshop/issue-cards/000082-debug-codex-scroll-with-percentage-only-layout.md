---
id: '000082'
status: punt
started: 2026-06-26T15:49:13-07:00
created: 2026-06-26
updated: 2026-06-29
---

# Debug Codex scroll with percentage-only layout

## Problem

Codex scrolling can still wedge in Pair even after #68's tracing work. The
current logs show pair-wrap still forwarding stdout and writing raw scrollback
with no write errors, while zellij reports layout/render errors:

- `Can't combine fixed panes`
- `Failed to focus stacked pane`
- `Failed to find position of flexible pane`

#68 established the tracing substrate and ruled out several earlier theories
(prompt injection, pair-wrap itself, sync-marker stripping as the sole cause).
This issue tracks the next focused experiment: remove integer fixed pane sizes
from Pair's zellij layout and run Codex that way long enough to see whether the
scroll wedge disappears.
