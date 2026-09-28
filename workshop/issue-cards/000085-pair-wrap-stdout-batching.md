---
id: '000085'
status: done
started: 2026-06-29T15:14:39-07:00
created: 2026-06-29
updated: 2026-06-29
estimate_hours: 1.42
actual_hours: 0.18
---

# Batch pair-wrap stdout redraws

## Problem

#82 is tracking a Codex scroll wedge where pair-wrap continues reading,
forwarding, and capturing stdout, but zellij can still get into a bad
scroll/render state. Codex produces dense redraw bursts. Today pair-wrap writes
each filtered PTY stdout chunk directly to `os.Stdout`, so zellij may receive a
redraw storm even though pair-wrap's raw capture and tracing remain healthy.
