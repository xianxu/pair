---
id: '000125'
status: done
started: 2026-07-28T12:27:14-07:00
created: 2026-07-28
updated: 2026-07-28
estimate_hours: 0.45
actual_hours: 1.43
---

# Stop auto-pasting terminal selections into the draft

## Problem

Selecting text anywhere outside the draft pane currently flashes the source
pane, steals focus to the draft, and inserts the selection as a `> `-prefixed
reflowed quote. The user reports this as distracting: a selection made only to
copy a path or a line hijacks the draft.

The pipeline is zellij's `copy_command "pair clip copy-on-select"` →
`RunCopyOnSelect` (mirror to OS clipboard, spawn detached orchestrator) →
`RunCopyOnSelectOrchestrate` (flash + hand off) → `RunClipboardToPane` (stage
at `quote-<tag>`, focus draft, send Ctrl-_) → nvim `PairPasteQuote`.
