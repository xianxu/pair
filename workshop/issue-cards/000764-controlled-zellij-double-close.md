---
id: 000764
status: open
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
github_issue:
---

# Controlled Zellij fixture closes its PTY twice and hides cleanup errors

## Problem

ControlledZellij.KillClient closes its PTY but retains the non-nil pointer. ControlledZellij.Close then joins a second Close error, even after exact server cleanup succeeds. TestStartupManagedAttachDetachTimingLive discards that cleanup error. A #761 scratch acceptance overlay surfaced close /dev/ptmx: file already closed after successful attach/detach and reattach/detach; exact server generation was absent and private roots removed.
