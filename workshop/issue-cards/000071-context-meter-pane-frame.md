---
id: '000071'
status: done
started: 2026-06-25T22:10:36-07:00
created: 2026-06-25
updated: 2026-06-26
estimate_hours: 4.7
actual_hours: 2.12
---

# per-agent context-window meter in the zellij pane frame

## Problem

The operator has no at-a-glance signal for how full each agent's context window is, so
deciding when to start a fresh session (Shift+Alt+N) is guesswork. We want the live context
size shown in each agent's **zellij pane frame title**, beside the agent name and cwd —
`claude (970k) [~/brain]`.

(Migrated from ariadne#131, which was misfiled — this is 100% `pair` code. The ariadne issue
originally proposed *estimating* context from scrollback line count; brainstorm + review
superseded that with reading precise token usage from each agent's transcript. Full review
history lives in ariadne#131's Log; the durable design is below.)
