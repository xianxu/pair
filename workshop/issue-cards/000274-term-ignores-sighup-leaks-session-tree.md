---
id: 000274
status: open
created: 2026-09-16
updated: 2026-09-16
estimate_hours:
github_issue:
---

# `pair term` ignores SIGHUP, so a dead session leaks its whole process tree forever

## Problem

When a zellij session ends abnormally, its panes do not die. `pair term` and
everything under it — including the embedded Neovim — survive indefinitely,
reparented to init, still holding their pty.

Measured on the operator's machine, 2026-09-16:

- **106 orphaned `pair term` trees** (`ppid=1`), plus **29 orphaned `nvim`**
  holding **16 GB each at the top end**.
- Machine state before cleanup: **95 GB PhysMem used, 54 GB in the compressor,
  69 MB unused**. After killing the orphans: **18 GB used, 3 GB compressor,
  77 GB unused**. The leak was ~77 GB.
- Oldest survivor started **Sep 6** — ten days of accumulation.
- **Measure with `top`, not `ps -o rss`.** These processes keep nearly all
  their pages in the compressor, so RSS reports a few hundred MB for a process
  whose real footprint is 16 GB. An RSS-based estimate under-reported this leak
  by more than an order of magnitude.

This was found while diagnosing a machine-wide memory shortage: a build step
in a sibling repo was killed by the OS for lack of memory, with ~73 MB of free
pages and 53 GB in the compressor.
