---
id: 000277
status: open
created: 2026-09-17
updated: 2026-09-17
estimate_hours:
github_issue:
---

# Context meter shows the dead conversation's count after Shift+Alt+N

## Problem

Shift+Alt+N restarts the agent with a **fresh conversation**, but the agent
pane's frame meter (`<agent> (<count>)`, #71) keeps rendering the **previous**
conversation's context-window size. Observed sequence (operator, 2026-09-17):

1. Shift+Alt+N → the frame still shows the old, now-dead count (e.g. `claude (562k)`).
2. First `Return` in the new conversation → the count blanks to bare `claude`.
3. Once the new session's transcript carries usage → the real (small) count appears.

Steps 2 and 3 are correct. Step 1 is the defect: for the whole window between the
restart and the first submit — unbounded, since it's however long the operator
takes to type the first prompt — the meter is actively lying about which
conversation it measures. The failure mode is the same *class* as #98 (a frozen
number sourced from a dead transcript), but a different mechanism: #98 rendered a
stale *twin agent*; this renders a stale *paint*.
