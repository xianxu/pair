---
id: '000073'
status: done
started: 2026-06-29T13:25:37-07:00
created: 2026-06-26
updated: 2026-06-29
estimate_hours: 2.56
actual_hours: 0.60
---

# pair Go migration inventory

## Problem

The Go consolidation needs a factual contract before code moves. Pair currently installs a mix of Go binaries, shell scripts, Lua files, zellij KDL assets, and helper libraries. Without an inventory, later migration issues risk either under-delivering the single-primary-binary target or breaking a hidden caller.
