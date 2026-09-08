---
id: 000217
status: open
deps: []
github_issue:
created: 2026-09-08
updated: 2026-09-08
estimate_hours:
---

# dim the right pane's tab strip when the pane loses focus

## Problem

The tab strip (`#199`) renders identically whether the right pane has focus or
not, so it competes for attention while the operator is typing in the draft
pane. The operator wants it muted when unfocused, the way the draft's statusline
reads as secondary information rather than a loud bar.

## Spec

**The strip renders muted when the pane does not have focus.**

### The mechanism is settled

`pair term` learns about focus the same way nvim does: **focus reporting**,
`DECSET 1004`. The terminal then sends `ESC [ I` on focus-in and `ESC [ O` on
focus-out, which `pumpStdin` reads alongside the chords it already parses.

Three things say this works here rather than being hopeful:

- nvim's `FocusLost`/`FocusGained` autocmds already drive real behaviour in the
  draft pane (`init.lua:434,443`), and those are exactly this mode.
- `wrapcmd` already knows the sequences by name (`wrap.go:798-799`), stripping
  them for agents that mishandle them.
- `hostty.ResetInteractiveModes` already includes `1004` in teardown.

Rendering is then pure: `StripModel` gains a `Focused bool`, and `RenderStrip`
emits the muted styling when false. That keeps every hard case a unit test, the
way the rest of the strip is.

### The hazard to design against, named up front

**A child that disables `1004` turns our focus reporting off with no signal** —
which is precisely the shape of `BR-16` in `#172` (a child writing `DECRST
?1000l` silently killed couch's clicks) and the mouse-mode family this repo has
gotten wrong four times. `ptychild.Screen` already scans DECSET/DECRST for
`1049/1047/47` and `1000/1002/1003`; `1004` joins that scan, and the strip must
**degrade to the focused (undimmed) rendering when focus state is unknown**
rather than assume unfocused. Unknown is not a synonym for either state, and a
permanently-dim strip is worse than one that never dims.

## The visual, decided

**A different COLOUR, not `SGR 2` (faint), and it must read correctly on both a
dark and a light scheme.** Operator decision, 2026-09-08.

Faint was rejected for the reason this repo keeps paying for elsewhere: a
terminal that ignores `SGR 2` renders nothing different, **silently**. A colour
either shows or is visibly wrong.

### The both-schemes requirement is new here, and the house convention fails it

`couchtty/menu_render.go` already has a muted-grey convention — `38;5;238` /
`240` / `245` / `250` as an age ramp — but those are **fixed** greys chosen
against a dark background. On a light scheme that ramp *inverts*: `238` (a dark
grey) becomes MORE prominent than `250` (a light grey), so "oldest" would read
as the loudest row. Copying it here would inherit that bug.

### Proposed: ANSI 90 (bright black), the one grey the terminal themes itself

`\x1b[90m` is not an absolute colour — terminals map the eight bright ANSI
slots per theme, so it lands as "dimmer than the default foreground" on both a
dark and a light background. That is exactly the semantic wanted, and it is the
only widely-supported way to get it without querying the background colour
(OSC 11) and picking a ramp, which is a round-trip and fragile.

Residual risk, to settle in the manual step rather than by argument: some themes
render bright-black at very low contrast against their own background. If the
operator's light scheme is one of them, the fallback is an OSC 11 query with a
grey chosen from the answer — more machinery, so only if measurement demands it.

**The active tab keeps its brackets in both states.** Dimming must not become
the only way to tell which tab is active, for the same reason `#199` made the
marker text rather than colour: a mono terminal, `ansi.Strip` in a log, and a
colour-blind reader all lose colour and none of them lose brackets.

## Plan

- [x] Pin the visual treatment with the operator — a colour, not `SGR 2`; both schemes must work.
- [ ] `Screen`: scan `1004` alongside the existing DECSET tracking; expose focus
      state as **known-focused / known-unfocused / unknown**, not a bool.
- [ ] `pumpStdin`: consume `ESC [ I` / `ESC [ O` without passing them to the
      child, the way it already consumes chords.
- [ ] `StripModel.Focused`; `RenderStrip` renders muted when unfocused and
      normal when unknown. Pure tests, at least two tabs per case.
- [ ] Enable `1004` on startup and reset it on teardown.
- [ ] Manual, **in BOTH a dark and a light scheme** — this is the acceptance,
      and the half a unit test cannot reach: focus the draft, confirm the strip
      mutes and the active tab is still identifiable; focus back, confirm it
      returns. A test can assert that the unfocused rendering DIFFERS and that
      the strip resets SGR afterwards; it cannot assert that the result is
      legible.

## Done when

- The strip is muted when the right pane is unfocused and normal when focused.
- A child disabling `1004` leaves the strip in the FOCUSED rendering, never
  stuck dim.
- The active tab is identifiable in both states without relying on colour.
- The muted rendering reads as muted on a dark scheme AND on a light one.

## Log

- 2026-09-08 — filed from the operator's request during `#199`'s close. Kept out
  of `#199` deliberately: that issue was mid-close and this is a new surface,
  not a defect in the strip.
