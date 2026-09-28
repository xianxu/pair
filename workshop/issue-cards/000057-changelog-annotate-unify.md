---
id: '000057'
status: done
created: 2026-06-12
updated: 2026-06-17
estimate_hours: 4.5
actual_hours: 0.55
---

# Alt+q annotation in change-log viewer + shared nvim/annotate.lua

## Problem

The `Alt+l` change-log viewer (#53) is read-only with no way to react to an
entry. The scrollback viewer (`Alt+/`) already has the right affordance: `Alt+q`
drops a 🤖-marker (a question/comment) on a line or selection, and on quit the
markers ship to the draft pane (→ the agent) via a pending sidecar. We want the
**same Alt+q flow in the change-log viewer** — ask a question about a logged
milestone/decision and have the agent see it.

Both viewers are read-only nvim windows over pair state, so the marker machinery
should be **shared, not duplicated** — but they differ (scrollback renders SGR
via extmarks; changelog renders markdown + has an async background refresh), so
unification is partial.

(Deferred from #53; operator asked to split it out since it refactors the
working scrollback viewer.)
