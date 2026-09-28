---
id: '000297'
status: done
started: 2026-09-20T09:53:35-07:00
created: 2026-09-20
updated: 2026-09-20
estimate_hours: 2.48
actual_hours: 3.65
---

# Alt+Shift+Return globally toggles the right pane between 50/50 and fullscreen, restoring focus

## Problem

`Alt+Shift+Return` with the right pane focused re-tiles the terminal column
between half the screen and about two thirds. Two thirds is the ceiling, and it
is reached by a blind three-step resize burst calibrated to zellij's 5%-per-step
`resize increase left` (`layoutcmd/resizeplan.go:15-18`, #124).

The operator wants the right pane maximised — a full-width terminal on a
186-column screen — in preparation for the carbonyl browser tab (#292), where a
rendered page wants every column it can get. The current ladder cannot express
that, and widening it by adding resize steps would keep inheriting the burst's
fragility: the step size is a zellij constant Pair re-derives by measurement, and
`resizeplan.go` already carries the caveat that a future zellij step change
"degrades to a different stable pair of widths".

The chord is also pane-local, which makes it useless at the moment the operator
actually wants it. The workflow is: typing in the draft → maximise the right pane
and start working in it → come back to the draft and keep typing. Today that
costs an `Alt+k` before and after, and the chord does not even exist outside the
right pane.

zellij has the operation natively:

    $ zellij action toggle-fullscreen --help
    Toggle between fullscreen focus pane and normal layout
      -p, --pane-id <PANE_ID>  Target a specific pane by ID

It is per-*pane*, not per-session: the focused pane fills the tab and the other
panes are hidden. Pair already drives zellij this way — `RunToggleFocused` calls
`rt.RunZellijAction("resize", …)` today, so this is the same seam with a
different verb.

It is also strictly safer than the alternatives considered. A fullscreen pane has
no neighbour boundary to drag and no floating frame, so it carries none of the
mouse-drag exposure that made the floating right terminal untenable in #123
(zellij still has no config gate for frame-drag move — verified against 0.45.1's
`setup --dump-config`; the mouse keys added since 0.44.3 are
`mouse_scroll_resize`, `scroll_mode_sync`, `mouse_hover_tips`,
`osc133_command_selection`, `osc8_hyperlinks`, none of which gates drag).
