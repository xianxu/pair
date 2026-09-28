---
id: '000243'
status: done
started: 2026-09-13T10:35:43-07:00
created: 2026-09-13
updated: 2026-09-13
estimate_hours: 1.00
actual_hours: 0.88
---

# regression: from-anywhere M-S-left/right stops switching the right terminal's tab when it shows a full-screen app (#227)

## Problem

Two things, one root:

1. **Regression from #227.** The from-anywhere `M-S-left`/`M-S-right` (switch the
   right terminal's tab from any pane, #216) stopped switching when the right
   pane shows a **full-screen app** (nvim). Confirmed at the pump: the delivery
   sends the ROLE-SCOPED `Alt+Left`/`Alt+Right` bytes to the right terminal
   (`TabChordFor` -> `ChordAltLeft`/`ChordAltRight`), and #227 now passes exactly
   those through to the full-screen child instead of switching:

       Alt+Left  (old delivery) under fullscreen -> ops=[write:ESC[1;3D]  (eaten by nvim)
       Alt+Shift+Left (global)  under fullscreen -> ops=[prev-tab]        (switches)

   The delivery reused the role-scoped chord; #227 made that chord passthrough-
   eligible. The fix is to deliver the GLOBAL chord, which pair term always
   handles and never passes through.

2. **Feature the operator asked for.** Make the from-anywhere right-terminal
   control a set of **three** global chords -- `M-S-left`, `M-S-right`, and a new
   `M-S-t` (create a tab) -- so the operator can drive the right pane from the
   draft or agent even while a full-screen app owns it. `M-S-t` is also the
   deliberate "escape chord" #227 left out: create a tab without leaving nvim.
