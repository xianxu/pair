---
id: '000194'
status: wontfix
created: 2026-09-03
updated: 2026-09-06
---

# Toggle a terminal pane in layout2

## Problem

`layout2` is agent above draft, with no terminal. Checking anything —
`git status`, a test run, a file — means leaving the workbench or running it
through the agent, which spends a model turn on something the operator can read
in a second.

`layout3` already provides a terminal, but as a **permanent third pane** taking
50% of the width. That is a different working posture, not a quick check: the
operator wants the terminal *when they want it* and the full-width agent the
rest of the time.
