---
id: 000731
status: open
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
github_issue:
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
