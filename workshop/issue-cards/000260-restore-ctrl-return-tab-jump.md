---
id: 000260
status: codecomplete
created: 2026-09-15
updated: 2026-09-28
estimate_hours:
github_issue:
started: 2026-09-28T09:47:22-07:00
actual_hours: 0.27
tracker:
    version: 1
    completion:
        token: close-f4e8bb77cbbc
        repository: github.com/xianxu/pair
        reviewed_head: 98255d69330b31e233fbfd6eda9e6c884254605e
        evidence_commit: 9ab230a018170b5dea0a711497191496eaf55804
---

# Investigate Ctrl+Return notification-tab regression

## Problem

`pair#251` already fixed `Ctrl+Return` notification jumps and closed with
focused tests plus operator smoke confirmation. During later operator smoke
testing after #255, the same behavior appeared broken again: `Ctrl+Return` no
longer jumps to the right-pane tab with a pending notification. This issue is a
regression investigation, not a second implementation of #251.
