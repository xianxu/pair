---
id: 000293
status: open
created: 2026-09-19
updated: 2026-09-19
estimate_hours:
github_issue:
---

# Alt+click opens a link in the right pane (browser tab or nvim)

## Problem

Coding agents' links are clickable in a plain terminal but seemed dead under
pair and couch. Diagnosis (2026-09-19), from recording what zellij 0.45.1 sends
to the terminal:

- **zellij passes hyperlinks through intact:** `ESC]8;;https://example.com …`.
- **zellij also turns plain URLs into links itself:** `ESC]8;id=1;https://…`.
- **zellij turns on full mouse reporting:** `?1000h ?1002h ?1003h ?1006h`. From
  then on Ghostty hands clicks to the app instead of opening links, and the
  mouse protocol can't carry Cmd. So Cmd+click reaches zellij as a plain click
  and does nothing.

**Shift+Cmd+click works** in zellij and under couch (confirmed by the operator).
Ghostty keeps every Shift-click for itself (`mouse-shift-capture = false`), so
it opens the link with the system default app. That settles "system default"
with no code.

What's missing is the *pair* destination: opening a link in couch's right pane
instead.
