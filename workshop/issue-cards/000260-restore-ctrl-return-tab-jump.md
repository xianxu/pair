---
id: 000260
status: open
created: 2026-09-15
updated: 2026-09-15
estimate_hours:
github_issue:
---

# Investigate Ctrl+Return notification-tab regression

## Problem

`pair#251` already fixed `Ctrl+Return` notification jumps and closed with
focused tests plus operator smoke confirmation. During later operator smoke
testing after #255, the same behavior appeared broken again: `Ctrl+Return` no
longer jumps to the right-pane tab with a pending notification. This issue is a
regression investigation, not a second implementation of #251.
