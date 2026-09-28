---
id: '000312'
status: done
started: 2026-09-23T19:34:09-07:00
created: 2026-09-23
updated: 2026-09-23
actual_hours: 0.11
---

# Keep workspace setup output out of the Couch terminal UI

## Problem

First-time pair:1 setup printed Homebrew/weave output over the active Couch switcher. The CLI assigns raw stderr to WorkspaceProgress even when a console owns the terminal.
