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

#125 narrowed the automatic quote-paste to a SOURCE-PANE GATE: selections in
the agent pane still land in the draft as a `> `-prefixed reflow, but
selections in the right terminal only reach the clipboard (that case was
distracting). So the capability is alive and has a production caller — this
issue is no longer "restore the lost feature".

What is still missing is a way to invoke it **on demand**, independent of
making a selection. `PairPasteQuote` (`nvim/init.lua:1541`) is reachable only
via the copy-on-select hand-off; the insert-mode `<C-_>` keymap
(`init.lua:3806`) is that hand-off's delivery gate, not a user-facing binding,
and Alt+n is `PairConfirmRestart`. So there is no way to say "take what is on
my clipboard right now and quote it into the draft" — including for a
right-terminal selection you DID want quoted, which is the natural escape
hatch from #125's gate.

## Spec

- A keybind in the draft pane pastes the CURRENT OS clipboard through the
  existing quote formatting, on demand only. Nothing automatic.
- It reuses `PairPasteQuote` and the `pair clip clipboard-to-pane` staging
  path as-is — this is a binding, not a rewrite. #125 kept that machinery
  wired for the agent-pane path, so nothing has to be revived first.
- Cursor-position semantics are unchanged (col 0 → quote block; col > 0 →
  inline stitch), since that logic already lives in `PairPasteQuote`.
- Pick a chord that does not collide with the workbench registry
  (`cmd/internal/workbenchshortcut`) or nvim defaults; it is draft-local, so
  it does not need to be a global forwarded chord.

## Done when

- Pressing the chord in the draft inserts the clipboard as a formatted quote.
- No selection anywhere triggers it.
- A right-terminal selection — which #125 deliberately stops auto-pasting —
  can still be quoted into the draft deliberately, giving that gate an escape
  hatch rather than a dead end.

## Plan

- [ ]

## Log

### 2026-07-28
- Filed as the named claimant for the machinery #125 leaves in place
  (plan-quality gate on #125 required this rather than a "preserved for
  later" paragraph).
- 2026-07-28 reframed after #125's scope correction: #125 kept the agent-pane
  path, so this is no longer the only surviving trigger. It is now purely an
  on-demand affordance — most useful as the escape hatch for right-terminal
  selections that #125 intentionally stops auto-pasting. Priority drops
  accordingly.
