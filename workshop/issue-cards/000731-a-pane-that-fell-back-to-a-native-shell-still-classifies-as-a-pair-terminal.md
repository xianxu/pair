---
id: 000731
status: done
created: 2026-10-20
updated: 2026-10-21
estimate_hours:
github_issue:
started: 2026-10-20T20:32:00-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
actual_hours: 1.22
tracker:
    version: 1
    completion:
        token: close-3bce1c87e68e
        repository: github.com/xianxu/pair
        reviewed_head: ae6dc590d696c87ed99f948c1d0d4b56c9762f5c
        evidence_commit: f046fc278933cd1b5bdd611e692517f8926ad92a
        landed_commit: 323986ed412b4dd1b885b560c9ba9a71e2132e11
---

# A pane that fell back to a native shell still classifies as a Pair terminal

## Problem

pair#727 improves wrapper failures by restoring the terminal and execing a native interactive shell, so the operator keeps a usable pane. The fallback shell can still be classified as a live Pair terminal.

Source findings in pair#727 (2026-10-20):

- termcmd.OSRuntime.RegisterTerminalPane records the wrapper PID; syscall.Exec preserves that PID.
- workbenchshortcut.TerminalPaneRegistry.LiveIDs checks only PID liveness, so the wrapper registration survives the handoff.
- RoleForPaneWith first calls RoleForPane, which accepts a startup TerminalCommand containing pair term or a title equal to terminal / prefixed terminal. Removing a registry record alone therefore does not prove the pane has lost its Pair role.
- TerminalPaneRegistry currently has no serialization lock: Register is plain append. A lock on the data inode would not serialize an atomic rename. FullscreenReturnStore has a separate stable lock, but borrowing it would couple registration to native fullscreen operations.

No native before/after capture of template, title and pane_command across exec has been made. These are source findings; pair#727's disposable native placement/focus conformance does not establish exec metadata behavior.
