---
id: 000622
status: open
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
github_issue:
---

# Couch server spends ~1 core emulating every hosted client's output in a vt parser, visible or not

## Problem

Profiled by pair:2 for pair#618 (2026-10-16, macOS `sample` of the Couch server, 5 s): the Couch server burns about
one core continuously, and ~500 of its busy samples are terminal emulation: `ptychild.(*Child).ingest` →
`internal/terminal` + charmbracelet's vt `ansiparser`. Couch parses the full output stream of every hosted session
client (~27 `pair --couch-session-v1 resume` children) in a vt emulator, so every spinner redraw on every slot costs
Couch CPU all the time, whether or not anyone is looking at that slot.
