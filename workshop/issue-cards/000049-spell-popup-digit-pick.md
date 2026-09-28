---
id: '000049'
status: done
created: 2026-06-04
updated: 2026-06-04
estimate_hours: 0.5
actual_hours: 0.5
---

# Spell popup: bare-digit pick + stay in normal mode

## Problem

The `z=` spell-suggestion popup in draft nvim had two UX rough edges:

1. Pressing a number while the popup is shown inserted that digit into the
   buffer instead of selecting the corresponding suggestion. (Picking was only
   on `<M-i>`, mirroring the path/word completers.)
2. `z=` is a normal-mode trigger, but it had to enter insert mode to host the
   completion popup — and after picking a suggestion the user was stranded in
   insert mode rather than returned to the normal mode they started in.
