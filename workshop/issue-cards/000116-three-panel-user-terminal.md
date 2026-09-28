---
id: '000116'
status: done
started: 2026-07-23T16:16:22-07:00
created: 2026-07-23
updated: 2026-07-24
estimate_hours: 7.02
actual_hours: 5.03
---

# Three-panel Pair layout with user terminal

## Problem

Pair's current session layout is optimized around two surfaces: an agent output
pane and a Neovim draft/input pane. That leaves the user's own shell workflow
outside the main Pair workspace, even though real pairing often needs a terminal
for commands, ad hoc inspection, or opening a full Neovim instance while the
agent continues working.

The desired workbench shape is a three-panel session: preserve the familiar
agent/draft split, but add a first-class user terminal surface where the user
can run a shell or launch `nvim` without stealing the agent/draft panes.
