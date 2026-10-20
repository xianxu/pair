---
id: 000733
status: open
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
github_issue:
---

# couch: notifications are ephemeral: a restart starts with an empty list, dismiss and clear-all remove items, old items age out

## Problem

Couch's "Waiting on you" list keeps every `couch --notify` forever, across restarts: after the 10-20 reboot it showed
11 items going back to 21:03 the night before, all long handled (operator, 10-20, screenshot in ops's session file).
A list that only grows stops being read, which defeats notifications.
