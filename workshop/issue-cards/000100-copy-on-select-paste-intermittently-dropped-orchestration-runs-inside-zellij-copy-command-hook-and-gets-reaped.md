---
id: '000100'
status: done
started: 2026-07-05T10:48:20-07:00
created: 2026-07-05
updated: 2026-07-05
estimate_hours: 1.16
actual_hours: 0.81
---

# copy-on-select paste intermittently dropped — orchestration runs inside zellij copy_command hook and gets reaped

## Problem

Copy-on-select → paste into the nvim draft pane intermittently drops the text.
The source pane flashes green (copy half succeeds) but nothing is inserted.
Worse since the Go migration (#93 M4); operator reports the **first copy after a
pair restart fails, subsequent copies work**.
