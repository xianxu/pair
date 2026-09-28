---
id: '000015'
status: done
created: 2026-05-03
updated: 2026-05-03
actual_hours: N/A
---

# Prompt history & future queue navigation in nvim pane

## Problem

Two related shortcomings in the pair drafting flow today:

1. **No way to recall previous prompts.** `log-<tag>.md` accumulates every send, but the user has to leave the pane and grep the file to look at past prompts. There's no in-buffer way to "go back" to what was sent.
2. **No way to queue a thought while the agent is busy.** If an idea strikes while the agent is mid-task and the user wants to send it later (after responding to the current agent reply), there's nowhere to park it. They either lose it, send too early, or maintain it in their head.
