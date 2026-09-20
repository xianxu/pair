---
id: 000297
status: open
deps: [pair#296]
github_issue:
created: 2026-09-20
updated: 2026-09-20
estimate_hours:
---

# Alt+Shift+Return toggles the right pane between 50/50 and fullscreen

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

## Spec

With the right pane focused, `Alt+Shift+Return` toggles that pane between the
layout's resting 50/50 tiling and zellij fullscreen. **Two states, one press each
way.** The ~2/3 rung is retired — operator decision, 2026-09-20: a true toggle is
worth more than a middle width, and the draft ladder (`Alt+Up`/`Alt+Down`) already
covers partial re-tiling.

- `ActionToggleFocusedLayout` and the `handleTerminalChord` seam keep their
  current shape (`termcmd/run.go:631`); only what `layoutcmd.RunToggleFocused`
  does changes.
- Use `toggle-fullscreen`, not `toggle-no-ui-fullscreen`. The operator's goal is
  columns; zellij's UI bars cost rows, not columns, so the no-UI variant buys
  nothing here and costs the status bar. Say this at the call site so the next
  reader doesn't "upgrade" it.
- Prefer `--pane-id` with the right terminal's known ID over relying on ambient
  focus. Pair already resolves it (`currentRightTerminalPane`,
  `rt.TerminalPaneIDs()`), and an explicit target keeps the action from acting on
  whatever zellij thinks is focused if the two ever disagree.
- Unchanged: the chord is a no-op when a right terminal is not focused
  (`layoutcmd.go:251-254`). This issue does not make it global.

**This is mostly a deletion.** `terminalToggleBurst`, `terminalToggleSteps`, the
60%-of-screen expanded/collapsed classification and `resizeplan_test.go` all go.
`zellij action toggle-fullscreen` is itself a toggle, so Pair no longer has to
infer which direction to move — today's width-threshold state machine exists only
because `resize` is directional. Removing it removes Pair's dependence on
zellij's resize step size entirely. Do not replace it with a fullscreen-state
detector unless something below proves one is needed (`zellijpane.Pane` parses no
`IsFullscreen` field today; check whether `list-panes --json` even carries one
before designing around it). ARCH-DRY / Simplicity First.

### Unknowns to settle live before the design is fixed

These are zellij behaviours, not Pair decisions, and each one changes the spec if
it goes the wrong way. Establish them in a live session and record the answers in
`## Log`:

1. **Does fullscreen follow focus?** `Alt+k` is the keyboard escape back to the
   left stack (`shortcut.go:390-397`). If moving focus while fullscreen makes the
   *newly focused* pane fullscreen, `Alt+k` from a fullscreen terminal traps the
   operator in a fullscreen draft. If instead focus movement exits fullscreen,
   the behaviour is fine as-is. If it traps, Pair must exit fullscreen before
   honouring `Alt+k`, and that becomes a Done-when.
2. **What do the swap-layout rungs do to a fullscreen pane?** `Alt+Up`/`Alt+Down`
   step the draft ladder through zellij swap layouts that re-tile existing panes
   (`main-3.kdl:27-32`). Whether a swap forces an exit from fullscreen, is
   refused, or corrupts the rung state is unknown.
3. **What happens under the `Alt+Shift+D` split?** Fullscreen is per-pane, so
   fullscreening one half should hide the other. Confirm that, and that `Alt+k`'s
   last-used-half memory (`RecordLastTerminalPaneID`) survives the round trip.
4. **Does `pair term` re-render correctly across the geometry jump?** The pane
   goes from ~93 to 186 columns in one step. The reserved tab-strip row and the
   presenter's geometry epoch already handle rung changes, so this should be
   free — but it is the largest single resize Pair will have made, and #223's
   scroll-region history says geometry edges are where this breaks.

### Relationship to #296

`Alt+Shift+Return` is currently forwarded to a full-screen child by #227's
passthrough, so under carbonyl — a TUI that owns the screen — this chord will not
reach Pair at all. **#296 is a hard dependency for the carbonyl use case**, not a
nice-to-have: without it the operator can enter fullscreen from a shell and then
has no way back out once carbonyl is running.

## Done when

- With the right pane focused at 50/50, `Alt+Shift+Return` makes it fill the
  whole tab; pressing it again restores the previous tiling with the agent and
  draft panes back in place.
- Round-tripping leaves the workbench in exactly the layout it started in —
  same rung, same pane sizes, same focus, no process restarts.
- The right pane measures full screen width at fullscreen (186 columns on the
  operator's machine; assert against the screen width, not a literal).
- `Alt+k` still escapes to the left stack from a fullscreen right pane, and does
  not leave a fullscreen draft behind (per unknown 1 — if zellij's behaviour
  forces an explicit exit, Pair does it).
- `Alt+Up`/`Alt+Down` behave sanely while fullscreen (per unknown 2 — either they
  work, or Pair exits fullscreen first; a corrupted rung ladder is a fail).
- Under an `Alt+Shift+D` split, fullscreening one half hides the other and
  `Alt+k`'s last-used-half memory survives the round trip.
- `terminalToggleBurst`, `terminalToggleSteps` and the 60% classification are
  gone, along with `resizeplan_test.go`; no code depends on zellij's resize step
  size any more.
- A test at the `layoutcmd` seam asserts the emitted zellij action for a focused
  right terminal and the no-op for every other focus, without a live zellij.
- README's `Alt+Shift+Return` row (`README.md:123`) and `pair keys` / `Alt+h`
  help describe the new two-state toggle; CHANGELOG entry written.
- `atlas/` updated if the layout vocabulary changes ("expanded" / "collapsed" no
  longer describe two widths).
- Operator smoke-tests it live in `~/workspace/pair`: toggle in and out with a
  shell, with nvim, and — once #292 lands — with carbonyl.

## Plan

- [ ] Settle the four live unknowns in a real session; record answers in `## Log`
      before writing code. Any that goes the wrong way amends the Spec.
- [ ] Replace `RunToggleFocused`'s burst with a single targeted
      `toggle-fullscreen`; delete `resizeplan.go` and its tests.
- [ ] Handle whatever unknowns 1–2 demand (exit-before-focus-move / before-rung).
- [ ] Seam test for the emitted action + the non-terminal-focus no-op.
- [ ] README row, `Alt+h` help, CHANGELOG, atlas vocabulary.
- [ ] `make test`, then operator smoke test before closing.

## Log

### 2026-09-20

Came out of a question about reviving the floating right pane to get a wider
terminal for carbonyl (#292). Floating turned out to be the wrong lever: the
#123 blocker was a plain left-drag on the pane frame (no modifier involved), and
zellij 0.45.1 still ships no gate for it — checked `setup --dump-config` directly
rather than assuming the version bump had helped.

`toggle-fullscreen` reaches the goal without any of that exposure and deletes
more than it adds. Operator chose the two-state toggle (50/50 ↔ full) over
keeping ~2/3 as a middle rung: one press each way is worth more than the
intermediate width.
