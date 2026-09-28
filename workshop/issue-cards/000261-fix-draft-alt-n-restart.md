---
id: 000261
status: open
created: 2026-09-15
updated: 2026-09-15
estimate_hours:
github_issue:
---

# Fix draft Alt+N restart confirmation

## Problem

During #255 smoke testing, `Alt+N` in the draft Neovim pane opened the restart
confirmation, but confirming it had no visible effect. Restarting from the
switcher menu still works; `Alt+Shift+N` remains the separate new-context
restart command.
