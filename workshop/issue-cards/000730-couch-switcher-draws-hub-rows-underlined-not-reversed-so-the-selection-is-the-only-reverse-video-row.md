---
id: 000730
status: open
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
github_issue:
---

# couch: switcher draws hub rows underlined, not reversed, so the selection is the only reverse-video row

## Problem

In Couch's thread switcher, a hub row (ops, ops-echo) is drawn in reverse video, the same treatment as the
selected row, so with two hubs and a selection the operator sees three highlighted rows and can't tell at a glance
which one is selected (operator, 10-20, screenshot in ops's session file).
