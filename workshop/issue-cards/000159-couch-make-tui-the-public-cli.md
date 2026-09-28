---
id: '000159'
status: done
started: 2026-09-01T09:03:30-07:00
created: 2026-09-01
updated: 2026-09-01
estimate_hours: 2.41
actual_hours: 2.03
---

# couch: make TUI the public CLI

## Problem

Couch is a terminal UI, but its executable currently presents every typed
operation as a peer CLI subcommand. Bare `couch` prints that implementation
inventory instead of opening the workspace, so the public interface describes
the internal state machine rather than the product the operator uses.
