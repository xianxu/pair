---
id: 000126
status: open
created: 2026-07-28
updated: 2026-07-28
estimate_hours:
github_issue:
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
