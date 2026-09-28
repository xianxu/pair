---
id: 000285
status: open
created: 2026-09-18
updated: 2026-09-18
estimate_hours:
github_issue:
---

# Remember a thread's right-pane tabs across cold starts

## Problem

The right pane's tabs (`pair term`) die with every cold start of a thread:
Alt+x / Alt+n / Shift+Alt+N restart, Couch park→resume and continuation, the
layout-conflict relaunch, a reboot or zellij server death. The operator then
rebuilds the same tabs by hand — the names, the directories, the server / test /
nvim command each one was running. Nothing about tabs is persisted today: the
only per-tag right-pane files are the pane-id/pid shortcut records
(`terminal-panes-<tag>`, `last-terminal-pane-<tag>`).
