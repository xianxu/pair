---
id: 000126
status: open
deps: [125]
github_issue:
created: 2026-07-28
updated: 2026-07-28
estimate_hours:
---

# Deliberate quote-paste keybind in the draft

## Problem

#125 removes the AUTOMATIC quote-paste (selecting text anywhere outside the
draft flashed the pane, stole focus, and inserted a `> `-prefixed reflow) —
it was distracting. But the *formatting* it produced is genuinely useful:
`par` reflow, `> ` prefix at column 0, inline stitching mid-line, flash, and
land in insert mode below.

After #125 there is no way to invoke that at all. `PairPasteQuote`
(`nvim/init.lua:1541`) has exactly one caller — the copy-on-select hand-off —
and the insert-mode `<C-_>` keymap (`init.lua:3806`) exists solely as that
hand-off's delivery gate. Alt+n is `PairConfirmRestart`, not quote-paste. So
the capability is dormant, not available on demand.

This issue gives it a deliberate trigger, so the machinery #125 deliberately
left unwired has a claimant rather than being orphaned.

## Spec

- A keybind in the draft pane pastes the CURRENT OS clipboard through the
  existing quote formatting, on demand only. Nothing automatic.
- It reuses `PairPasteQuote` and the `pair clip clipboard-to-pane` staging
  path as-is — this is a binding, not a rewrite. That is precisely why #125
  unwired rather than deleted them.
- Cursor-position semantics are unchanged (col 0 → quote block; col > 0 →
  inline stitch), since that logic already lives in `PairPasteQuote`.
- Pick a chord that does not collide with the workbench registry
  (`cmd/internal/workbenchshortcut`) or nvim defaults; it is draft-local, so
  it does not need to be a global forwarded chord.

## Done when

- Pressing the chord in the draft inserts the clipboard as a formatted quote.
- No selection anywhere triggers it.
- The previously-unwired pipeline has a production caller again, so the
  "unwired since #125" atlas notes can be retired.

## Plan

- [ ]

## Log

### 2026-07-28
- Filed as the named claimant for the machinery #125 leaves in place
  (plan-quality gate on #125 required this rather than a "preserved for
  later" paragraph).
