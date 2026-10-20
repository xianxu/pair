---
id: 000731
status: open
deps: [pair#727]
github_issue:
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'c9052a900ff0764d291901f8f6ae9870c8520b5a' # card fields mirrored from issue-cards; edit via sdlc
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

## Spec

Measure the native metadata across a wrapper-to-shell exec in a disposable session, then choose a bounded lifetime/role fix. A plain native fallback shell must not be treated as a Pair-aware terminal for global shortcut delivery. Keep the working shell even when bookkeeping fails. Preserve other panes and unknown registry records; avoid a new native settings resolver or heuristic expansion.

## Done when

- A disposable native regression captures the PID, initial command, title and actual pane_command before/after the handoff and proves the selected classification behavior.
- After fallback, LiveIDs and RoleForPaneWith no longer grant that pane a Pair-terminal role; ordinary active Pair terminals still work.
- Another pane's registration survives, including concurrent registration/retirement if the chosen design mutates that registry; unknown records remain intact.
- A bookkeeping failure is visible and leaves the native shell usable.

## Plan

- [ ]

## Log

Deferred from pair#727 by the hub on 2026-10-20 (delivery 7225aa16-eefc-46c0-97b6-49c250d1ca29): land the reviewed resilience improvement, which leaves a working shell instead of a dead pane. No registry/role change belongs in #727. The earlier retirement-only proposal was withdrawn before implementation when the role and lock assumptions proved incomplete.
