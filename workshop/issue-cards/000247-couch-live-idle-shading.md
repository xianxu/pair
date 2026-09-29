---
id: 000247
status: done
created: 2026-09-13
updated: 2026-09-29
estimate_hours: 4.35
github_issue:
started: 2026-09-28T21:31:56-07:00
actual_hours: 1.81
tracker:
    version: 1
    completion:
        token: close-2ace06a59323
        repository: github.com/xianxu/pair
        reviewed_head: 9011130807b360cf82fa2bc9408f3f78b426d761
        evidence_commit: 93994d3d2ab88c4a2d08244cfa85a8b9a328b3ec
        landed_commit: 3f71ea6d3626f013d052c9e6a79f4830dad7c08c
---

# Shade live Couch threads by idle time

## Problem

Live Couch threads all use normal foreground even when some have seen no action
for hours or days. The operator wants active threads to stand out and idle
threads to recede through shades of gray, in both the tab/status bar and the
switcher.
