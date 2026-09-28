---
id: 000241
status: open
created: 2026-09-12
updated: 2026-09-12
estimate_hours:
github_issue:
---

# pair term: reconcile ALL pane DEC private modes across ALL takeover paths, not just mouse on switch/close

## Problem

#240 made `pair term` reconcile the active tab's MOUSE modes to the pane on a
takeover, fixing the reported "can't select in the shell" bug. Its close
review (BR-3) measured, by widening the probe's regex to every DEC private
mode, that OTHER modes still leak across a tab switch:

- switching from an nvim tab to a shell tab writes only `?1002l ?1006l`,
  leaving nvim's `?1004h` (focus in/out reporting) on the pane, so zellij
  then forwards focus events to the shell that never asked for them;
- `?1h` (DECCKM, cursor-key application mode) and `?2004h` (bracketed paste)
  likewise stay set.

And (BR-1) the mouse reconcile itself is skipped on one takeover path:
`removeTab` with a rename open takes the `paintStripInline` branch
(`run.go:1471-1486`) instead of `applyTakeover`, so a closed child's modes
outlive it until the next real switch.

The general shape #240 named but did not finish: **the pane is a proxy for
its active child, so ALL of the child's terminal modes must follow the active
child across EVERY takeover.** #240 did one mode family on two paths.
