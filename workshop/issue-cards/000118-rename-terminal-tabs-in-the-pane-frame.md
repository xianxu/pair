---
id: '000118'
status: done
started: 2026-07-24T17:04:26-07:00
created: 2026-07-24
updated: 2026-07-28
estimate_hours: 2.48
actual_hours: 2.61
---

# Rename terminal tabs in the pane frame

## Problem

Alt+r currently calls a raw prompt rendered into the active child terminal's
content area. It feels acceptable at an idle shell prompt but visually corrupts
or competes with full-screen applications such as Neovim. Zellij's native
rename-tab editor cannot be reused because the right-side tabs are Pair's own
PTY multiplexer tabs inside one Zellij floating pane.
