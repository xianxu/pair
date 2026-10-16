---
id: 000604
status: open
created: 2026-10-15
updated: 2026-10-16
estimate_hours:
github_issue:
started: 2026-10-16T15:06:21-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
---

# Couch switcher: Ctrl+Y 'copy last notice' never reaches the clipboard (OSC 52 only); use the system clipboard locally

## Problem

The operator (2026-10-15): "the ctrl+y to copy last notice in switcher never worked for me." Couch's console
(`couchtty/console_menu.go` `copyLastNotice`) copies through the terminal's OSC 52 clipboard escape
(`ESC ]52;c;<base64> ESC \`) and then says "Full notice copy requested; your terminal may not support it." On the
operator's setup (Ghostty, with Couch's console possibly behind a multiplexer layer) the clipboard never receives it, and
the UI can't tell. Tests only assert the escape bytes were written, never that a clipboard got them.
