---
id: '000095'
status: done
started: 2026-07-03T14:17:31-07:00
created: 2026-07-01
updated: 2026-07-04
estimate_hours: 2.4
actual_hours: N/A
---

# native nvim and zellij startup assets

## Problem

After #94, the embedded runtime bundle carries only the native assets —
`nvim/*.lua` and `zellij/*.kdl` — which are still extracted to
`$PAIR_DATA_DIR/runtime/<digest>/pair-home` on a copied binary's first run.
That extraction step is the last thing standing between Pair and a *true* native
single binary: an executable that provisions its runtime without writing a
Pair-owned tree to disk. Whether extraction is even removable is an open
question — `nvim` and `zellij` are external processes that read config from
files/paths — so this step is a decision as much as an implementation.
