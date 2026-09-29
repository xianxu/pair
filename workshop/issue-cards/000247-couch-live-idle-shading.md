---
id: 000247
status: working
created: 2026-09-13
updated: 2026-09-28
estimate_hours:
github_issue:
started: 2026-09-28T21:31:56-07:00
---

# Shade live Couch threads by idle time

## Problem

Live Couch threads all use normal foreground even when some have seen no action
for hours or days. The operator wants active threads to stand out and idle
threads to recede through shades of gray, in both the tab/status bar and the
switcher.
