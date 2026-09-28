---
id: 000302
status: open
created: 2026-09-20
updated: 2026-09-20
estimate_hours:
github_issue:
---

# Preserve long draft submissions to agent

## Problem

Submitting a draft from Neovim to the agent pane can truncate the message in
the middle. The observed payload was only 24 logical lines; some lines wrapped
to five or six rows at 93 columns, so visual height alone should not explain
the loss.
