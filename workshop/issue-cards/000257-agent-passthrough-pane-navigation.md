---
id: 000257
status: open
created: 2026-09-15
updated: 2026-09-15
estimate_hours:
github_issue:
---

# Preserve pane navigation exceptions for agent passthrough

## Problem

Follow up to #245. Agent-focused routing now passes most Pair workbench chords
through to the focused agent for compatibility, but `Alt+j` and `Alt+k` must
remain Pair-owned pane-navigation actions. They move focus between the left and
right panes and are the keyboard escape path from an agent pane. Passing them to
the agent would let an agent bind or consume the keys and strand the operator in
the current pane.
