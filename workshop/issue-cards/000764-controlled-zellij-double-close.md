---
id: 000764
status: working
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
github_issue:
started: 2026-10-21T03:15:27-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
---

# Controlled Zellij fixture closes its PTY twice and hides cleanup errors

## Problem

ControlledZellij.KillClient closes its PTY but retains the non-nil pointer. ControlledZellij.Close then joins a second Close error, even after exact server cleanup succeeds. TestStartupManagedAttachDetachTimingLive discards that cleanup error. A #761 scratch acceptance overlay surfaced close /dev/ptmx: file already closed after successful attach/detach and reattach/detach; exact server generation was absent and private roots removed.
