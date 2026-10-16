---
id: 000604
status: open
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
github_issue:
---

# Couch switcher: Ctrl+Y 'copy last notice' never reaches the clipboard (OSC 52 only); use the system clipboard locally

## Problem

The operator (2026-10-15): "the ctrl+y to copy last notice in switcher never worked for me." Couch's console
(`couchtty/console_menu.go` `copyLastNotice`) copies through the terminal's OSC 52 clipboard escape
(`ESC ]52;c;<base64> ESC \`) and then says "Full notice copy requested; your terminal may not support it." On the
operator's setup (Ghostty, with Couch's console possibly behind a multiplexer layer) the clipboard never receives it, and
the UI can't tell. Tests only assert the escape bytes were written, never that a clipboard got them.
